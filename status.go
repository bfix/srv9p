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
	"sync/atomic"
	"time"
)

// status codes
const (
	StatUNK     = iota // unknown status (init)
	StatOK             // processing active
	StatDEV            // device failure
	StatNS             // namespace construction failed
	StatSRV            // can't serve namespace
	StatIP             // invalid IP address
	StatWIFI           // can't connect to AP
	StatWPA2           // WPA2 failed
	StatDHCP1          // DHCP request failed
	StatDHCP2          // no DHCP reply
	StatLISTEN1        // failed to create listener
	StatLISTEN2        // failed to initialize listener
	StatPORT           // invalid port specified
	StatEXCP           // exception (panic) occured
)

// Status handler.
// Show current status depending on hardware device.
type Status struct {
	dev    Device       // reference to device
	curr   atomic.Int32 // current state
	repeat atomic.Int32 // current repeat counter
}

// NewStatus creates a new status display
func NewStatus(dev Device) (state *Status) {
	state = new(Status)
	state.dev = dev
	go func() {
		state.curr.Store(StatOK)
		state.repeat.Store(0)
		// blink LED <state>; <repeat> times
		for {
			time.Sleep(5 * time.Second)
			num := state.curr.Load()
			for num > 5 {
				dev.LED(true)
				time.Sleep(1000 * time.Millisecond)
				dev.LED(false)
				time.Sleep(300 * time.Millisecond)
				num -= 5
			}
			for range num {
				dev.LED(true)
				time.Sleep(150 * time.Millisecond)
				dev.LED(false)
				time.Sleep(150 * time.Millisecond)
			}
			if state.repeat.Add(-1) == 0 {
				state.curr.Store(StatOK)
			}
		}
	}()
	return
}

// Set status and repeat <num> times.
func (state *Status) Set(flag, num int) {
	if state != nil {
		state.curr.Store(int32(flag))
		state.repeat.Store(int32(num))
	}
}

// Get current state and repeat counter
func (state *Status) Get() (int, int) {
	return int(state.curr.Load()), int(state.repeat.Load())
}

// Trap critical failures (panic)
func (state *Status) Trap(t time.Duration) {
	s, _ := state.Get()
	if r := recover(); r != nil {
		fmt.Printf("EXCP: %v\n", r)
		if s == StatOK {
			state.Set(StatEXCP, 0)
		}
	} else if s == StatOK {
		state.Set(StatUNK, 0)
	}
	time.Sleep(t)
}
