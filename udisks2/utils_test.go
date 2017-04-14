/*
 * Copyright 2017 Canonical Ltd.
 *
 * Authors:
 * Konrad Zapalowicz <konrad.zapalowicz@canonical.com>
 *
 * ciborium is free software; you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation; version 3.
 *
 * ciborium is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package udisks2

import (
	. "launchpad.net/gocheck"
)

type UtilsTestSuite struct{}

var _ = Suite(&UtilsTestSuite{})

func (s *UtilsTestSuite) TestFibonacci(c *C) {
	fib := fibonacci()
	c.Assert(fib(), Equals, 1)
	c.Assert(fib(), Equals, 1)
	c.Assert(fib(), Equals, 2)
	c.Assert(fib(), Equals, 3)
	c.Assert(fib(), Equals, 5)
}
