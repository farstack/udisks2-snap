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
	"reflect"

	"launchpad.net/go-dbus/v1"
)

type Event struct {
	Path       dbus.ObjectPath
	Props      InterfacesAndProperties
	Interfaces Interfaces
}

// isRemovalEvent returns if an event represents an InterfacesRemoved signal from the dbus ObjectManager
// dbus interface. An event is a removal event when it carries a set of the interfaces that have been lost
// in a dbus object path.
func (e *Event) isRemovalEvent() bool {
	return len(e.Interfaces) != 0
}

func (e *Event) getDriveObjectPath() (dbus.ObjectPath, error) {
	propBlock, ok := e.Props[dbusBlockInterface]
	if !ok {
		return "", fmt.Errorf("interface %s not found", dbusBlockInterface)
	}
	driveVariant, ok := propBlock["Drive"]
	if !ok {
		return "", errors.New("property 'Drive' not found")
	}
	return dbus.ObjectPath(reflect.ValueOf(driveVariant.Value).String()), nil
}

func (e *Event) hasInterface(name string) bool {
	_, ok := e.Props[name]
	return ok
}

func (e *Event) isBlockDeviceAndIgnored() bool {

	return false
}
