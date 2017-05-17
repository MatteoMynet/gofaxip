// This file is part of the GOfax.IP project - https://github.com/MatteoMynet/gofaxip
// Copyright (C) 2014 GONICUS GmbH, Germany - http://www.gonicus.de
//
// This program is free software; you can redistribute it and/or
// modify it under the terms of the GNU General Public License
// as published by the Free Software Foundation; version 2
// of the License.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program; if not, write to the Free Software
// Foundation, Inc., 51 Franklin Street, Fifth Floor, Boston, MA  02110-1301, USA.

package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"
	"syscall"
)

const (
	// NewQfileMode is the default access mode for created queue files
	NewQfileMode = 0660
)

type param struct {
	Tag   string
	Value string
}

// Qfile is a HylaFAX queue file
type Qfile struct {
	filename string
	qfh      *os.File
	params   []param
}

// OpenQfile opens and parses a HylaFAX queue file
func OpenQfile(filename string) (*Qfile, error) {
	var err error

	// Open queue file
	qfh, err := os.OpenFile(filename, os.O_RDWR, 0666)
	if err != nil {
		return nil, err
	}

	q := &Qfile{
		filename: filename,
		qfh:      qfh,
	}

	// Lock queue file using flock (like Hylafax)
	err = syscall.Flock(int(qfh.Fd()), syscall.LOCK_EX)
	if err != nil {
		qfh.Close()
		return nil, err
	}

	// Read tags
	line := 1
	scanner := bufio.NewScanner(qfh)
	for scanner.Scan() {
		parts := strings.SplitN(scanner.Text(), ":", 2)
		if len(parts) != 2 {
			qfh.Close()
			return nil, fmt.Errorf("%s: Error parsing line %d", filename, line)
		}
		q.params = append(q.params, param{strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])})
		line++
	}
	if err = scanner.Err(); err != nil {
		qfh.Close()
		return nil, err
	}

	return q, nil
}

// Close closes an open queue file
func (q *Qfile) Close() error {
	return q.qfh.Close()
}

// Write re-writes an opened queue file
func (q *Qfile) Write() error {
	var err error

	if _, err = q.qfh.Seek(0, 0); err != nil {
		return err
	}

	var bytes int64
	for _, param := range q.params {
		var n int
		if n, err = q.qfh.WriteString(fmt.Sprintf("%s:%s\n", param.Tag, param.Value)); err != nil {
			return err
		}
		bytes = bytes + int64(n)
	}

	if err = q.qfh.Truncate(bytes); err != nil {
		return err
	}

	if err = q.qfh.Sync(); err != nil {
		return err
	}

	return nil
}

// GetAll returns a slice containting all values for
// given tag.
func (q *Qfile) GetAll(tag string) []string {
	var result []string
	for _, param := range q.params {
		if param.Tag == tag {
			result = append(result, param.Value)
		}
	}
	return result
}

// GetFirst returns the value of the first parameter with given tag.
func (q *Qfile) GetFirst(tag string) string {
	for _, param := range q.params {
		if param.Tag == tag {
			return param.Value
		}
	}
	return ""
}

// Set replaces the value of the first found param
// with given value.
// If the param does not exist, it is appended.
func (q *Qfile) Set(tag string, value string) error {
	for i, param := range q.params {
		if param.Tag == tag {
			q.params[i].Value = value
			return nil
		}
	}
	return errors.New("Tag not found")
}

// Add adds a param with given tag and value. If the
// tag already exists, a second one is added.
func (q *Qfile) Add(tag string, value string) {
	q.params = append(q.params, param{tag, value})
}
