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
	"reflect"

	"launchpad.net/go-dbus/v1"
)

type Drive struct {
	Path         dbus.ObjectPath
	BlockDevices map[dbus.ObjectPath]InterfacesAndProperties
	DriveInfo    InterfacesAndProperties
	Mounted      bool
}

func (d *Drive) IsRemovable() bool {
	props, ok := d.DriveInfo[dbusDriveInterface]
	if !ok {
		return false
	}

	removable, hasRemovable := props["Removable"]
	mediaRemovable, hasMediaRemovable := props["MediaRemovable"]

	if hasRemovable && hasMediaRemovable {
		// Some USB thumb devices report they have removable media where they
		// have not. See udisks2 API documentation for more details.
		// In summary we only accept devices which are marked as removable and
		// are thumb drivers. Otherwise all devices are not accepted.
		return reflect.ValueOf(removable.Value).Bool() ||
			reflect.ValueOf(mediaRemovable.Value).Bool()
	} else if hasRemovable {
		return reflect.ValueOf(removable.Value).Bool()
	} else if hasMediaRemovable {
		return reflect.ValueOf(mediaRemovable.Value).Bool()
	}
	return false
}

func (d *Drive) HasSystemBlockDevices() bool {
	for _, blockDevice := range d.BlockDevices {
		if propBlock, ok := blockDevice[dbusBlockInterface]; ok {
			if systemHintVariant, ok := propBlock["HintSystem"]; ok {
				return reflect.ValueOf(systemHintVariant.Value).Bool()
			}
		}
	}
	return false
}

func (d *Drive) Model() string {
	propDrive, ok := d.DriveInfo[dbusDriveInterface]
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
	for p, _ := range d.BlockDevices {
		if p == path {
			d.Mounted = true
			return true
		}
	}
	return false
}

func NewDrive(s *Event) *Drive {
	return &Drive{
		Path:         s.Path,
		BlockDevices: make(map[dbus.ObjectPath]InterfacesAndProperties),
		DriveInfo:    s.Props,
		Mounted:      s.Props.isMounted(),
	}
}
