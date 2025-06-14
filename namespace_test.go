//----------------------------------------------------------------------
// This file is part of srv9p.
// Copyright (C) 2024-present Bernd Fix   >Y<
//
// srv9p is free software: you can redistribute it and/or modify it
// under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License,
// or (at your option) any later version.
//
// srv9p is distributed in the hope that it will be useful, but
// WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU
// Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.
//
// SPDX-License-Identifier: AGPL3.0-or-later
//----------------------------------------------------------------------

package srv9p

import (
	"fmt"
	"math/rand/v2"
	"testing"
)

// build a test namespace
func newNamespace() (ns *Namespace, err error) {
	ns = NewNamespace("sys", "sys")
	if err = ns.NewFile("/readme", 0444, NewTextFile("Just a test...\n")); err != nil {
		return
	}
	if err = ns.NewDir("/sensors", 0777); err != nil {
		return
	}
	err = ns.NewFile("/sensors/temp", 0444, NewFuncFile(
		func() ([]byte, error) {
			s := fmt.Sprintf("%f\n", rand.Float32())
			return []byte(s), nil
		},
	))
	return
}

func TestNamespaceNew(t *testing.T) {
	newNamespace()
}
