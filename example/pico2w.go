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

package main

import (
	"fmt"
	"math/rand"
	"net"
	"strconv"
	"time"

	"git.sr.ht/~moody/ninep"
	"github.com/bfix/srv9p"
)

// WiFi credentials and 9p port
// Variables can be set at compile time by adding
//
//	-ldflags "-X 'main.SSID=MyWiFi' -X 'main.Passwd=MySecret' -X 'main.Host=pico' -X 'main.Port=564'"
//
// to the build/install command.
var (
	SSID   string
	Passwd string
	Host   string
	IP     string
	Port   string
)

// run 9p server
func main() {
	// access device
	dev := srv9p.InitDevice()
	state := srv9p.NewStatus(dev)
	defer func() {
		s, _ := state.Get()
		if r := recover(); r != nil {
			if s == srv9p.StatOK {
				state.Set(srv9p.StatEXCP, 0)
			}
		} else if s == srv9p.StatOK {
			state.Set(srv9p.StatUNK, 0)
		}
		time.Sleep(24 * time.Hour)
	}()
	state.Set(srv9p.StatOK, 0)

	// construct filesystem
	check := func(err error) {
		if err != nil {
			state.Set(srv9p.StatNS, 0)
			panic(err)
		}
	}
	fs := srv9p.NewNamespace("sys", "sys")
	check(fs.NewFile("/readme", 0444, NewTextFile("Just a test...\n")))
	check(fs.NewDir("/sensors", 0777))
	check(fs.NewFile("/sensors/temp", 0444, new(DynamicFile)))

	// connect to WiFi and listen to 9p connections
	port, err := strconv.ParseInt(Port, 10, 16)
	if err != nil {
		state.Set(srv9p.StatPORT, 0)
	}
	var lst net.Listener
	var stat int
	if lst, stat = dev.SetupListener(Host, IP, SSID, Passwd, uint16(port)); stat != srv9p.StatOK {
		state.Set(stat, 0)
		return
	}

	// serve filesystem via 9p
	for {
		c, err := lst.Accept()
		if err != nil {
			state.Set(srv9p.StatSRV, 3)
			continue
		}
		srv := ninep.NewSrv(func() ninep.FS { return fs })
		go srv.ServeIO(c, c)
	}

	// srv tcp!<host>!9fs test
	// mount /src/test /n/test
	// lc /n/test
	// cat /n/test/static
	// unmount /n/test
	// rm /srv/test
}

//======================================================================
// File implementations
//======================================================================

// TextFile with static text content.
type TextFile struct {
	srv9p.NopFile
	body string
}

// NewTextFile with given text content.
func NewTextFile(content string) *TextFile {
	return &TextFile{
		body: content,
	}
}

// Read implementation: return the file content.
func (f *TextFile) Read() ([]byte, error) {
	return []byte(f.body), nil
}

//----------------------------------------------------------------------

// DynamicFile with volatile content.
type DynamicFile struct {
	srv9p.NopFile
}

// Read implementation: return the file content.
func (f *DynamicFile) Read() ([]byte, error) {
	s := fmt.Sprintf("%f\n", rand.Float32())
	return []byte(s), nil
}
