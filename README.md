# srv9p : serve namespace over 9P protocol on embedded devices

Copyright (C) 2024-present, Bernd Fix  >Y<

srv9p is free software: you can redistribute it and/or modify it
under the terms of the GNU Affero General Public License as published
by the Free Software Foundation, either version 3 of the License,
or (at your option) any later version.

srv9p is distributed in the hope that it will be useful, but
WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU
Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <http://www.gnu.org/licenses/>.

SPDX-License-Identifier: AGPL3.0-or-later

## Caveat

This is work-in-progess. Use at your own risk...

## Intro

`srv9p` serves namespaces over 9P and is meant for embedded devices
with TCP/IP connectivity but limited RAM/ROM space. It is written in
Go and can be compiled with [tinygo](https://tinygo.org/) for (nearly)
all supported devices.

For the 9P protocol implementation, `srv9p` uses a library
(https://git.sr.ht/~moody/ninep) that is compact and sufficient.

## Example

See the example app to learn how to use `srv9p`:

    tinygo build -target=pico2-w -o srv9p.uf2 ./example/pico2w.go