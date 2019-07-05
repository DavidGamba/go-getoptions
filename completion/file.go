// This file is part of go-getoptions.
//
// Copyright (C) 2015-2019  David Gamba Rios
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package completion

import (
	"os"
	"sort"
	"strings"
)

// readDirNoSort - Same as ioutil/ReadDir but doesn't sort results.
//
//   Taken from https://golang.org/src/io/ioutil/ioutil.go
//   Copyright 2009 The Go Authors. All rights reserved.
//   Use of this source code is governed by a BSD-style
//   license that can be found in the LICENSE file.
func readDirNoSort(dirname string) ([]os.FileInfo, error) {
	f, err := os.Open(dirname)
	if err != nil {
		return nil, err
	}
	list, err := f.Readdir(-1)
	f.Close()
	if err != nil {
		return nil, err
	}
	return list, nil
}

// trimLeftDots - Given a string it trims the leading dots (".") and returns a count of how many were removed.
func trimLeftDots(s string) (int, string) {
	charFound := false
	count := 0
	return count, strings.TrimLeftFunc(s, func(r rune) bool {
		if !charFound && r == '.' {
			count++
			return true
		}
		return false
	})
}

// trimLeftDashes - Given a string it trims the leading dashes ("-") and returns a count of how many were removed.
func trimLeftDashes(s string) (int, string) {
	charFound := false
	count := 0
	return count, strings.TrimLeftFunc(s, func(r rune) bool {
		if !charFound && r == '-' {
			count++
			return true
		}
		return false
	})
}

// sortForCompletion - Places hidden files in the same sort possition as their non hidden counterparts.
// Also used for sorting options in the same fashion.
// Example:
//   file.txt
//   .file.txt.~
//   .hidden.txt
//   ..hidden.txt.~
//
//   -d
//   --debug
//   -h
//   --help
func sortForCompletion(list []string) {
	sort.Slice(list,
		func(i, j int) bool {
			an, a := trimLeftDots(list[i])
			bn, b := trimLeftDots(list[j])
			if a == b {
				return an < bn
			}
			an, a = trimLeftDashes(a)
			bn, b = trimLeftDashes(b)
			if a == b {
				return an < bn
			}
			return a < b
		})
}

// listDir - Given a dir and a prefix returns a list of files in the dir filtered by their prefix.
// NOTE: dot (".") is a valid dirname.
func listDir(dirname string, prefix string) ([]string, error) {
	filenames := []string{}
	fileInfoList, err := readDirNoSort(dirname)
	if err != nil {
		return filenames, err
	}
	for _, fi := range fileInfoList {
		name := fi.Name()
		if !strings.HasPrefix(name, prefix) {
			continue
		}
		if fi.IsDir() {
			filenames = append(filenames, name+"/")
		} else {
			filenames = append(filenames, name)
		}
	}
	sortForCompletion(filenames)
	return filenames, err
}

