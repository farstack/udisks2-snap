/*
 * Copyright 2014-2015 Canonical Ltd.
 *
 * Authors:
 * Sergio Schvezov: sergio.schvezov@cannical.com
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
)

type Drive struct {
	Path         dbus.ObjectPath
	blockDevices map[dbus.ObjectPath]InterfacesAndProperties
	driveInfo    InterfacesAndProperties
	Mounted      bool
}

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
	formatCompleted chan *Event
	formatErrors    chan error
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

func (u *UDisks2) SubscribeFormatEvents() (<-chan *Event, <-chan error) {
	u.formatCompleted = make(chan *Event)
	u.formatErrors = make(chan error)
	return u.formatCompleted, u.formatErrors
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
		for blockPath, _ := range d.blockDevices {
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

func (u *UDisks2) syncFormat(o dbus.ObjectPath) error {
	// perform sync call to format the device
	log.Println("Formatting", o)
	obj := u.conn.Object(dbusName, o)
	options := make(VariantMap)
	options["auth.no_user_interaction"] = dbus.Variant{true}
	_, err := obj.Call(dbusBlockInterface, "Format", "vfat", options)
	return err
}

func (u *UDisks2) Format(d *Drive) {
	go func() {
		log.Println("Format", d)
		// do a sync call to unmount
		for blockPath, _ := range d.blockDevices {
			mps := u.mountpointsForPath(blockPath)
			if len(mps) > 0 {
				log.Println("Unmounting", blockPath)
				err := u.syncUmount(blockPath)
				if err != nil {
					log.Println("Error while doing a pre-format unmount:", err)
					u.formatErrors <- err
					return
				}
			}
		}

		// delete all the partitions
		for blockPath, block := range d.blockDevices {
			if block.hasPartition() {
				if err := u.deletePartition(blockPath); err != nil {
					log.Println("Issues while deleting partition on", blockPath, ":", err)
					u.formatErrors <- err
					return
				}
				// delete the block from the map as it shouldn't exist anymore
				delete(d.blockDevices, blockPath)
			}
		}

		// format the blocks with PartitionTable
		for blockPath, block := range d.blockDevices {
			if !block.isPartitionable() {
				continue
			}

			// perform sync call to format the device
			log.Println("Formatting", blockPath)
			err := u.syncFormat(blockPath)
			if err != nil {
				u.formatErrors <- err
			}
		}
		// no, we do not send a success because it should be done ONLY when we get a format job done
		// event from the dispatcher.
	}()
}

func (u *UDisks2) deletePartition(o dbus.ObjectPath) error {
	log.Println("Calling delete on", o)
	obj := u.conn.Object(dbusName, o)
	options := make(VariantMap)
	options["auth.no_user_interaction"] = dbus.Variant{true}
	_, err := obj.Call(dbusPartitionInterface, "Delete", options)
	return err
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
		log.Println("New mp found", mp)
		mountpoints[i] = mp
	}
	return mountpoints
}

func (u *UDisks2) ExternalDrives() []Drive {
	u.startLock.Lock()
	defer u.startLock.Unlock()
	var drives []Drive
	for _, d := range u.drives {
		if !d.hasSystemBlockDevices() && len(d.blockDevices) != 0 {
			drives = append(drives, *d)
		}
	}
	return drives
}

func (u *UDisks2) Init() (err error) {
	d, err := newDispatcher(u.conn)
	if err == nil {
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
				case j := <-u.jobs.FormatEraseJobs:
					if j.WasCompleted {
						log.Print("Erase job completed.")
					} else {
						log.Print("Erase job started.")
					}
				case j := <-u.jobs.FormatMkfsJobs:
					if j.WasCompleted {
						log.Println("Format job done for", j.Event.Path)
						u.pendingMounts = append(u.pendingMounts, j.Paths...)
						sort.Strings(u.pendingMounts)
					} else {
						log.Print("Format job started.")
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
	return err
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
	reply, err := obj.Call(dbusObjectManagerInterface, "GetManagedObjects")
	if err != nil {
		log.Println("Cannot get initial state for devices:", err)
	}
	log.Println("GetManagedObjects was done")

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
		u.formatCompleted <- s
	}

	if isBlockDevice, err := u.drives.addInterface(s); err != nil {
		return err
	} else if isBlockDevice {
		log.Println("New block device added", s.Path)
		if u.blockAdded != nil && u.blockError != nil {
			if ok, err := u.desiredMountableEvent(s); err != nil {
				u.blockError <- err
			} else if ok {
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

func isThumbDrive(driveProps VariantMap) bool {
	// According to the udisks docs USB drives will have "thumb" compatibility
	mediaCompatVar, ok := driveProps["MediaCompatibility"]
	if !ok {
		return false
	}

	mediaCompat := reflect.ValueOf(mediaCompatVar.Value)
	length := mediaCompat.Len()
	for i := 0; i < length; i++ {
		if mediaCompat.Index(i).Interface().(string) == "thumb" {
			return true
		}
	}
	return false
}

func exists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return true
}

func isAcceptedDevice(mediaRemovable, removable, thumbDrive bool) bool {
	// Some USB thumb devices report they have removable media where they
	// have not. See udisks2 API documentation for more details.
	// In summary we only accept devices which are marked as removable and
	// are thumb drivers. Otherwise all devices are not accepted.
	return (mediaRemovable || removable) && thumbDrive
}

func isAutomountEnabled() bool {
	return exists(os.Getenv("SNAP_COMMON") + "/.automount_enabled")
}

func (u *UDisks2) desiredMountableEvent(s *Event) (bool, error) {
	if isAutomountEnabled() {
		return false, nil
	}

	_, ok := s.Props[dbusBlockInterface]
	if !ok {
		log.Println("Block interface is missing")
		return false, nil
	}

	// No file system interface means we can't mount it even if we wanted to
	_, ok = s.Props[dbusFilesystemInterface]
	if !ok {
		log.Println("Filesystem interface is missing.")
		return false, nil
	}

	drivePath, err := s.getDrive()
	if err != nil {
		log.Println("Issues while getting drive:", err)
		return false, nil
	}

	drive, ok := u.drives[drivePath]
	if !ok {
		log.Println("Drive with path", drivePath, "not found")
		return false, nil
	}

	if ok := drive.hasSystemBlockDevices(); ok {
		log.Println(drivePath, "which contains", s.Path, "has HintSystem set")
		return false, nil
	}

	driveProps, ok := drive.driveInfo[dbusDriveInterface]
	if !ok {
		log.Println(drivePath, "doesn't hold a Drive interface")
		return false, nil
	}
	removableVariant, ok := driveProps["Removable"]
	if !ok {
		log.Println(drivePath, "which holds", s.Path, "doesn't have Removable")
		return false, nil
	}
	mediaRemovableVariant, ok := driveProps["MediaRemovable"]
	if !ok {
		log.Println(drivePath, "which holds", s.Path, "doesn't have MediaRemovable")
		return false, nil
	}

	removable := reflect.ValueOf(removableVariant.Value).Bool()
	mediaRemovable := reflect.ValueOf(mediaRemovableVariant.Value).Bool()

	if !isAcceptedDevice(mediaRemovable, removable, true) {
		log.Println(drivePath, "which holds", s.Path, "is not Removable or MediaRemovable and a thumb drive")
		return false, nil
	}

	if s.Props.isMounted() {
		return false, nil
	}

	propBlock, ok := s.Props[dbusBlockInterface]
	if !ok {
		return false, nil
	}

	hintIgnore, ok := propBlock["HintIgnore"]
	if !ok {
		log.Println(s.Path, "doesn't hold HintIgnore")
		return false, nil
	}

	if reflect.ValueOf(hintIgnore.Value).Bool() {
		return false, nil
	}

	hintSystem, ok := propBlock["HintSystem"]
	if !ok {
		log.Println(s.Path, "doesn't hold HintIgnore")
		return false, nil
	}

	if reflect.ValueOf(hintSystem.Value).Bool() {
		return false, nil
	}

	id, ok := propBlock["IdType"]
	if !ok {
		log.Println(s.Path, "doesn't hold IdType")
		return false, nil
	}

	fs := reflect.ValueOf(id.Value).String()
	if fs == "" {
		return false, nil
	}

	return true, nil
}

func (d *Drive) hasSystemBlockDevices() bool {
	for _, blockDevice := range d.blockDevices {
		if propBlock, ok := blockDevice[dbusBlockInterface]; ok {
			if systemHintVariant, ok := propBlock["HintSystem"]; ok {
				return reflect.ValueOf(systemHintVariant.Value).Bool()
			}
		}
	}
	return false
}

func (d *Drive) Model() string {
	propDrive, ok := d.driveInfo[dbusDriveInterface]
	if !ok {
		return ""
	}
	modelVariant, ok := propDrive["Model"]
	if !ok {
		return ""
	}
	return reflect.ValueOf(modelVariant.Value).String()
}

func (d *Drive) SetMounted(path dbus.ObjectPath) bool {
	for p, _ := range d.blockDevices {
		if p == path {
			d.Mounted = true
			return true
		}
	}
	return false
}

func (s *Event) getDrive() (dbus.ObjectPath, error) {
	propBlock, ok := s.Props[dbusBlockInterface]
	if !ok {
		return "", fmt.Errorf("interface %s not found", dbusBlockInterface)
	}
	driveVariant, ok := propBlock["Drive"]
	if !ok {
		return "", errors.New("property 'Drive' not found")
	}
	return dbus.ObjectPath(reflect.ValueOf(driveVariant.Value).String()), nil
}

func newDrive(s *Event) *Drive {
	return &Drive{
		Path:         s.Path,
		blockDevices: make(map[dbus.ObjectPath]InterfacesAndProperties),
		driveInfo:    s.Props,
		Mounted:      s.Props.isMounted(),
	}
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
		(*dm)[s.Path] = newDrive(s)
	case deviceTypeBlock:
		driveObjectPath, err := s.getDrive()
		if err != nil {
			return blockDevice, err
		}
		if _, ok := (*dm)[driveObjectPath]; !ok {
			drive := newDrive(s)
			log.Println("Creating new drive", drive)
			(*dm)[s.Path] = drive
		} else {
			(*dm)[driveObjectPath].blockDevices[s.Path] = s.Props
			(*dm)[driveObjectPath].Mounted = s.Props.isMounted()
		}
		blockDevice = true
	default:
		// we don't care about other object paths
		log.Println("Unhandled object path", s.Path)
	}

	return blockDevice, nil
}
