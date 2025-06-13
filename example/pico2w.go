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
	"errors"
	"fmt"
	"math/rand"
	"net"
	"strconv"
	"time"

	"git.sr.ht/~moody/ninep"
	"github.com/bfix/srv9p"
)

// StaticFile with static content.
type StaticFile struct {
	content []byte
}

// NewStaticFile with given content.
func NewStaticFile(content string) *StaticFile {
	return &StaticFile{
		content: []byte(content),
	}
}

// Read implementation: return the file content.
func (f *StaticFile) Read() ([]byte, error) {
	return f.content, nil
}

// Write implementation: we are read only, so return an error
func (f *StaticFile) Write(data []byte) error {
	return errors.New("write prohibited")
}

//----------------------------------------------------------------------

// DynamicFile with volatile content.
type DynamicFile struct{}

// Read implementation: return the file content.
func (f *DynamicFile) Read() ([]byte, error) {
	s := fmt.Sprintf("%f\n", rand.Float32())
	return []byte(s), nil
}

// Write implementation: we are read only, so return an error
func (f *DynamicFile) Write(data []byte) error {
	return errors.New("write prohibited")
}

//----------------------------------------------------------------------

// WiFi credentials and 9p port
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
		if r := recover(); r != nil {
			state.Set(srv9p.StatEXCP, 0)
		} else {
			if s, _ := state.Get(); s == srv9p.StatOK {
				state.Set(srv9p.StatUNK, 0)
			}
		}
		time.Sleep(30 * time.Second)
	}()
	state.Set(srv9p.StatOK, 0)

	// construct filesystem
	fs := srv9p.NewNamespace("sys", "sys", 0777)
	root := fs.Root()
	sfile := NewStaticFile("Just a test...\n")
	fs.AddChild(root, srv9p.NewFile("static", "sys", "sys", 0444, sfile))
	dir := srv9p.NewDir("sensors", "sys", "sys", 0777)
	fs.AddChild(root, dir)
	dfile := new(DynamicFile)
	fs.AddChild(dir, srv9p.NewFile("temp", "sys", "sys", 0444, dfile))

	// connect to WiFi and listen to 9p connections
	port, err := strconv.ParseInt(Port, 10, 16)
	if err != nil {
		state.Set(srv9p.StatPORT, 0)
	}
	var lst net.Listener
	var stat int
	if lst, stat = srv9p.SetupListener(dev, Host, IP, SSID, Passwd, uint16(port)); stat != srv9p.StatOK {
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
