/*
 * Copyright 2014-2015 Canonical Ltd.
 *
 * Authors:
 * Sergio Schvezov: sergio.schvezov@canonical.com
 * Manuel de la Pena: manuel.delapena@canonical.com
 *
 * ciborium is free software; you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation; version 3.
 *
 * nuntium is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package udisks2

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"log"

	"launchpad.net/go-dbus/v1"
)

const (
	dbusName                    = "org.freedesktop.UDisks2"
	dbusObject                  = "/org/freedesktop/UDisks2"
	dbusObjectManagerInterface  = "org.freedesktop.DBus.ObjectManager"
	dbusBlockInterface          = "org.freedesktop.UDisks2.Block"
	dbusDriveInterface          = "org.freedesktop.UDisks2.Drive"
	dbusFilesystemInterface     = "org.freedesktop.UDisks2.Filesystem"
	dbusPartitionInterface      = "org.freedesktop.UDisks2.Partition"
	dbusPartitionTableInterface = "org.freedesktop.UDisks2.PartitionTable"
	dbusJobInterface            = "org.freedesktop.UDisks2.Job"
	dbusPropertiesInterface     = "org.freedesktop.DBus.Properties"
	dbusAddedSignal             = "InterfacesAdded"
	dbusRemovedSignal           = "InterfacesRemoved"
	defaultMaximumWaitTime      = 64
)

type MountEvent struct {
	Path       dbus.ObjectPath
	Mountpoint string
}

type driveMap map[dbus.ObjectPath]*Drive

type mountpointMap map[dbus.ObjectPath]string

type UDisks2 struct {
	conn            *dbus.Connection
	blockAdded      chan *Event
	driveAdded      *dbus.SignalWatch
	mountRemoved    chan string
	blockError      chan error
	driveRemoved    *dbus.SignalWatch
	blockDevice     chan bool
	drives          driveMap
	mountpoints     mountpointMap
	mapLock         sync.Mutex
	startLock       sync.Mutex
	dispatcher      *dispatcher
	jobs            *jobManager
	pendingMounts   []string
	umountCompleted chan string
	unmountErrors   chan error
	mountCompleted  chan MountEvent
	mountErrors     chan error
}

func NewStorageWatcher(conn *dbus.Connection) (u *UDisks2) {
	u = &UDisks2{
		conn:          conn,
		drives:        make(driveMap),
		mountpoints:   make(mountpointMap),
		pendingMounts: make([]string, 0, 0),
	}
	runtime.SetFinalizer(u, cleanDriveWatch)
	return u
}

func (u *UDisks2) SubscribeAddEvents() (<-chan *Event, <-chan error) {
	u.blockAdded = make(chan *Event)
	u.blockError = make(chan error)
	return u.blockAdded, u.blockError
}

func (u *UDisks2) SubscribeRemoveEvents() <-chan string {
	u.mountRemoved = make(chan string)
	return u.mountRemoved
}

func (u *UDisks2) SubscribeBlockDeviceEvents() <-chan bool {
	u.blockDevice = make(chan bool)
	return u.blockDevice
}

func (u *UDisks2) SubscribeUnmountEvents() (<-chan string, <-chan error) {
	u.umountCompleted = make(chan string)
	u.unmountErrors = make(chan error)
	return u.umountCompleted, u.unmountErrors
}

func (u *UDisks2) SubscribeMountEvents() (<-chan MountEvent, <-chan error) {
	u.mountCompleted = make(chan MountEvent)
	u.mountErrors = make(chan error)
	return u.mountCompleted, u.mountErrors
}

func (u *UDisks2) Mount(s *Event) {
	go func() {
		var mountpoint string
		obj := u.conn.Object(dbusName, s.Path)
		options := make(VariantMap)
		options["auth.no_user_interaction"] = dbus.Variant{true}
		reply, err := obj.Call(dbusFilesystemInterface, "Mount", options)
		if err != nil {
			u.mountErrors <- err
			return
		}
		if err := reply.Args(&mountpoint); err != nil {
			u.mountErrors <- err
		}

		log.Println("Mounth path for '", s.Path, "' set to be", mountpoint)
	}()
}

func (u *UDisks2) Unmount(d *Drive) {
	if d.Mounted {
		for blockPath, _ := range d.BlockDevices {
			u.umount(blockPath)
		}
	} else {
		log.Println("Block is not mounted", d)
		u.unmountErrors <- fmt.Errorf("Drive is not mounted ", d)
	}
}

func (u *UDisks2) syncUmount(o dbus.ObjectPath) error {
	log.Println("Unmounting", o)
	obj := u.conn.Object(dbusName, o)
	options := make(VariantMap)
	options["auth.no_user_interaction"] = dbus.Variant{true}
	_, err := obj.Call(dbusFilesystemInterface, "Unmount", options)
	return err
}

func (u *UDisks2) umount(o dbus.ObjectPath) {
	go func() {
		err := u.syncUmount(o)
		if err != nil {
			u.unmountErrors <- err
		}
	}()
}

func (u *UDisks2) mountpointsForPath(p dbus.ObjectPath) []string {
	var mountpoints []string
	proxy := u.conn.Object(dbusName, p)
	reply, err := proxy.Call(dbusPropertiesInterface, "Get", dbusFilesystemInterface, mountPointsProperty)
	if err != nil {
		log.Println("Error getting mount points")
		return mountpoints
	}
	if reply.Type == dbus.TypeError {
		log.Println("dbus error: %", reply.ErrorName)
		return mountpoints
	}

	mountpointsVar := dbus.Variant{}
	if err = reply.Args(&mountpointsVar); err != nil {
		log.Println("Error reading arg", err)
		return mountpoints
	}

	mountPointsVal := reflect.ValueOf(mountpointsVar.Value)
	length := mountPointsVal.Len()
	mountpoints = make([]string, length, length)
	for i := 0; i < length; i++ {
		array := reflect.ValueOf(mountPointsVal.Index(i).Interface())
		arrayLenght := array.Len()
		byteArray := make([]byte, arrayLenght, arrayLenght)
		for j := 0; j < arrayLenght; j++ {
			byteArray[j] = array.Index(j).Interface().(byte)
		}
		mp := string(byteArray)
		mp = mp[0 : len(mp)-1]
		mountpoints[i] = mp
	}
	return mountpoints
}

func (u *UDisks2) Init() (err error) {
	d, err := newDispatcher(u.conn)
	if err != nil {
		return err
	}

	u.dispatcher = d
	u.jobs = newJobManager(d)
	go func() {
		for {
			select {
			case e := <-u.dispatcher.Additions:
				if err := u.processAddEvent(&e); err != nil {
					log.Print("Issues while processing ", e.Path, ": ", err)
				}
			case e := <-u.dispatcher.Removals:
				if err := u.processRemoveEvent(e.Path, e.Interfaces); err != nil {
					log.Println("Issues while processing remove event:", err)
				}
			case j := <-u.jobs.UnmountJobs:
				if j.WasCompleted {
					log.Println("Unmount job was finished for", j.Event.Path, "for paths", j.Paths)
					for _, path := range j.Paths {
						u.umountCompleted <- path
						log.Println("Removing", path, "from", u.mountpoints)
						delete(u.mountpoints, dbus.ObjectPath(path))
					}
				} else {
					log.Print("Unmount job started.")
				}
			case j := <-u.jobs.MountJobs:
				if j.WasCompleted {
					log.Println("Mount job was finished for", j.Event.Path, "for paths", j.Paths)
					for _, path := range j.Paths {
						// grab the mointpoints from the variant
						mountpoints := u.mountpointsForPath(dbus.ObjectPath(path))
						log.Println("Mount points are", mountpoints)
						if len(mountpoints) > 0 {
							p := dbus.ObjectPath(path)
							mp := string(mountpoints[0])
							u.mountpoints[p] = string(mp)
							// update the drives
							for _, d := range u.drives {
								changed := d.SetMounted(p)
								if changed {
									e := MountEvent{d.Path, mp}
									log.Println("New mount event", e)
									go func() {
										for _, t := range [...]int{1, 2, 3, 4, 5, 10} {
											_, err := os.Stat(mp)
											if err != nil {
												log.Println("Mountpoint", mp, "not yet present. Wating", t, "seconds due to", err)
												time.Sleep(time.Duration(t) * time.Second)
											} else {
												break
											}
										}
										log.Println("Sending new event to channel.")
										u.mountCompleted <- e
									}()
								}
							}
						}

					}
				} else {
					log.Print("Mount job started.")
				}
			}
		}
	}()
	d.Init()
	u.emitExistingDevices()
	return nil
}

func (u *UDisks2) connectToSignal(path dbus.ObjectPath, inter, member string) (*dbus.SignalWatch, error) {
	w, err := u.conn.WatchSignal(&dbus.MatchRule{
		Type:      dbus.TypeSignal,
		Sender:    dbusName,
		Interface: dbusObjectManagerInterface,
		Member:    member,
		Path:      path})
	return w, err
}

func (u *UDisks2) connectToSignalInterfacesAdded() (*dbus.SignalWatch, error) {
	return u.connectToSignal(dbusObject, dbusObjectManagerInterface, dbusAddedSignal)
}

func (u *UDisks2) connectToSignalInterfacesRemoved() (*dbus.SignalWatch, error) {
	return u.connectToSignal(dbusObject, dbusObjectManagerInterface, dbusRemovedSignal)
}

func (u *UDisks2) emitExistingDevices() {
	log.Println("emitExistingDevices")
	u.startLock.Lock()
	defer u.startLock.Unlock()
	obj := u.conn.Object(dbusName, dbusObject)

	// At certain systems it has been observed that although the
	// org.freedesktop.UDisks2 name is registered the ObjectManager
	// interface is not immediately ready. As a result the
	// "GetManagedObjects" call fails and the cold-plugged usb devices
	// are not reported. The code below calls GetManagedObjects several
	// times waiting for it to return successfully. It is assumed that
	// once the org.freedesktop.UDisks2 name is acquired the interface
	// will show up.
	var reply *dbus.Message
	f := fibonacci()
	for i := f(); i < defaultMaximumWaitTime; i = f() {
		var err error
		reply, err = obj.Call(dbusObjectManagerInterface, "GetManagedObjects")
		if err != nil {
			log.Println("Cannot get initial state for devices:", err)
			time.Sleep(time.Duration(i) * time.Second)
		} else {
			break
		}
	}

	allDevices := make(map[dbus.ObjectPath]InterfacesAndProperties)
	if err := reply.Args(&allDevices); err != nil {
		log.Println("Cannot get initial state for devices:", err)
	}

	var blocks, drives []*Event
	// separate drives from blocks to avoid aliasing
	for objectPath, props := range allDevices {
		s := &Event{objectPath, props, make([]string, 0, 0)}
		switch objectPathType(objectPath) {
		case deviceTypeDrive:
			drives = append(drives, s)
		case deviceTypeBlock:
			blocks = append(blocks, s)
		}
	}

	for i := range drives {
		if err := u.processAddEvent(drives[i]); err != nil {
			log.Println("Error while processing events:", err)
		}
	}

	for i := range blocks {
		if err := u.processAddEvent(blocks[i]); err != nil {
			log.Println("Error while processing events:", err)
		}
	}
}

func (u *UDisks2) processAddEvent(s *Event) error {
	u.mapLock.Lock()
	defer u.mapLock.Unlock()

	pos := sort.SearchStrings(u.pendingMounts, string(s.Path))
	if pos != len(u.pendingMounts) && s.Props.isFilesystem() {
		log.Println("Path", s.Path, "must be remounted.")
	}

	if isBlockDevice, err := u.drives.addInterface(s); err != nil {
		return err
	} else if isBlockDevice {
		log.Println("New block device added", s.Path)
		if u.blockAdded != nil && u.blockError != nil {
			if !u.desiredMountableEvent(s) {
				u.blockError <- err
			} else {
				u.blockAdded <- s
			}
		}
		if u.blockDevice != nil {
			log.Println("Sedding block device to channel")
			u.blockDevice <- true
		}
	}

	return nil
}

func (u *UDisks2) processRemoveEvent(objectPath dbus.ObjectPath, interfaces Interfaces) error {
	log.Println("Remove event for", objectPath)
	mountpoint, mounted := u.mountpoints[objectPath]
	if mounted {
		log.Println("Removing mountpoint", mountpoint)
		delete(u.mountpoints, objectPath)
		if u.mountRemoved != nil && interfaces.desiredUnmountEvent() {
			u.mountRemoved <- mountpoint
		} else {
			return errors.New("mounted but does not remove filesystem interface")
		}
	}
	u.mapLock.Lock()
	log.Println("Removing device", objectPath)
	if strings.HasPrefix(string(objectPath), path.Join(dbusObject, "drives")) {
		delete(u.drives, objectPath)
	} else {
		// TODO: remove filesystem interface from map
	}
	u.mapLock.Unlock()
	if u.blockDevice != nil {
		log.Println("Removing block device to channel.")
		u.blockDevice <- false
	}
	return nil
}

func cleanDriveWatch(u *UDisks2) {
	log.Print("Cancelling Interfaces signal watch")
	u.driveAdded.Cancel()
	u.driveRemoved.Cancel()
}

func (iface Interfaces) desiredUnmountEvent() bool {
	for i := range iface {
		fmt.Println(iface[i])
		if iface[i] == dbusFilesystemInterface {
			return true
		}
	}
	return false
}

func isAutomountEnabled() bool {
	_, err := os.Stat(filepath.Join(os.Getenv("SNAP_COMMON"), ".automount_enabled"))
	if err != nil {
		return false
	}
	return true
}

func (u *UDisks2) getDriveFromEvent(e *Event) *Drive {
	path, err := e.getDriveObjectPath()
	if err != nil {
		return nil
	}

	drive, ok := u.drives[path]
	if !ok {
		return nil
	}

	return drive
}

func (u *UDisks2) desiredMountableEvent(s *Event) bool {
	if !isAutomountEnabled() {
		return false
	}

	if s.Props.isBlockIgnored() {
		log.Println(s.Path, "will not be automounted as it is marked to be ignored")
		return false
	}

	if !s.hasInterface(dbusBlockInterface) || !s.hasInterface(dbusFilesystemInterface) {
		log.Println(s.Path, "will not be automounted as it is not a block device or does not have a filesystem")
		return false
	}

	drive := u.getDriveFromEvent(s)
	if drive == nil {
		log.Println(s.Path, "will not be automounted as we can't find the corresponding drive for it")
		return false
	}

	if drive.HasSystemBlockDevices() {
		log.Println(s.Path, "will not be automounted as it's on a system drive")
		return false
	}

	if !drive.IsRemovable() {
		log.Println(s.Path, "will not be automounted as its drive is not removable")
		return false
	}

	if s.Props.isMounted() {
		log.Println(s.Path, "will not be automounted as it is already mounted")
		return false
	}

	if !s.Props.hasIdType() {
		log.Println(s.Path, "will not be automounted as it has no id type set")
		return false
	}

	return true
}

const (
	deviceTypeBlock = iota
	deviceTypeDrive
	deviceTypeUnhandled
)

type dbusObjectPathType uint

func objectPathType(objectPath dbus.ObjectPath) dbusObjectPathType {
	objectPathString := string(objectPath)
	if strings.HasPrefix(objectPathString, path.Join(dbusObject, "drives")) {
		return deviceTypeDrive
	} else if strings.HasPrefix(objectPathString, path.Join(dbusObject, "block_devices")) {
		return deviceTypeBlock
	} else {
		return deviceTypeUnhandled
	}
}

func (dm *driveMap) addInterface(s *Event) (bool, error) {
	var blockDevice bool

	switch objectPathType(s.Path) {
	case deviceTypeDrive:
		if _, ok := (*dm)[s.Path]; ok {
			log.Println("WARNING: replacing", s.Path, "with new drive event")
		}
		(*dm)[s.Path] = NewDrive(s)
	case deviceTypeBlock:
		driveObjectPath, err := s.getDriveObjectPath()
		if err != nil {
			return blockDevice, err
		}
		if _, ok := (*dm)[driveObjectPath]; !ok {
			drive := NewDrive(s)
			//log.Println("Creating new drive", drive)
			(*dm)[s.Path] = drive
		} else {
			(*dm)[driveObjectPath].BlockDevices[s.Path] = s.Props
			(*dm)[driveObjectPath].Mounted = s.Props.isMounted()
		}
		blockDevice = true
	default:
		// we don't care about other object paths
		log.Println("Unhandled object path", s.Path)
	}

	return blockDevice, nil
}
