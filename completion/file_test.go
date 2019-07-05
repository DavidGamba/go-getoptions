// This file is part of go-getoptions.
//
// Copyright (C) 2015-2019  David Gamba Rios
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package completion

import (
	"reflect"
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
		{"dir", "test_tree", "", []string{"aFile1", "aFile2", ".aFile2", "..aFile2", "...aFile2", "bDir1/", "bDir2/", "cFile1", "cFile2"}, ""},
		{"dir", "test_tree", ".", []string{"./", "../", ".aFile2", "..aFile2", "...aFile2"}, ""},
		{"dir", "test_tree", "b", []string{"bDir1/", "bDir2/"}, ""},
		{"error", "x", "", []string{}, "open x: no such file or directory"},
		{"error", "test_tree/aFile1", "", []string{}, "readdirent: not a directory"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotErr := listDir(tt.dirname, tt.prefix)
			if gotErr == nil && tt.err != "" {
				t.Errorf("getFileList() got = '%v', want '%v'", gotErr, tt.err)
			}
			if gotErr != nil && gotErr.Error() != tt.err {
				t.Errorf("getFileList() got = '%v', want '%v'", gotErr, tt.err)
			}
			if !reflect.DeepEqual(got, tt.list) {
				t.Errorf("getFileList() got = %v, want %v", got, tt.list)
			}
		})
	}
}
