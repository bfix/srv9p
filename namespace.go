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
	"errors"
	"strings"

	"git.sr.ht/~moody/ninep"
)

// Error messages
var (
	errNoRoot = errors.New("no root directory")
	errNoFile = errors.New("no such file or directory")
	errNoDir  = errors.New("not a directory")
	errNoAbs  = errors.New("no absolute path")
)

//----------------------------------------------------------------------

// File interface for file handler implementations:
// The interface methods are called by the 9p protocol handler on demand.
// The implementation is free to handle the read/write calls according
// to its own logic.
type File interface {
	Read() ([]byte, error)
	Write([]byte) error
}

//----------------------------------------------------------------------

// Entry in the filesystem
type Entry struct {
	ref      *ninep.Dir        // 9p reference
	children map[string]*Entry // list of children (for folders) or nil
	file     File              // file implementation or nil (for folders)
}

// IsDir returns true if entry is a directory
func (e *Entry) IsDir() bool {
	return e.children != nil
}

// NewFile creates a file entry for the filesystem.
func NewFile(name, user, group string, perm uint32, impl File) *Entry {
	return newEntry(name, user, group, perm, impl)
}

// NewDir creates a directory entry for the filesystem.
func NewDir(name, user, group string, perm uint32) *Entry {
	return newEntry(name, user, group, perm, nil)
}

// Create a new entry in the filesystem.
// If impl is nil, the entry represents a directory; otherwise a file.
func newEntry(name, user, group string, perm uint32, impl File) *Entry {
	e := new(Entry)
	kind := ninep.QTFile
	if impl == nil {
		kind = ninep.QTDir
		e.children = make(map[string]*Entry)
		perm |= ninep.DMDir
	} else {
		e.file = impl
	}
	e.ref = &ninep.Dir{
		Qid: ninep.Qid{
			Path: newId(),
			Vers: 0,
			Type: byte(kind),
		},
		Name: name,
		Mode: perm,
		Uid:  user,
		Gid:  group,
		Muid: user,
	}
	return e
}

// next possible identifier (Qid.Path) for an entry in the filesystem.
var nextId uint64 = 0

// get next identifier for an entry.
func newId() uint64 {
	id := nextId
	nextId++
	return id
}

//----------------------------------------------------------------------

// Namespace is a synthetic file system.
type Namespace struct {
	ninep.NopFS                   // use default handlers where needed
	dict        map[uint64]*Entry // map Qid.Path to filesystem entry
}

// NewNamespace creates a new filesystem (with root directory) for the given
// user/group with the specified permissions.
func NewNamespace(user, group string, perm uint32) *Namespace {
	ns := new(Namespace)
	ns.dict = make(map[uint64]*Entry)
	e := NewDir("/", user, group, perm)
	ns.dict[e.ref.Path] = e
	return ns
}

// Root returns the entry of the root directory
func (ns *Namespace) Root() *Entry {
	return ns.dict[0]
}

// Get entry with given path
func (ns *Namespace) Get(path string) (*Entry, error) {
	if path[0] != '/' {
		return nil, errNoAbs
	}
	curr := ns.Root()
	for _, label := range strings.Split(path[1:], "/") {
		if len(label) == 0 {
			continue
		}
		if curr.children == nil {
			return nil, errNoDir
		}
		qid := ns.Walk(&curr.ref.Qid, label)
		e, ok := ns.dict[qid.Path]
		if !ok {
			return nil, errNoFile
		}
		curr = e
	}
	return curr, nil
}

// AddChild to parent entry. Parent must be a directory.
func (ns *Namespace) AddChild(parent, child *Entry) error {
	if parent.children == nil {
		return errNoDir
	}
	parent.children[child.ref.Name] = child
	ns.dict[child.ref.Path] = child
	return nil
}

// Serve the 9p protocol for the given listen string
func (ns *Namespace) Serve(listen string) error {
	srv := ninep.NewSrv(func() ninep.FS { return ns })
	return srv.ListenAndServe(listen)
}

// ninep FS implementation

// Attach to 9p session
func (ns *Namespace) Attach(t *ninep.Tattach) {
	if e, ok := ns.dict[0]; ok {
		t.Respond(&e.ref.Qid)
	} else {
		t.Err(errNoRoot)
	}
}

// Walk to child entry with name "next".
func (ns *Namespace) Walk(cur *ninep.Qid, next string) *ninep.Qid {
	e := ns.dict[cur.Path]
	for _, c := range e.children {
		if c.ref.Name == next {
			return &c.ref.Qid
		}
	}
	return nil
}

// Open entry for file operation
func (ns *Namespace) Open(t *ninep.Topen, q *ninep.Qid) {
	t.Respond(q, 8192)
}

// Read from entry. Either return the content of a file
// or the listing from a directory.
func (ns *Namespace) Read(t *ninep.Tread, q *ninep.Qid) {
	e, ok := ns.dict[q.Path]
	if !ok {
		t.Err(errNoFile)
		return
	}
	if e.children != nil {
		var kids []ninep.Dir
		for _, c := range e.children {
			kids = append(kids, *c.ref)
		}
		ninep.ReadDir(t, kids)
		return
	}
	data, err := e.file.Read()
	if err != nil {
		t.Err(err)
	} else {
		ninep.ReadBuf(t, data)
	}
}

// Stat returns information for a filesytem entry.
func (ns *Namespace) Stat(t *ninep.Tstat, q *ninep.Qid) {
	e, ok := ns.dict[q.Path]
	if !ok {
		t.Err(errNoFile)
	} else {
		t.Respond(e.ref)
	}
}
