/*
 * Copyright 2016 Canonical Ltd.
 *
 * Authors:
 * Simon Fels <simon.fels@canonical.com>
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
	"launchpad.net/go-dbus/v1"
	. "launchpad.net/gocheck"
)

type DriveTestSuite struct{}

var _ = Suite(&DriveTestSuite{})

func (s *DriveTestSuite) TestDriveNoSystemDrivesWithEmptyEvent(c *C) {
	drive := &Drive{}
	c.Assert(drive.HasSystemBlockDevices(), Equals, false)
}

func (s *DriveTestSuite) TestDriveHasSystemDevices(c *C) {
	drive := &Drive{
		BlockDevices: map[dbus.ObjectPath]InterfacesAndProperties{
			dbus.ObjectPath("/foo/bar"): InterfacesAndProperties{
				dbusBlockInterface: VariantMap{
					"HintSystem": dbus.Variant{true},
				},
			},
		},
	}
	c.Assert(drive.HasSystemBlockDevices(), Equals, true)
}

func (s *DriveTestSuite) TestDriveHasNoSystemDevices(c *C) {
	drive := &Drive{
		BlockDevices: map[dbus.ObjectPath]InterfacesAndProperties{
			dbus.ObjectPath("/foo/bar"): InterfacesAndProperties{
				dbusBlockInterface: VariantMap{
					"HintSystem": dbus.Variant{false},
				},
			},
		},
	}
	c.Assert(drive.HasSystemBlockDevices(), Equals, false)
}

func (s *DriveTestSuite) TestDriveIsNotRemovable(c *C) {
	drive := &Drive{}
	c.Assert(drive.IsRemovable(), Equals, false)

	drive = &Drive{
		DriveInfo: InterfacesAndProperties{
			dbusDriveInterface: VariantMap{
				"Removable":      dbus.Variant{false},
				"MediaRemovable": dbus.Variant{false},
			},
		},
	}
	c.Assert(drive.IsRemovable(), Equals, false)

	drive = &Drive{
		DriveInfo: InterfacesAndProperties{
			dbusDriveInterface: VariantMap{
				"Removable": dbus.Variant{false},
			},
		},
	}
	c.Assert(drive.IsRemovable(), Equals, false)

	drive = &Drive{
		DriveInfo: InterfacesAndProperties{
			dbusDriveInterface: VariantMap{
				"MediaRemovable": dbus.Variant{false},
			},
		},
	}
	c.Assert(drive.IsRemovable(), Equals, false)
}

func (s *DriveTestSuite) TestDriveIsRemovable(c *C) {
	drive := &Drive{
		DriveInfo: InterfacesAndProperties{
			dbusDriveInterface: VariantMap{
				"Removable":      dbus.Variant{true},
				"MediaRemovable": dbus.Variant{false},
			},
		},
	}
	c.Assert(drive.IsRemovable(), Equals, true)

	drive = &Drive{
		DriveInfo: InterfacesAndProperties{
			dbusDriveInterface: VariantMap{
				"Removable":      dbus.Variant{true},
				"MediaRemovable": dbus.Variant{true},
			},
		},
	}
	c.Assert(drive.IsRemovable(), Equals, true)

	drive = &Drive{
		DriveInfo: InterfacesAndProperties{
			dbusDriveInterface: VariantMap{
				"Removable":      dbus.Variant{false},
				"MediaRemovable": dbus.Variant{true},
			},
		},
	}
	c.Assert(drive.IsRemovable(), Equals, true)
}
