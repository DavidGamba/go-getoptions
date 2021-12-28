// This file is part of go-getoptions.
//
// Copyright (C) 2015-2021  David Gamba Rios
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package sliceiterator

import (
	"reflect"
	"testing"
)

func TestIterator(t *testing.T) {
	data := []string{"a", "b", "c", "d"}
	i := New(&data)
	if i.Size() != len(data) {
		t.Errorf("wrong size: %d\n", i.Size())
	}
	if i.Index() != -1 {
		t.Errorf("wrong initial index: %d\n", i.Index())
	}
	for i.Next() {
		if i.Index() < len(data)-1 {
			if !i.ExistsNext() {
				t.Errorf("wrong ExistsNext: idx %d, size %d", i.Index(), i.Size())
			}
		}
		if i.Index() == 0 {
			if i.Value() != "a" {
				t.Errorf("wrong value: %s\n", i.Value())
			}
		}
		if i.Index() == 2 {
			if i.Value() != "c" {
				t.Errorf("wrong value: %s\n", i.Value())
			}
			val, ok := i.PeekNextValue()
			if !ok {
				t.Errorf("wrong next value: %v\n", val)
			}
			if val != "d" {
				t.Errorf("wrong next value: %v\n", val)
			}
			if !reflect.DeepEqual(i.Remaining(), []string{"c", "d"}) {
				t.Errorf("wrong remaining value: %v\n", i.Remaining())
			}
			if i.IsLast() {
				t.Errorf("not last\n")
			}
		}
		if i.Index() == 3 && !i.IsLast() {
			t.Errorf("last not marked properly\n")
		}
	}
	if i.ExistsNext() {
		t.Errorf("wrong ExistsNext: idx %d, size %d", i.Index(), i.Size())
	}
	if i.Next() != false {
		t.Errorf("wrong next return\n")
	}
	if i.Value() != "" {
		t.Errorf("wrong value: %s\n", i.Value())
	}
	if i.Index() != len(data) {
		t.Errorf("wrong final index: %d\n", i.Index())
	}
	val, ok := i.PeekNextValue()
	if ok {
		t.Errorf("wrong next value: %v\n", val)
	}
	if val != "" {
		t.Errorf("wrong next value: %v\n", val)
	}
	if !reflect.DeepEqual(i.Remaining(), []string{}) {
		t.Errorf("wrong remaining value: %v\n", i.Remaining())
	}
	i.Reset()
	if i.Index() != -1 {
		t.Errorf("wrong index after reset: %d\n", i.Index())
	}
}
