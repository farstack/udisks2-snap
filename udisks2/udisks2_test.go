package udisks2

import (
	"io/ioutil"
	"os"
	"path/filepath"

	. "launchpad.net/gocheck"
)

type Udisks2TestSuite struct{}

var _ = Suite(&Udisks2TestSuite{})

func (s *Udisks2TestSuite) TestIsDeviceAcceptedForAutomount(c *C) {
	// Device with MediaRemovable = true, Removable = true and MediaCompatibility = []
	c.Assert(isAcceptedDevice(true, true, false), Equals, false)
	// Device with MediaRemovable = true, Removable = false and MediaCompatibility = []
	c.Assert(isAcceptedDevice(true, false, false), Equals, false)
	// Device with MediaRemovable = false, Removable = false and MediaCompatibility = []
	c.Assert(isAcceptedDevice(false, false, false), Equals, false)
	// Device with MediaRemovable = true, Removable = true and MediaCompatibility = [thumb]
	c.Assert(isAcceptedDevice(true, true, true), Equals, true)
	// Device with MediaRemovable = true, Removable = false and MediaCompatibility = [thumb]
	c.Assert(isAcceptedDevice(true, false, true), Equals, true)
	// Device with MediaRemovable = false, Removable = true and MediaCompatibility = [thumb]
	c.Assert(isAcceptedDevice(false, true, true), Equals, true)
	// Device with MediaRemovable = false, Removable = false and MediaCompatibility = [thumb]
	c.Assert(isAcceptedDevice(false, false, true), Equals, false)
}

func (s *Udisks2TestSuite) TestAutomountEnabledOrNot(c *C) {
	snapCommonDir, err := ioutil.TempDir("/tmp", "udisks2")
	c.Assert(err, IsNil)
	os.Setenv("SNAP_COMMON", snapCommonDir)
	c.Assert(isAutomountEnabled(), Equals, false)
	automountMarkerPath := filepath.Join(snapCommonDir, ".automount_enabled")
	err = ioutil.WriteFile(automountMarkerPath, []byte("nothing"), 0644)
	c.Assert(err, IsNil)
	c.Assert(isAutomountEnabled(), Equals, true)
	err = os.Remove(automountMarkerPath)
	c.Assert(err, IsNil)
	c.Assert(isAutomountEnabled(), Equals, false)
}
