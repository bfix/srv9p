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

import "net"

// Device is a hardware abstraction
type Device interface {
	// LED on or off (if applicable)
	LED(on bool)

	// SetupListener returns a TCP listener on the given port.
	// On embedded devices with WiFi connectivity the following steps are
	// performed:
	//    1. Connect to access point with given SSID and join with WPA2 password
	//    2. Use DHCP to get a network address; query for given hostname. Assign
	//       IP address if DHCP fails (if IP is a valid address).
	//    3. Listen to the specified TCP port
	// Devices with running TCP/IP stack can skip steps 1. and 2. in their
	// implementation.
	SetupListener(host, ip, ssid, passwd string, port uint16) (lst net.Listener, state int)
}
