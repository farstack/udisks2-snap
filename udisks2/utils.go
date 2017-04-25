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

// Returns a function that returns fibonacci sequence
// that starts from 1
func fibonacci() func() int {
	a, b := 1, 1
	return func() int {
		retval := a
		a, b = b, a+b
		return retval
	}
}
