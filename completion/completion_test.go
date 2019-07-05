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

func TestGetChildNames(t *testing.T) {
	// Tree setup
	rootNode := NewNode("executable", Root, nil)
	rootNode.AddChild(NewNode("options", OptionsNode, []string{"--version", "--help", "-v", "-h"}))

	logNode := NewNode("log", StringNode, nil)
	rootNode.AddChild(logNode)

	loggerNode := NewNode("logger", StringNode, nil)
	rootNode.AddChild(loggerNode)

	showNode := NewNode("show", StringNode, nil)
	showNode.AddChild(NewNode("custom", CustomNode, []string{"--hola", "..hola", "abcd1234", "bbcd/1234"}))
	rootNode.AddChild(showNode)

	sublogNode := NewNode("sublog", StringNode, nil)
	logNode.AddChild(sublogNode)

	logNode.AddChild(NewNode("options", OptionsNode, []string{"--help"}))
	logNode.AddChild(NewNode("test_tree", FileListNode, nil))

	// Test Raw Completions
	tests := []struct {
		name    string
		node    *Node
		prefix  string
		results []string
	}{
		{"get commands", rootNode, "", []string{"log", "logger", "show"}},
		{"get commands", rootNode, "log", []string{"log", "logger"}},
		{"get commands", rootNode, "show", []string{"show"}},
		{"get options", rootNode, "-", []string{"-h", "--help", "-v", "--version"}},
		{"get commands", rootNode.GetChildByName("x"), "", []string{}},
		{"filter out hidden files", rootNode.GetChildByName("log"), "", []string{"sublog", "aFile1", "aFile2", "bDir1/", "bDir2/", "cFile1", "cFile2"}},
		{"show hidden files", rootNode.GetChildByName("log"), ".", []string{"./", "../", ".aFile2", "..aFile2", "...aFile2"}},
		{"show dir contents", rootNode.GetChildByName("log"), "bDir1/", []string{"bDir1/file", "bDir1/.file"}},
		{"show custom output", rootNode.GetChildByName("show"), "", []string{"abcd1234", "bbcd/1234", "..hola", "--hola"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.node.Completions(tt.prefix)
			if !reflect.DeepEqual(got, tt.results) {
				t.Errorf("Completions() got = '%#v', want '%#v'", got, tt.results)
			}
		})
	}

	// Test Completions with CompLine
	compLineTests := []struct {
		name     string
		node     *Node
		compLine string
		results  []string
	}{
		{"nil", rootNode, "", []string{}},
		{"top level", rootNode, "./executable ", []string{"log", "logger", "show"}},
		{"top level", rootNode, "./executable l", []string{"log", "logger"}},
		{"top level", rootNode, "./executable lo", []string{"log", "logger"}},
		{"top level", rootNode, "./executable log", []string{"log", "logger"}},
		{"top level", rootNode, "./executable sh", []string{"show"}},
		{"options", rootNode, "./executable -", []string{"-h", "--help", "-v", "--version"}},
		{"command", rootNode, "./executable log ", []string{"sublog", "aFile1", "aFile2", "bDir1/", "bDir2/", "cFile1", "cFile2"}},
		{"command", rootNode, "./executable show", []string{"abcd1234", "bbcd/1234", "..hola", "--hola"}},
	}
	for _, tt := range compLineTests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.node.CompLineComplete(tt.compLine)
			if !reflect.DeepEqual(got, tt.results) {
				t.Errorf("CompLineComplete() got = '%#v', want '%#v'", got, tt.results)
			}
		})
	}
}
