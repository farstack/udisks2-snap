/*
 * Copyright 2014 Canonical Ltd.
 *
 * Authors:
 * Sergio Schvezov: sergio.schvezov@canonical.com
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

	"launchpad.net/ciborium/udisks2"
	dbus "launchpad.net/go-dbus/v1"
)

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

	watch, err := systemBus.WatchName("org.freedesktop.UDisks2")
	if err != nil {
		log.Fatal("Failed to setup watch for udisks2 service:", err)
	}
	defer watch.Cancel()

	ownerAppeared := make(chan int, 1)
	go func() {
		for owner := range watch.C {
			log.Println("New owner", owner, "appeared for udisks2 service name")
			ownerAppeared <- 0
		}
	}()

	// Wait until an owner for udisks2 appears and we can start talking to it
	<-ownerAppeared

	udisks2 := udisks2.NewStorageWatcher(systemBus)

	blockAdded, blockError := udisks2.SubscribeAddEvents()
	mountRemoved := udisks2.SubscribeRemoveEvents()
	mountCompleted, mountErrors := udisks2.SubscribeMountEvents()

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
				log.Println("New block device added")
				udisks2.Mount(a)
			case e := <-blockError:
				log.Println("Issues in block for added drive:", e)
			case e := <-mountErrors:
				log.Println("Failed to mount device:", e)
			case path := <-mountCompleted:
				log.Println("Successfully mount device:", path)
			case m := <-mountRemoved:
				log.Println("Path removed", m)
			}
		}
	}()

	if err := udisks2.Init(); err != nil {
		log.Fatal("Cannot monitor storage devices:", err)
	}

	done := make(chan bool)
	<-done
}
