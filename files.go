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

// File interface for file handler implementations:
// The interface methods are called by the 9p protocol handler on demand.
// The implementation is free to handle the read/write calls according
// to its own logic.
type File interface {
	Read() ([]byte, error)
	Write([]byte) error
}

//----------------------------------------------------------------------

// NopFile ignores all read/write requests
type NopFile struct{}

// Read returns emtpy file
func (f *NopFile) Read() (data []byte, err error) {
	return
}

// Write to file is ignored
func (f *NopFile) Write([]byte) (err error) {
	return
}

//----------------------------------------------------------------------

// TextFile with (small) static text content.
type TextFile struct {
	NopFile
	body string
}

// NewTextFile with given text content.
func NewTextFile(content string) *TextFile {
	return &TextFile{
		body: content,
	}
}

// Read implementation: return file content.
func (f *TextFile) Read() ([]byte, error) {
	return []byte(f.body), nil
}

//----------------------------------------------------------------------

// FuncFile content is returned by a function.
type FuncFile struct {
	NopFile
	fcn func() ([]byte, error)
}

// NewFuncFile with specified function.
func NewFuncFile(fcn func() ([]byte, error)) *FuncFile {
	return &FuncFile{
		fcn: fcn,
	}
}

// Read implementation: return file content.
func (f *FuncFile) Read() ([]byte, error) {
	return f.fcn()
}
