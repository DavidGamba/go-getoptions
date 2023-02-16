// This file is part of go-getoptions.
//
// Copyright (C) 2015-2023  David Gamba Rios
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package completion

import (
	"reflect"
	"regexp"
	"testing"
)

func TestListDir(t *testing.T) {
	tests := []struct {
		name    string
		dirname string
		prefix  string
		list    []string
		err     string
	}{
		{"dir", "test/test_tree", "", []string{"aFile1", "aFile2", ".aFile2", "..aFile2", "...aFile2", "bDir1/", "bDir2/", "cFile1", "cFile2"}, ""},
		{"dir", "test/test_tree", ".", []string{"./", "../", ".aFile2", "..aFile2", "...aFile2"}, ""},
		{"dir", "test/test_tree", "b", []string{"bDir1/", "bDir2/"}, ""},
		{"dir", "test/test_tree", "bDir1", []string{"bDir1/", "bDir1/ "}, ""},
		{"dir", "test/test_tree", "bDir1/", []string{"bDir1/file", "bDir1/.file"}, ""},
		{"dir", "test/test_tree", "bDir1/f", []string{"bDir1/file"}, ""},
		{"dir", "test/test_tree/bDir1", "../", []string{"../aFile1", "../aFile2", "../.aFile2", "../..aFile2", "../...aFile2", "../bDir1/", "../bDir2/", "../cFile1", "../cFile2"}, ""},
		{"dir", "test/test_tree/bDir1", "../.", []string{".././", "../../", "../.aFile2", "../..aFile2", "../...aFile2"}, ""},
		{"error", "x", "", []string{}, "open x: no such file or directory"},
		{"error", "test/test_tree/aFile1", "", []string{}, "not a directory"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotErr := listDir(tt.dirname, tt.prefix)
			if gotErr == nil && tt.err != "" {
				t.Errorf("getFileList() got = '%v', want '%v'", gotErr, tt.err)
			}
			r, err := regexp.Compile(tt.err)
			if err != nil {
				t.Fatalf("bad regex in test: %s", err)
			}
			if gotErr != nil && !r.MatchString(gotErr.Error()) {
				t.Errorf("getFileList() got = '%s', want '%s'", gotErr.Error(), tt.err)
			}
			if !reflect.DeepEqual(got, tt.list) {
				t.Errorf("getFileList() got = %v, want %v", got, tt.list)
			}
		})
	}
}

func TestSortForCompletion(t *testing.T) {
	tests := []struct {
		name   string
		list   []string
		sorted []string
	}{
		{"basic", []string{"b", "a"}, []string{"a", "b"}},
		{"level up", []string{"..", ".", "a"}, []string{".", "..", "a"}},
		{"level up", []string{".", "..", "a"}, []string{".", "..", "a"}},
		{"level up", []string{"a", ".", ".."}, []string{".", "..", "a"}},
		{"level up", []string{"../", "./"}, []string{"./", "../"}},
		{"level up", []string{"../../", ".././"}, []string{".././", "../../"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sortForCompletion(tt.list)
			if !reflect.DeepEqual(tt.list, tt.sorted) {
				t.Errorf("sortForCompletion() got = %v, want %v", tt.list, tt.sorted)
			}
		})
	}
}
