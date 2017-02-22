package udisks2

import . "launchpad.net/gocheck"

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
