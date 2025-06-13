//go:build !rp2350

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
	"context"
	"fmt"
	"net"
)

// LinuxDevice (for testing purposes)
type LinuxDevice struct{}

// LED on or off (not applicable)
func (dev *LinuxDevice) LED(on bool) {}

// Initialize device
func InitDevice() (dev Device) {
	return new(LinuxDevice)
}

// SetupListener returns a TCP listener on the given port.
func (dev *LinuxDevice) SetupListener(_, _, _, _ string, port uint16) (lst net.Listener, state int) {
	ctx := context.Background()
	cfg := new(net.ListenConfig)
	lis, err := cfg.Listen(ctx, "tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, statLISTEN1
	}
	return lis, nil
}
