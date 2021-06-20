// This file is part of go-getoptions.
//
// Copyright (C) 2015-2021  David Gamba Rios
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package sliceiterator - builds an iterator from a slice to allow peaking for the next value.
package sliceiterator

// Iterator - iterator data
type Iterator struct {
	data *[]string
	idx  int
}

// New - builds a string Iterator
func New(s *[]string) *Iterator {
	return &Iterator{data: s, idx: -1}
}

// Size - returns Iterator size
func (a *Iterator) Size() int {
	return len(*a.data)
}

// Index - return current index.
func (a *Iterator) Index() int {
	return a.idx
}

// Next - moves the index forward and returns a bool to indicate if there is another value.
func (a *Iterator) Next() bool {
	if a.idx < len(*a.data) {
		a.idx++
	}
	return a.idx < len(*a.data)
}

// ExistsNext - tells if there is more data to be read.
func (a *Iterator) ExistsNext() bool {
	return a.idx+1 < len(*a.data)
}

// Value - returns value at current index or an empty string if you are trying to read the value after having fully read the list.
func (a *Iterator) Value() string {
	if a.idx >= len(*a.data) {
		return ""
	}
	return (*a.data)[a.idx]
}

// PeekNextValue - Returns the next value and indicates whether or not it is valid.
func (a *Iterator) PeekNextValue() (string, bool) {
	if a.idx+1 >= len(*a.data) {
		return "", false
	}
	return (*a.data)[a.idx+1], true
}

// IsLast - Tells if the current element is the last.
func (a *Iterator) IsLast() bool {
	return a.idx == len(*a.data)-1
}

// Remaining - Get all remaining values index inclusive.
func (a *Iterator) Remaining() []string {
	if a.idx >= len(*a.data) {
		return []string{}
	}
	return (*a.data)[a.idx:]
}

// Reset - resets the index of the Iterator.
func (a *Iterator) Reset() {
	a.idx = -1
}
