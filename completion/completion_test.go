// This file is part of go-getoptions.
//
// Copyright (C) 2015-2020  David Gamba Rios
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package completion

import (
	"bytes"
	"os"
	"reflect"
	"testing"
)

func setupLogging() *bytes.Buffer {
	s := ""
	buf := bytes.NewBufferString(s)
	Debug.SetOutput(buf)
	return buf
}

func treeSetup() *Node {
	rootNode := NewNode("executable", Root, nil)
	rootNode.AddChild(NewNode("options", OptionsNode, []string{"--version", "--help", "-v", "-h"}))
	optionWithCompletion := NewNode("profile", OptionsWithCompletion, []string{"--profile", "-p"})
	optionWithCompletion.OptionCompletions = []string{"development", "production"}
	rootNode.AddChild(optionWithCompletion)
	rootNode.AddChild(NewNode("region", OptionsWithCompletion, []string{"--region", "-r"}))

	logNode := NewNode("log", CommandNode, nil)
	rootNode.AddChild(logNode)

	loggerNode := NewNode("logger", CommandNode, nil)
	loggerNode.AddChild(NewNode("test/test_tree/bDir1", FileListNode, nil))
	rootNode.AddChild(loggerNode)

	showNode := NewNode("show", CommandNode, nil)
	showNode.AddChild(NewNode("custom", CustomNode, []string{"--hola", "..hola", "abcd1234", "bbcd/1234"}))
	rootNode.AddChild(showNode)

	sublogNode := NewNode("sublog", CommandNode, nil)
	logNode.AddChild(sublogNode)

	logNode.AddChild(NewNode("options", OptionsNode, []string{"--help"}))
	logNode.AddChild(NewNode("test/test_tree", FileListNode, nil))
	return rootNode
}

func TestGetChildNames(t *testing.T) {
	Debug.SetOutput(os.Stderr)

	rootNode := treeSetup()
	getNode := func(name string) *Node { n, _ := rootNode.GetChildByName(name); return n }

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
		{"get options", rootNode, "-", []string{"-h", "--help", "-p", "--profile", "-r", "--region", "-v", "--version"}},
		{"get options", rootNode, "-h", []string{"-h"}},
		{"get commands", getNode("x"), "", []string{}},
		{"filter out hidden files", getNode("log"), "", []string{"sublog", "aFile1", "aFile2", "bDir1/", "bDir2/", "cFile1", "cFile2"}},
		{"filter out hidden files", getNode("logger"), "", []string{"file"}},
		{"show hidden files", getNode("log"), ".", []string{"./", "../", ".aFile2", "..aFile2", "...aFile2"}},
		{"show dir contents", getNode("log"), "bDir1/", []string{"bDir1/file", "bDir1/.file"}},
		{"Recurse back", getNode("log"), "..", []string{"../", "..aFile2", "...aFile2"}},
		{"Recurse back", getNode("logger"), "..", []string{"../", "../ "}},
		{"Recurse back", getNode("logger"), "../", []string{"../aFile1", "../aFile2", "../.aFile2", "../..aFile2", "../...aFile2", "../bDir1/", "../bDir2/", "../cFile1", "../cFile2"}},
		{"Recurse back", getNode("logger"), "../.", []string{".././", "../../", "../.aFile2", "../..aFile2", "../...aFile2"}},
		{"Recurse back", getNode("logger"), "../..", []string{"../../", "../..aFile2", "../...aFile2"}},
		{"show dir contents", getNode("logger"), "../.a", []string{"../.aFile2"}},
		{"Full match", getNode("logger"), "../.aFile2", []string{"../.aFile2"}},
		{"show custom output", getNode("show"), "", []string{"abcd1234", "bbcd/1234", "..hola", "--hola"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := setupLogging()
			Debug.Printf("TestGetChildNames - name: %s, prefix: %s\n", tt.name, tt.prefix)
			got := tt.node.Completions(nil, tt.prefix)
			if !reflect.DeepEqual(got, tt.results) {
				t.Errorf("(%s).Completions(%s) got = '%#v', want '%#v'", tt.node.Name, tt.prefix, got, tt.results)
			}
			t.Log(buf.String())
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
		{"top level", rootNode, "./executable  ", []string{"log", "logger", "show"}},
		{"top level", rootNode, "./executable   ", []string{"log", "logger", "show"}},
		{"top level", rootNode, "./executable l", []string{"log", "logger"}},
		{"top level", rootNode, "./executable  l", []string{"log", "logger"}},
		{"top level", rootNode, "./executable   l", []string{"log", "logger"}},
		{"top level", rootNode, "./executable lo", []string{"log", "logger"}},
		{"top level", rootNode, "./executable log", []string{"log", "logger"}},
		{"top level", rootNode, "./executable  log", []string{"log", "logger"}},
		{"top level", rootNode, "./executable sh", []string{"show"}},
		{"options", rootNode, "./executable -", []string{"-h", "--help", "-p", "--profile", "-r", "--region", "-v", "--version"}},
		{"options", rootNode, "./executable -h", []string{"-h"}},
		{"options", rootNode, "./executable -h ", []string{"log", "logger", "show"}},
		{"options", rootNode, "./executable  -h  l", []string{"log", "logger"}},
		{"options", rootNode, "./executable  --help  l", []string{"log", "logger"}},
		{"options", rootNode, "./executable  --profile=qa  l", []string{"log", "logger"}},
		{"options", rootNode, "./executable  --pro", []string{"--profile"}},
		{"options", rootNode, "./executable  --profile", []string{"--profile"}},
		{"options", rootNode, "./executable  --profile=", []string{}},
		{"options", rootNode, "./executable  --profile=qa", []string{}},
		{"options", rootNode, "./executable  --profile qa", []string{"qa"}},
		{"options", rootNode, "./executable  --profile qa  l", []string{"log", "logger"}},
		{"command", rootNode, "./executable log ", []string{"sublog", "aFile1", "aFile2", "bDir1/", "bDir2/", "cFile1", "cFile2"}},
		{"command", rootNode, "./executable log bDir1/f", []string{"bDir1/file"}},
		{"command", rootNode, "./executable log bDir1/file ", []string{"sublog", "aFile1", "aFile2", "bDir1/", "bDir2/", "cFile1", "cFile2"}},
		{"command", rootNode, "./executable log bDir1/file -", []string{"--help"}},
		{"command", rootNode, "./executable   log   bDir1/file  -", []string{"--help"}},
		{"command", rootNode, "./executable logger ../.a", []string{"../.aFile2"}},
		{"command", rootNode, "./executable logger ../.aFile2", []string{"../.aFile2"}},
		{"command", rootNode, "./executable show", []string{"abcd1234", "bbcd/1234", "..hola", "--hola"}},
		{"not a valid arg", rootNode, "./executable dev", []string{}},
	}
	for _, tt := range compLineTests {
		t.Run(tt.name, func(t *testing.T) {
			buf := setupLogging()
			got := tt.node.CompLineComplete(nil, tt.compLine)
			if !reflect.DeepEqual(got, tt.results) {
				t.Errorf("CompLineComplete() got = '%#v', want '%#v'", got, tt.results)
			}
			t.Log(buf.String())
		})
	}
}

func TestOptionsWithCompletion(t *testing.T) {
	Debug.SetOutput(os.Stderr)

	rootNode := treeSetup()

	// Test Completions with CompLine
	compLineTests := []struct {
		name     string
		node     *Node
		compLine string
		results  []string
	}{
		{"options", rootNode, "./executable  --region ", []string{}},
		{"options", rootNode, "./executable  --profile ", []string{"development", "production"}},
		{"options", rootNode, "./executable  --profile d", []string{"development"}},
		{"options", rootNode, "./executable  --profile pro", []string{"production"}},
		{"options", rootNode, "./executable  --profile profile", []string{"profile"}},
		{"options", rootNode, "./executable  --profile  l", []string{"l"}},
		{"options", rootNode, "./executable  --profile=qa  l", []string{"log", "logger"}},
		{"options", rootNode, "./executable  --pro", []string{"--profile"}},
		{"options", rootNode, "./executable  --profile", []string{"--profile"}},
		{"options", rootNode, "./executable  --profile=", []string{}},
		{"options", rootNode, "./executable  --profile=qa", []string{}},
		{"options", rootNode, "./executable  --profile qa", []string{"qa"}},
		{"options", rootNode, "./executable  --profile qa  l", []string{"log", "logger"}},
	}
	for _, tt := range compLineTests {
		t.Run(tt.name, func(t *testing.T) {
			buf := setupLogging()
			got := tt.node.CompLineComplete(nil, tt.compLine)
			if !reflect.DeepEqual(got, tt.results) {
				t.Errorf("CompLineComplete() got = '%#v', want '%#v'", got, tt.results)
			}
			t.Log(buf.String())
		})
	}
}
