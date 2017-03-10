package udisks2

import (
	"io/ioutil"
	"os"
	"path/filepath"

	. "launchpad.net/gocheck"
)

type Udisks2TestSuite struct{}

var _ = Suite(&Udisks2TestSuite{})

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
