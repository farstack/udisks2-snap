/*
 * Copyright 2014 Canonical Ltd.
 *
 * Authors:
 * Sergio Schvezov: sergio.schvezov@cannical.com
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

package main

import (
	"log"
	"strings"
	"sync"

	"launchpad.net/ciborium/udisks2"
	"launchpad.net/go-dbus/v1"
)

type message struct{ Summary, Body string }
type notifyFreeFunc func(mountpoint) error

type mountpoint string

func (m mountpoint) external() bool {
	return strings.HasPrefix(string(m), "/media")
}

type mountwatch struct {
	lock        sync.Mutex
	mountpoints map[mountpoint]bool
}

func (m *mountwatch) set(path mountpoint, state bool) {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.mountpoints[path] = state
}

func (m *mountwatch) getMountpoints() []mountpoint {
	m.lock.Lock()
	defer m.lock.Unlock()

	mapLen := len(m.mountpoints)
	mountpoints := make([]mountpoint, 0, mapLen)
	for p := range m.mountpoints {
		mountpoints = append(mountpoints, p)
	}
	return mountpoints
}

func (m *mountwatch) warn(path mountpoint) bool {
	m.lock.Lock()
	defer m.lock.Unlock()

	return m.mountpoints[path]
}

func (m *mountwatch) remove(path mountpoint) {
	m.lock.Lock()
	defer m.lock.Unlock()

	delete(m.mountpoints, path)
}

func newMountwatch() *mountwatch {
	return &mountwatch{
		mountpoints: make(map[mountpoint]bool),
	}
}

const (
	sdCardIcon                = "media-memory-sd"
	errorIcon                 = "error"
	homeMountpoint mountpoint = "/home"
	freeThreshold             = 5
)

var (
	mw *mountwatch
)

func init() {
	mw = newMountwatch()
	mw.set(homeMountpoint, true)
}

func main() {
	// set default logger flags to get more useful info
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	var (
		systemBus *dbus.Connection
		err       error
	)

	if systemBus, err = dbus.Connect(dbus.SystemBus); err != nil {
		log.Fatal("Connection error: ", err)
	}
	log.Print("Using system bus on ", systemBus.UniqueName)

	udisks2 := udisks2.NewStorageWatcher(systemBus)

	blockAdded, blockError := udisks2.SubscribeAddEvents()
	formatCompleted, formatErrors := udisks2.SubscribeFormatEvents()
	unmountCompleted, unmountErrors := udisks2.SubscribeUnmountEvents()
	mountCompleted, mountErrors := udisks2.SubscribeMountEvents()
	mountRemoved := udisks2.SubscribeRemoveEvents()

	// create a routine per couple of channels, the select algorithm will make use
	// ignore some events if more than one channels is being written to the algorithm
	// will pick one at random but we want to make sure that we always react, the pairs
	// are safe since the deal with complementary events

	// block additions
	go func() {
		log.Println("Listening for addition and removal events.")
		for {
			select {
			case a := <-blockAdded:
				udisks2.Mount(a)
			case e := <-blockError:
				log.Println("Issues in block for added drive:", e)
			case m := <-mountRemoved:
				log.Println("Path removed", m)
				mw.remove(mountpoint(m))
			}
		}
	}()

	// mount operations
	go func() {
		log.Println("Listening for mount and unmount events.")
		for {
			select {
			case m := <-mountCompleted:
				log.Println("Mounted", m)
				mw.set(mountpoint(m.Mountpoint), true)
			case e := <-mountErrors:
				log.Println("Error while mounting device", e)

			case m := <-unmountCompleted:
				log.Println("Path removed", m)
				mw.remove(mountpoint(m))
			case e := <-unmountErrors:
				log.Println("Error while unmounting device", e)

			}
		}
	}()

	// format operations
	go func() {
		for {
			select {
			case f := <-formatCompleted:
				log.Println("Format done. Trying to mount.")
				udisks2.Mount(f)
			case e := <-formatErrors:
				log.Println("There was an error while formatting", e)
			}
		}
	}()

	if err := udisks2.Init(); err != nil {
		log.Fatal("Cannot monitor storage devices:", err)
	}

	done := make(chan bool)
	<-done
}
