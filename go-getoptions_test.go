// This file is part of go-getoptions.
//
// Copyright (C) 2015-2021  David Gamba Rios
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
package getoptions

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/DavidGamba/go-getoptions/option"
	"github.com/DavidGamba/go-getoptions/text"
)

func firstDiff(got, expected string) string {
	same := ""
	for i, gc := range got {
		if len([]rune(expected)) <= i {
			return fmt.Sprintf("Index: %d | diff: got '%s' - exp '%s'\n", len(expected), got, expected)
		}
		if gc != []rune(expected)[i] {
			return fmt.Sprintf("Index: %d | diff: got '%c' - exp '%c'\n%s\n", i, gc, []rune(expected)[i], same)
		}
		same += string(gc)
	}
	if len(expected) > len(got) {
		return fmt.Sprintf("Index: %d | diff: got '%s' - exp '%s'\n", len(got), got, expected)
	}
	return ""
}

func setupLogging() *bytes.Buffer {
	s := ""
	buf := bytes.NewBufferString(s)
	Debug.SetOutput(buf)
	option.Debug.SetOutput(buf)
	return buf
}

// Verifies that a panic is reached when the same option is defined twice.
func TestDuplicateDefinition(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Duplicate definition did not panic")
		}
	}()
	opt := New()
	opt.Bool("flag", false)
	opt.Bool("flag", false)
}

// Verifies that a panic is reached when the same alias is defined twice.
func TestDuplicateAlias(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Duplicate alias definition did not panic")
		}
	}()
	opt := New()
	opt.Bool("flag", false, opt.Alias("t"))
	opt.Bool("bool", false, opt.Alias("t"))
}

// Verifies that a panic is reached when an alias is named after an option.
func TestAliasMatchesOption(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Duplicate alias definition did not panic")
		}
	}()
	opt := New()
	opt.Bool("flag", false)
	opt.Bool("bool", false, opt.Alias("flag"))
}

func TestRequired(t *testing.T) {
	opt := New()
	opt.Bool("flag", false, opt.Required())
	_, err := opt.Parse([]string{"--flag"})
	if err != nil {
		t.Errorf("Required option called but error raised")
	}

	opt = New()
	opt.Bool("flag", false, opt.Required())
	_, err = opt.Parse([]string{})
	if err == nil {
		t.Errorf("Required option missing didn't raise error")
	}
	if err != nil && err.Error() != "Missing required option 'flag'!" {
		t.Errorf("Error string didn't match expected value")
	}

	opt = New()
	opt.Bool("flag", false, opt.Required("Missing --flag!"))
	_, err = opt.Parse([]string{})
	if err == nil {
		t.Errorf("Required option missing didn't raise error")
	}
	if err != nil && err.Error() != "Missing --flag!" {
		t.Errorf("Error string didn't match expected value")
	}
}

// TODO
func TestUnknownOptionModes(t *testing.T) {
	// Default
	opt := New()
	_, err := opt.Parse([]string{"--flags"})
	if err == nil {
		t.Errorf("Unknown option 'flags' didn't raise error")
	}
	if err != nil && err.Error() != "Unknown option 'flags'" {
		t.Errorf("Error string didn't match expected value: %s\n", err)
	}

	opt = New()
	opt.SetUnknownMode(Fail)
	_, err = opt.Parse([]string{"--flags"})
	if err == nil {
		t.Errorf("Unknown option 'flags' didn't raise error")
	}
	if err != nil && err.Error() != "Unknown option 'flags'" {
		t.Errorf("Error string didn't match expected value")
	}

	buf := new(bytes.Buffer)
	opt = New()
	opt.Writer = buf
	opt.SetUnknownMode(Warn)
	remaining, err := opt.Parse([]string{"--flags", "--flegs"})
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if buf.String() !=
		fmt.Sprintf("WARNING: "+text.MessageOnUnknown+"\nWARNING: "+text.MessageOnUnknown+"\n", "flags", "flegs") {
		t.Errorf("Warning message didn't match expected value: %s", buf.String())
	}
	if !reflect.DeepEqual(remaining, []string{"--flags", "--flegs"}) {
		t.Errorf("remaining didn't have expected value: %v != %v", remaining, []string{"--flags", "--flegs"})
	}

	// Tests first unknown argument as a passthrough
	buf = new(bytes.Buffer)
	opt = New()
	opt.Writer = buf
	opt.SetUnknownMode(Pass)
	remaining, err = opt.Parse([]string{"--flags"})
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if buf.String() != "" {
		t.Errorf("output didn't match expected value: %s", buf.String())
	}
	if !reflect.DeepEqual(remaining, []string{"--flags"}) {
		t.Errorf("remaining didn't have expected value: %v != %v", remaining, []string{"--flags"})
	}

	// Tests first unknown argument as a passthrough with a known one after
	buf = new(bytes.Buffer)
	opt = New()
	opt.Writer = buf
	opt.Bool("known", false)
	opt.Bool("another", false)
	opt.SetUnknownMode(Pass)
	remaining, err = opt.Parse([]string{"--flags", "--known", "--another", "--unknown"})
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if buf.String() != "" {
		t.Errorf("output didn't match expected value: %s", buf.String())
	}
	if !reflect.DeepEqual(remaining, []string{"--flags", "--unknown"}) {
		t.Errorf("remaining didn't have expected value: %v != %v", remaining, []string{"--flags", "--unknown"})
	}
	if !opt.Called("known") && !opt.Called("another") {
		t.Errorf("known or another were not called")
	}
}

func TestSetRequireOrder(t *testing.T) {
	buf := new(bytes.Buffer)
	opt := New()
	opt.Writer = buf
	opt.String("opt", "")
	opt.Bool("help", false)
	opt.SetRequireOrder()
	remaining, err := opt.Parse([]string{"--opt", "arg", "subcommand", "--help"})
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if buf.String() != "" {
		t.Errorf("output didn't match expected value: %s", buf.String())
	}
	if !reflect.DeepEqual(remaining, []string{"subcommand", "--help"}) {
		t.Errorf("remaining didn't have expected value: %v != %v", remaining, []string{"subcommand", "--help"})
	}
	if opt.Called("help") {
		t.Errorf("help called when it wasn't supposed to")
	}

	// Tests requireOrder with PassThrough
	buf = new(bytes.Buffer)
	opt = New()
	opt.Writer = buf
	opt.Bool("known", false)
	opt.Bool("another", false)
	opt.SetUnknownMode(Pass)
	opt.SetRequireOrder()
	remaining, err = opt.Parse([]string{"--flags", "--known", "--another", "--unknown"})
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if buf.String() != "" {
		t.Errorf("output didn't match expected value: %s", buf.String())
	}
	if !reflect.DeepEqual(remaining, []string{"--flags", "--known", "--another", "--unknown"}) {
		t.Errorf("remaining didn't have expected value: %v != %v", remaining, []string{"--flags", "--known", "--another", "--unknown"})
	}

	buf = new(bytes.Buffer)
	opt = New()
	opt.Writer = buf
	opt.Bool("known", false)
	opt.Bool("another", false)
	opt.SetUnknownMode(Pass)
	opt.SetRequireOrder()
	remaining, err = opt.Parse([]string{"--known", "--flags", "--another", "--unknown"})
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if buf.String() != "" {
		t.Errorf("output didn't match expected value: %s", buf.String())
	}
	if !reflect.DeepEqual(remaining, []string{"--flags", "--another", "--unknown"}) {
		t.Errorf("remaining didn't have expected value: %v != %v", remaining, []string{"--flags", "--another", "--unknown"})
	}
	if !opt.Called("known") {
		t.Errorf("known was not called")
	}
}

func TestOptionals(t *testing.T) {
	// Missing argument without default
	opt := New()
	opt.String("string", "")
	_, err := opt.Parse([]string{"--string"})
	if err == nil {
		t.Errorf("Missing argument for option 'string' didn't raise error")
	}
	if err != nil && err.Error() != fmt.Sprintf(text.ErrorMissingArgument, "string") {
		t.Errorf("Error string didn't match expected value")
	}

	// Missing argument with default
	opt = New()
	opt.StringOptional("string", "default", opt.Alias("alias"))
	_, err = opt.Parse([]string{"--string"})
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if opt.Value("string") != "default" {
		t.Errorf("Default value not set for 'string'")
	}

	opt = New()
	opt.IntOptional("int", 123, opt.Alias("alias"))
	_, err = opt.Parse([]string{"--int"})
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if opt.Value("int") != 123 {
		t.Errorf("Default value not set for 'int'")
	}

	// Missing argument, next argument is option
	opt = New()
	opt.StringOptional("string", "default")
	opt.IntOptional("int", 123)
	_, err = opt.Parse([]string{"--string", "--int"})
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if opt.Value("string") != "default" {
		t.Errorf("Default value not set for 'string'")
	}
	opt = New()
	opt.StringOptional("string", "default")
	opt.IntOptional("int", 123)
	_, err = opt.Parse([]string{"--int", "--string"})
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if opt.Value("int") != 123 {
		t.Errorf("Default value not set for 'int'")
	}

	// Argument given
	opt = New()
	opt.StringOptional("string", "default")
	opt.IntOptional("int", 123)
	_, err = opt.Parse([]string{"--string=arg", "--int=456"})
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if opt.Value("string") != "arg" {
		t.Errorf("string Optional didn't take argument")
	}
	if opt.Value("int") != 456 {
		t.Errorf("int Optional didn't take argument")
	}
	opt = New()
	opt.StringOptional("string", "default")
	opt.IntOptional("int", 123)
	_, err = opt.Parse([]string{"--string", "arg", "--int", "456"})
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if opt.Value("string") != "arg" {
		t.Errorf("string Optional didn't take argument")
	}
	if opt.Value("int") != 456 {
		t.Errorf("int Optional didn't take argument")
	}

	// VarOptional
	var result string
	var i int
	opt = New()
	opt.StringVarOptional(&result, "string", "default")
	opt.IntVarOptional(&i, "int", 123)
	_, err = opt.Parse([]string{"--string=arg", "--int=456"})
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if result != "arg" {
		t.Errorf("StringVarOptional didn't take argument")
	}
	if i != 456 {
		t.Errorf("IntVarOptional didn't take argument")
	}

	result = ""
	opt = New()
	opt.StringVarOptional(&result, "string", "default")
	_, err = opt.Parse([]string{"--string=arg"})
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if result != "arg" {
		t.Errorf("StringVarOptional didn't take argument")
	}

	i = 0
	opt = New()
	opt.IntVarOptional(&i, "int", 123)
	_, err = opt.Parse([]string{"--int=456"})
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if i != 456 {
		t.Errorf("IntVarOptional didn't take argument")
	}

	result = ""
	opt = New()
	opt.StringVarOptional(&result, "string", "default")
	_, err = opt.Parse([]string{"--string"})
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if result != "default" {
		t.Errorf("Default value not set for 'string'")
	}

	i = 0
	opt = New()
	opt.IntVarOptional(&i, "int", 123)
	_, err = opt.Parse([]string{"--int"})
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if i != 123 {
		t.Errorf("Default value not set for 'int'")
	}

	// Cast errors
	opt = New()
	opt.IntOptional("int", 0)
	_, err = opt.Parse([]string{"--int=hello"})
	if err == nil {
		t.Errorf("Int cast didn't raise errors")
	}
	if err != nil && err.Error() != fmt.Sprintf(text.ErrorConvertToInt, "int", "hello") {
		t.Errorf("Error string didn't match expected value '%s'", err)
	}

	opt = New()
	opt.IntOptional("int", 0)
	_, err = opt.Parse([]string{"--int", "hello"})
	if err == nil {
		t.Errorf("Int cast didn't raise errors")
	}
	if err != nil && err.Error() != fmt.Sprintf(text.ErrorConvertToInt, "int", "hello") {
		t.Errorf("Error string didn't match expected value '%s'", err)
	}
}

func TestGetOptBool(t *testing.T) {
	setup := func() *GetOpt {
		opt := New()
		opt.Bool("flag", false)
		return opt
	}

	cases := []struct {
		opt    *GetOpt
		option string
		input  []string
		value  bool
	}{
		{setup(),
			"flag",
			[]string{"--flag"},
			true,
		},
	}
	for _, c := range cases {
		_, err := c.opt.Parse(c.input)
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		if c.opt.Value(c.option) != c.value {
			t.Errorf("Wrong value: %v != %v", c.opt.Value(c.option), c.value)
		}
	}

	// Test case sensitivity
	opt := New()
	opt.Bool("v", false)
	opt.Bool("V", false)
	_, err := opt.Parse([]string{"-v"})
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if !opt.Called("v") {
		t.Errorf("v didn't have expected value %v", false)
	}
	if opt.Called("V") {
		t.Errorf("V didn't have expected value %v", true)
	}
	opt = New()
	opt.Bool("v", false)
	opt.Bool("V", false)
	_, err = opt.Parse([]string{"-V"})
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if !opt.Called("V") {
		t.Errorf("V didn't have expected value %v", false)
	}
	if opt.Called("v") {
		t.Errorf("v didn't have expected value %v", true)
	}
}

func TestCalled(t *testing.T) {
	opt := New()
	opt.Bool("hello", false)
	opt.Bool("happy", false)
	opt.Bool("world", false)
	opt.String("string", "")
	opt.String("string2", "")
	opt.Int("int", 0)
	opt.Int("int2", 0)
	_, err := opt.Parse([]string{"--hello", "--world", "--string2", "str", "--int2", "123"})
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if !opt.Called("hello") {
		t.Errorf("hello didn't have expected value %v", false)
	}
	if opt.Called("happy") {
		t.Errorf("happy didn't have expected value %v", true)
	}
	if !opt.Called("world") {
		t.Errorf("world didn't have expected value %v", false)
	}
	if opt.Called("string") {
		t.Errorf("string didn't have expected value %v", true)
	}
	if !opt.Called("string2") {
		t.Errorf("string2 didn't have expected value %v", false)
	}
	if opt.Called("int") {
		t.Errorf("int didn't have expected value %v", true)
	}
	if !opt.Called("int2") {
		t.Errorf("int2 didn't have expected value %v", false)
	}
	if opt.Called("unknown") {
		t.Errorf("unknown didn't have expected value %v", false)
	}
}

func TestCalledAs(t *testing.T) {
	opt := New()
	opt.Bool("flag", false, opt.Alias("f", "hello"))
	_, err := opt.Parse([]string{"--flag"})
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if opt.CalledAs("flag") != "flag" {
		t.Errorf("Wrong CalledAs! got: %s, expected: %s", opt.CalledAs("flag"), "flag")
	}

	opt = New()
	opt.Bool("flag", false, opt.Alias("f", "hello"))
	_, err = opt.Parse([]string{"--hello"})
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if opt.CalledAs("flag") != "hello" {
		t.Errorf("Wrong CalledAs! got: %s, expected: %s", opt.CalledAs("flag"), "hello")
	}

	opt = New()
	opt.Bool("flag", false, opt.Alias("f", "hello"))
	_, err = opt.Parse([]string{"--h"})
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if opt.CalledAs("flag") != "hello" {
		t.Errorf("Wrong CalledAs! got: %s, expected: %s", opt.CalledAs("flag"), "hello")
	}

	opt = New()
	opt.Bool("flag", false, opt.Alias("f", "hello"))
	_, err = opt.Parse([]string{})
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if opt.CalledAs("flag") != "" {
		t.Errorf("Wrong CalledAs! got: %s, expected: %s", opt.CalledAs("flag"), "")
	}

	opt = New()
	opt.Bool("flag", false, opt.Alias("f", "hello"))
	_, err = opt.Parse([]string{})
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if opt.CalledAs("x") != "" {
		t.Errorf("Wrong CalledAs! got: %s, expected: %s", opt.CalledAs("x"), "")
	}

	opt = New()
	opt.StringSlice("list", 1, 1, opt.Alias("array", "slice"))
	_, err = opt.Parse([]string{"--list=list", "--array=array", "--slice=slice"})
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if opt.CalledAs("list") != "slice" {
		t.Errorf("Wrong CalledAs! got: %s, expected: %s", opt.CalledAs("list"), "slice")
	}
}

func TestEndOfParsing(t *testing.T) {
	opt := New()
	opt.Bool("hello", false)
	opt.Bool("world", false)
	remaining, err := opt.Parse([]string{"hola", "--hello", "--", "mundo", "--world"})
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}

	if !reflect.DeepEqual(remaining, []string{"hola", "mundo", "--world"}) {
		t.Errorf("remaining didn't have expected value: %v != %v", remaining, []string{"hola", "mundo", "--world"})
	}
}

func TestGetOptAliases(t *testing.T) {
	setup := func() *GetOpt {
		opt := New()
		opt.Bool("flag", false, opt.Alias("f", "h"))
		return opt
	}

	cases := []struct {
		opt    *GetOpt
		option string
		input  []string
		value  bool
	}{
		{setup(),
			"flag",
			[]string{"--flag"},
			true,
		},
		{setup(),
			"flag",
			[]string{"-f"},
			true,
		},
		{setup(),
			"flag",
			[]string{"-h"},
			true,
		},
		// TODO: Add flag to allow for this.
		{setup(),
			"flag",
			[]string{"--fl"},
			true,
		},
	}
	for _, c := range cases {
		_, err := c.opt.Parse(c.input)
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		if c.opt.Value(c.option) != c.value {
			t.Errorf("Wrong value: %v != %v", c.opt.Value(c.option), c.value)
		}
	}

	opt := New()
	opt.Bool("flag", false)
	opt.Bool("fleg", false)
	_, err := opt.Parse([]string{"--fl"})
	if err == nil {
		t.Errorf("Ambiguous argument 'fl' didn't raise unknown option error")
	}
	if err != nil && err.Error() != fmt.Sprintf(text.ErrorAmbiguousArgument, "fl", []string{"flag", "fleg"}) {
		t.Errorf("Error string didn't match expected value: %s", err)
	}

	// Bug: Startup panic when alias matches the beginning of preexisting option
	// https://github.com/DavidGamba/go-getoptions/issues/1
	opt = New()
	opt.Bool("fleg", false)
	opt.Bool("flag", false, opt.Alias("f"))
	_, err = opt.Parse([]string{"--f"})
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if opt.Called("fleg") {
		t.Errorf("fleg should not have been called")
	}
	if !opt.Called("flag") {
		t.Errorf("flag not called")
	}

	opt = New()
	opt.Int("flag", 0, opt.Alias("f", "h"))
	_, err = opt.Parse([]string{"--h"})
	if err == nil {
		t.Errorf("Int didn't raise errors")
	}
	if err != nil && err.Error() != fmt.Sprintf(text.ErrorMissingArgument, "h") {
		t.Errorf("Error string didn't match expected value '%s'", err)
	}
}

func TestGetOptString(t *testing.T) {
	setup := func() *GetOpt {
		opt := New()
		opt.String("string", "", opt.Alias("alias"))
		return opt
	}

	cases := []struct {
		opt    *GetOpt
		option string
		input  []string
		value  string
	}{
		{setup(),
			"string",
			[]string{"--string=hello"},
			"hello",
		},
		{setup(),
			"string",
			[]string{"--string=hello", "world"},
			"hello",
		},
		{setup(),
			"string",
			[]string{"--string", "hello"},
			"hello",
		},
		{setup(),
			"string",
			[]string{"--alias", "hello"},
			"hello",
		},
		{setup(),
			"string",
			[]string{"--string", "hello", "world"},
			"hello",
		},
		// String should only accept an option looking string as an argument when passed after =
		{setup(),
			"string",
			[]string{"--string=--hello", "world"},
			"--hello",
		},
		// TODO: Set up a flag to decide whether or not to err on this
		// To have the definition of string overridden. This should probably fail since it is most likely not what the user intends.
		{setup(),
			"string",
			[]string{"--string", "hello", "--string", "world"},
			"world",
		},
	}
	for _, c := range cases {
		_, err := c.opt.Parse(c.input)
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		if c.opt.Value(c.option) != c.value {
			t.Errorf("Wrong value: %v != %v", c.opt.Value(c.option), c.value)
		}
	}

	opt := New()
	opt.String("string", "")
	_, err := opt.Parse([]string{"--string", "--hello"})
	if err == nil {
		t.Errorf("Passing option where argument expected didn't raise error")
	}
	if err != nil && err.Error() != fmt.Sprintf(text.ErrorArgumentWithDash, "string") {
		t.Errorf("Error string didn't match expected value")
	}
}

func TestGetOptInt(t *testing.T) {
	setup := func() *GetOpt {
		opt := New()
		opt.Int("int", 0, opt.Alias("alias"))
		return opt
	}

	cases := []struct {
		opt    *GetOpt
		option string
		input  []string
		value  int
	}{
		{setup(),
			"int",
			[]string{"--int=-123"},
			-123,
		},
		{setup(),
			"int",
			[]string{"--int=-123", "world"},
			-123,
		},
		{setup(),
			"int",
			[]string{"--int", "123"},
			123,
		},
		{setup(),
			"int",
			[]string{"--alias", "123"},
			123,
		},
		{setup(),
			"int",
			[]string{"--int", "123", "world"},
			123,
		},
	}
	for _, c := range cases {
		_, err := c.opt.Parse(c.input)
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		if c.opt.Value(c.option) != c.value {
			t.Errorf("Wrong value: %v != %v", c.opt.Value(c.option), c.value)
		}
	}

	// Missing Argument errors
	opt := New()
	opt.Int("int", 0)
	_, err := opt.Parse([]string{"--int"})
	if err == nil {
		t.Errorf("Int didn't raise errors")
	}
	if err != nil && err.Error() != fmt.Sprintf(text.ErrorMissingArgument, "int") {
		t.Errorf("Error string didn't match expected value '%s'", err)
	}

	// Cast errors
	opt = New()
	opt.Int("int", 0)
	_, err = opt.Parse([]string{"--int=hello"})
	if err == nil {
		t.Errorf("Int cast didn't raise errors")
	}
	if err != nil && err.Error() != fmt.Sprintf(text.ErrorConvertToInt, "int", "hello") {
		t.Errorf("Error string didn't match expected value '%s'", err)
	}

	opt = New()
	opt.Int("int", 0)
	_, err = opt.Parse([]string{"--int", "hello"})
	if err == nil {
		t.Errorf("Int cast didn't raise errors")
	}
	if err != nil && err.Error() != fmt.Sprintf(text.ErrorConvertToInt, "int", "hello") {
		t.Errorf("Error string didn't match expected value '%s'", err)
	}

	opt = New()
	opt.Int("int", 0)
	_, err = opt.Parse([]string{"--int", "-123"})
	if err == nil {
		t.Errorf("Passing option where argument expected didn't raise error")
	}
	if err != nil && err.Error() != fmt.Sprintf(text.ErrorArgumentWithDash, "int") {
		t.Errorf("Error string didn't match expected value: %s", err.Error())
	}
}

func TestGetOptFloat64(t *testing.T) {
	setup := func() *GetOpt {
		opt := New()
		opt.Float64("float", 0, opt.Alias("alias"))
		return opt
	}

	cases := []struct {
		opt    *GetOpt
		option string
		input  []string
		value  float64
	}{
		{setup(),
			"float",
			[]string{"--float=-1.23"},
			-1.23,
		},
		{setup(),
			"float",
			[]string{"--float=-1.23", "world"},
			-1.23,
		},
		{setup(),
			"float",
			[]string{"--float", "1.23"},
			1.23,
		},
		{setup(),
			"float",
			[]string{"--alias", "1.23"},
			1.23,
		},
		{setup(),
			"float",
			[]string{"--float", "1.23", "world"},
			1.23,
		},
	}
	for _, c := range cases {
		_, err := c.opt.Parse(c.input)
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		if c.opt.Value(c.option) != c.value {
			t.Errorf("Wrong value: %v != %v", c.opt.Value(c.option), c.value)
		}
	}

	// Missing Argument errors
	opt := New()
	opt.Float64("float", 0)
	_, err := opt.Parse([]string{"--float"})
	if err == nil {
		t.Errorf("Float64 didn't raise errors")
	}
	if err != nil && err.Error() != fmt.Sprintf(text.ErrorMissingArgument, "float") {
		t.Errorf("Error string didn't match expected value '%s'", err)
	}

	// Cast errors
	opt = New()
	opt.Float64("float", 0)
	_, err = opt.Parse([]string{"--float=hello"})
	if err == nil {
		t.Errorf("Float cast didn't raise errors")
	}
	if err != nil && err.Error() != fmt.Sprintf(text.ErrorConvertToFloat64, "float", "hello") {
		t.Errorf("Error string didn't match expected value '%s'", err)
	}

	opt = New()
	opt.Float64("float", 0)
	_, err = opt.Parse([]string{"--float", "hello"})
	if err == nil {
		t.Errorf("Int cast didn't raise errors")
	}
	if err != nil && err.Error() != fmt.Sprintf(text.ErrorConvertToFloat64, "float", "hello") {
		t.Errorf("Error string didn't match expected value '%s'", err)
	}

	opt = New()
	opt.Float64("float", 0)
	_, err = opt.Parse([]string{"--float", "-123"})
	if err == nil {
		t.Errorf("Passing option where argument expected didn't raise error")
	}
	if err != nil && err.Error() != fmt.Sprintf(text.ErrorArgumentWithDash, "float") {
		t.Errorf("Error string didn't match expected value: %s", err.Error())
	}
}

// TODO: Allow passing : as the map divider
func TestGetOptStringMap(t *testing.T) {
	setup := func() *GetOpt {
		opt := New()
		opt.StringMap("string", 1, 3, opt.Alias("alias"))
		opt.String("opt", "")
		return opt
	}

	cases := []struct {
		opt    *GetOpt
		option string
		input  []string
		value  map[string]string
	}{
		{setup(),
			"string",
			[]string{"--string=hello=world"},
			map[string]string{"hello": "world"},
		},
		{setup(),
			"string",
			[]string{"--alias=hello=world"},
			map[string]string{"hello": "world"},
		},
		{setup(),
			"string",
			[]string{"--string=hello=happy", "world"},
			map[string]string{"hello": "happy"},
		},
		{setup(),
			"string",
			[]string{"--string", "hello=world", "--opt", "happy"},
			map[string]string{"hello": "world"},
		},
		{setup(),
			"string",
			[]string{"--string", "hello=happy", "world"},
			map[string]string{"hello": "happy"},
		},
		{setup(),
			"string",
			[]string{"--string=--hello=happy", "world"},
			map[string]string{"--hello": "happy"},
		},
		{setup(),
			"string",
			[]string{"--string", "hello=world", "--string", "key=value", "--string", "key2=value2"},
			map[string]string{"hello": "world", "key": "value", "key2": "value2"},
		},
		{setup(),
			"string",
			[]string{"--string", "hello=happy", "happy=world"},
			map[string]string{"hello": "happy", "happy": "world"},
		},
		{setup(),
			"string",
			[]string{"--string=--hello=happy", "happy=world"},
			map[string]string{"--hello": "happy", "happy": "world"},
		},
		{setup(),
			"string",
			[]string{"--string", "hello=world", "key=value", "key2=value2"},
			map[string]string{"hello": "world", "key": "value", "key2": "value2"},
		},
		{setup(),
			"string",
			[]string{"--string", "key=value", "Key=value1", "kEy=value2"},
			map[string]string{"key": "value", "Key": "value1", "kEy": "value2"},
		},
	}
	for _, c := range cases {
		_, err := c.opt.Parse(c.input)
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		if !reflect.DeepEqual(c.opt.Value(c.option), c.value) {
			t.Errorf("Wrong value: %v != %v", c.opt.Value(c.option), c.value)
		}
	}
	opt := New()
	opt.StringMap("string", 1, 3)
	_, err := opt.Parse([]string{"--string", "hello"})
	if err != nil && err.Error() != fmt.Sprintf(text.ErrorArgumentIsNotKeyValue, "string") {
		t.Errorf("Error string didn't match expected value: %s", err.Error())
	}
	opt = New()
	opt.StringMap("string", 1, 3)
	_, err = opt.Parse([]string{"--string=hello"})
	if err != nil && err.Error() != fmt.Sprintf(text.ErrorArgumentIsNotKeyValue, "string") {
		t.Errorf("Error string didn't match expected value: %s", err.Error())
	}
	opt = New()
	opt.StringMap("string", 1, 3)
	_, err = opt.Parse([]string{"--string"})
	if err == nil {
		t.Errorf("Missing argument for option 'string' didn't raise error")
	}
	if err != nil && err.Error() != fmt.Sprintf(text.ErrorMissingArgument, "string") {
		t.Errorf("Error string didn't match expected value: %s", err.Error())
	}
	opt = New()
	opt.StringMap("string", 1, 3)
	_, err = opt.Parse([]string{"--string", "--hello=happy", "world"})
	if err == nil {
		t.Errorf("Missing argument for option 'string' didn't raise error")
	}
	if err != nil && err.Error() != fmt.Sprintf(text.ErrorArgumentWithDash, "string") {
		t.Errorf("Error string didn't match expected value: %s", err.Error())
	}

	opt = New()
	sm := opt.StringMap("string", 1, 3)
	_, err = opt.Parse([]string{"--string", "hello=world", "key=value", "key2=value2"})
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if !reflect.DeepEqual(map[string]string{"hello": "world", "key": "value", "key2": "value2"}, sm) {
		t.Errorf("Wrong value: %v != %v", map[string]string{"hello": "world", "key": "value", "key2": "value2"}, sm)
	}
	if sm["hello"] != "world" || sm["key"] != "value" || sm["key2"] != "value2" {
		t.Errorf("Wrong value: %v", sm)
	}
	var m map[string]string
	opt = New()
	opt.StringMapVar(&m, "string", 1, 3)
	_, err = opt.Parse([]string{"--string", "hello=world", "key=value", "key2=value2"})
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if !reflect.DeepEqual(map[string]string{"hello": "world", "key": "value", "key2": "value2"}, sm) {
		t.Errorf("Wrong value: %v != %v", map[string]string{"hello": "world", "key": "value", "key2": "value2"}, sm)
	}
	if sm["hello"] != "world" || sm["key"] != "value" || sm["key2"] != "value2" {
		t.Errorf("Wrong value: %v", sm)
	}

	opt = New()
	opt.StringMap("string", 2, 3)
	_, err = opt.Parse([]string{"--string", "hello=world"})
	if err == nil {
		t.Errorf("Passing less than min didn't raise error")
	}
	if err != nil && err.Error() != fmt.Sprintf(text.ErrorMissingArgument, "string") {
		t.Errorf("Error string didn't match expected value: %s", err.Error())
	}

	opt = New()
	opt.StringMap("string", 2, 3)
	_, err = opt.Parse([]string{"--string", "hello=world", "happy"})
	if err == nil {
		t.Errorf("Passing less than min didn't raise error")
	}
	if err != nil && err.Error() != fmt.Sprintf(text.ErrorArgumentIsNotKeyValue, "string") {
		t.Errorf("Error string didn't match expected value: %s", err.Error())
	}

	opt = New()
	opt.SetMapKeysToLower()
	sm = opt.StringMap("string", 1, 3)
	_, err = opt.Parse([]string{"--string", "Key1=value1", "kEy2=value2", "keY3=value3"})
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if !reflect.DeepEqual(map[string]string{"key1": "value1", "key2": "value2", "key3": "value3"}, sm) {
		t.Errorf("Wrong value: %v != %v", map[string]string{"key1": "value1", "key2": "value2", "key3": "value3"}, sm)
	}
}

func TestGetOptStringSlice(t *testing.T) {
	setup := func() *GetOpt {
		opt := New()
		opt.StringSlice("string", 1, 3, opt.Alias("alias"))
		opt.String("opt", "")
		return opt
	}
	cases := []struct {
		opt    *GetOpt
		option string
		input  []string
		value  []string
	}{
		{setup(),
			"string",
			[]string{"--string", "hello"},
			[]string{"hello"},
		},
		{setup(),
			"string",
			[]string{"--string=hello"},
			[]string{"hello"},
		},
		{setup(),
			"string",
			[]string{"--string", "hello", "world"},
			[]string{"hello", "world"},
		},
		{setup(),
			"string",
			[]string{"--alias", "hello", "world"},
			[]string{"hello", "world"},
		},
		{setup(),
			"string",
			[]string{"--string=hello", "world"},
			[]string{"hello", "world"},
		},
		{setup(),
			"string",
			[]string{"--string", "hello", "happy", "world"},
			[]string{"hello", "happy", "world"},
		},
		{setup(),
			"string",
			[]string{"--string=hello", "happy", "world"},
			[]string{"hello", "happy", "world"},
		},
		{setup(),
			"string",
			[]string{"--string", "hello", "--opt", "world"},
			[]string{"hello"},
		},
		{setup(),
			"string",
			[]string{"--string", "hello", "--string", "world"},
			[]string{"hello", "world"},
		},
	}
	for _, c := range cases {
		_, err := c.opt.Parse(c.input)
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		if !reflect.DeepEqual(c.opt.Value(c.option), c.value) {
			t.Errorf("Wrong value: %v != %v", c.opt.Value(c.option), c.value)
		}
	}

	opt := New()
	opt.StringSlice("string", 2, 3)
	_, err := opt.Parse([]string{"--string", "hello"})
	if err == nil {
		t.Errorf("Passing less than min didn't raise error")
	}
	if err != nil && err.Error() != fmt.Sprintf(text.ErrorMissingArgument, "string") {
		t.Errorf("Error string didn't match expected value")
	}

	opt = New()
	opt.StringSlice("string", 1, 1)
	_, err = opt.Parse([]string{"--string"})
	if err == nil {
		t.Errorf("Passing option where argument expected didn't raise error")
	}
	if err != nil && err.Error() != fmt.Sprintf(text.ErrorMissingArgument, "string") {
		t.Errorf("Error string didn't match expected value: %s", err.Error())
	}

	opt = New()
	opt.StringSlice("string", 1, 1)
	_, err = opt.Parse([]string{"--string", "--hello", "world"})
	if err == nil {
		t.Errorf("Passing option where argument expected didn't raise error")
	}
	if err != nil && err.Error() != fmt.Sprintf(text.ErrorArgumentWithDash, "string") {
		t.Errorf("Error string didn't match expected value: %s", err.Error())
	}

	opt = New()
	ss := opt.StringSlice("string", 1, 1)
	_, err = opt.Parse([]string{"--string", "hello", "--string", "world"})
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if !reflect.DeepEqual(*ss, []string{"hello", "world"}) {
		t.Errorf("Wrong value: %v != %v", *ss, []string{"hello", "world"})
	}

	opt = New()
	var ssVar []string
	opt.StringSliceVar(&ssVar, "string", 1, 1)
	_, err = opt.Parse([]string{"--string", "hello", "--string", "world"})
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if !reflect.DeepEqual(ssVar, []string{"hello", "world"}) {
		t.Errorf("Wrong value: %v != %v", ssVar, []string{"hello", "world"})
	}
}

func TestGetOptIntSlice(t *testing.T) {
	setup := func() *GetOpt {
		opt := New()
		opt.IntSlice("int", 1, 3, opt.Alias("alias"))
		opt.String("opt", "")
		return opt
	}
	cases := []struct {
		opt    *GetOpt
		option string
		input  []string
		value  []int
	}{
		{setup(),
			"int",
			[]string{"--int", "123"},
			[]int{123},
		},
		{setup(),
			"int",
			[]string{"--int=-123"},
			[]int{-123},
		},
		{setup(),
			"int",
			[]string{"--int", "123", "456", "hello"},
			[]int{123, 456},
		},
		{setup(),
			"int",
			[]string{"--int=123", "456"},
			[]int{123, 456},
		},
		{setup(),
			"int",
			[]string{"--int", "123", "456", "789"},
			[]int{123, 456, 789},
		},
		{setup(),
			"int",
			[]string{"--alias", "123", "456", "789"},
			[]int{123, 456, 789},
		},
		{setup(),
			"int",
			[]string{"--int=123", "456", "789"},
			[]int{123, 456, 789},
		},
		{setup(),
			"int",
			[]string{"--int", "123", "--opt", "world"},
			[]int{123},
		},
		{setup(),
			"int",
			[]string{"--int", "123", "--int", "456"},
			[]int{123, 456},
		},
		{setup(),
			"int",
			[]string{"--int", "1..5"},
			[]int{1, 2, 3, 4, 5},
		},
	}
	for _, c := range cases {
		_, err := c.opt.Parse(c.input)
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		if !reflect.DeepEqual(c.opt.Value(c.option), c.value) {
			t.Errorf("Wrong value: %v != %v", c.opt.Value(c.option), c.value)
		}
	}

	opt := New()
	opt.IntSlice("int", 2, 3)
	_, err := opt.Parse([]string{"--int", "123"})
	if err == nil {
		t.Errorf("Passing less than min didn't raise error")
	}
	if err != nil && err.Error() != fmt.Sprintf(text.ErrorMissingArgument, "int") {
		t.Errorf("Error int didn't match expected value")
	}

	opt = New()
	opt.IntSlice("int", 1, 3)
	_, err = opt.Parse([]string{"--int", "hello"})
	if err == nil {
		t.Errorf("Passing string didn't raise error")
	}
	if err != nil && err.Error() != fmt.Sprintf(text.ErrorConvertToInt, "int", "hello") {
		t.Errorf("Error int didn't match expected value: %s", err)
	}

	opt = New()
	opt.IntSlice("int", 1, 3)
	_, err = opt.Parse([]string{"--int", "hello..3"})
	if err == nil {
		t.Errorf("Passing string didn't raise error")
	}
	if err != nil && err.Error() != fmt.Sprintf(text.ErrorConvertToInt, "int", "hello..3") {
		t.Errorf("Error int didn't match expected value: %s", err)
	}

	opt = New()
	opt.IntSlice("int", 1, 3)
	_, err = opt.Parse([]string{"--int", "1..hello"})
	if err == nil {
		t.Errorf("Passing string didn't raise error")
	}
	if err != nil && err.Error() != fmt.Sprintf(text.ErrorConvertToInt, "int", "1..hello") {
		t.Errorf("Error int didn't match expected value: %s", err)
	}

	opt = New()
	opt.IntSlice("int", 1, 3)
	_, err = opt.Parse([]string{"--int", "3..1"})
	if err == nil {
		t.Errorf("Passing string didn't raise error")
	}
	if err != nil && err.Error() != fmt.Sprintf(text.ErrorConvertToInt, "int", "3..1") {
		t.Errorf("Error int didn't match expected value: %s", err)
	}

	opt = New()
	is := opt.IntSlice("int", 1, 1)
	_, err = opt.Parse([]string{"--int", "1", "--int", "2"})
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if !reflect.DeepEqual(*is, []int{1, 2}) {
		t.Errorf("Wrong value: %v != %v", *is, []int{1, 2})
	}

	opt = New()
	var isVar []int
	opt.IntSliceVar(&isVar, "int", 1, 1)
	_, err = opt.Parse([]string{"--int", "1", "--int", "2"})
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if !reflect.DeepEqual(isVar, []int{1, 2}) {
		t.Errorf("Wrong value: %v != %v", isVar, []int{1, 2})
	}
}

// Verifies that a panic is reached when StringSlice has wrong min
func TestGetOptStringSlicePanicWithWrongMin(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Wrong min didn't panic")
		}
	}()
	opt := New()
	opt.StringSlice("string", 0, 1)
	opt.Parse([]string{})
}

// Verifies that a panic is reached when StringMap has wrong min
func TestGetOptStringMapPanicWithWrongMin(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Wrong min didn't panic")
		}
	}()
	opt := New()
	opt.StringMap("string", 0, 1)
	opt.Parse([]string{})
}

// Verifies that a panic is reached when IntSlice has wrong min
func TestGetOptIntSlicePanicWithWrongMin(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Wrong min didn't panic")
		}
	}()
	opt := New()
	opt.IntSlice("int", 0, 1)
	opt.Parse([]string{})
}

// Verifies that a panic is reached when StringSlice has wrong max
func TestGetOptStringSlicePanicWithWrongMax(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Wrong max didn't panic")
		}
	}()
	opt := New()
	opt.StringSlice("string", 2, 1)
	opt.Parse([]string{})
}

// Verifies that a panic is reached when StringMap has wrong max
func TestGetOptStringMapPanicWithWrongMax(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Wrong max didn't panic")
		}
	}()
	opt := New()
	opt.StringMap("string", 2, 1)
	opt.Parse([]string{})
}

// Verifies that a panic is reached when IntSlice has wrong max
func TestGetOptIntSlicePanicWithWrongMax(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Wrong max didn't panic")
		}
	}()
	opt := New()
	opt.IntSlice("int", 2, 1)
	opt.Parse([]string{})
}

func TestVars(t *testing.T) {
	opt := New()

	var flag, flag2, flag5, flag6 bool
	opt.BoolVar(&flag, "flag", false)
	opt.BoolVar(&flag2, "flag2", true)
	flag3 := opt.Bool("flag3", false)
	flag4 := opt.Bool("flag4", true)
	opt.BoolVar(&flag5, "flag5", false)
	opt.BoolVar(&flag6, "flag6", true)

	var str, str2 string
	opt.StringVar(&str, "stringVar", "")
	opt.StringVar(&str2, "stringVar2", "")

	var integer int
	opt.IntVar(&integer, "intVar", 0)

	var float float64
	opt.Float64Var(&float, "float64Var", 0)

	_, err := opt.Parse([]string{
		"-flag",
		"-flag2",
		"-flag3",
		"-flag4",
		"--stringVar", "hello",
		"--stringVar2=world",
		"--intVar", "123",
		"--float64Var", "1.23",
	})
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}

	if flag != true {
		t.Errorf("flag didn't have expected value: %v != %v", flag, true)
	}
	if flag2 != false {
		t.Errorf("flag2 didn't have expected value: %v != %v", flag2, false)
	}
	if *flag3 != true {
		t.Errorf("flag3 didn't have expected value: %v != %v", *flag3, true)
	}
	if *flag4 != false {
		t.Errorf("flag4 didn't have expected value: %v != %v", *flag4, false)
	}
	if flag5 != false {
		t.Errorf("flag5 didn't have expected value: %v != %v", flag5, false)
	}
	if flag6 != true {
		t.Errorf("flag6 didn't have expected value: %v != %v", flag6, true)
	}

	if str != "hello" {
		t.Errorf("str didn't have expected value: %v != %v", str, "hello")
	}
	if str2 != "world" {
		t.Errorf("str2 didn't have expected value: %v != %v", str, "world")
	}
	if integer != 123 {
		t.Errorf("integer didn't have expected value: %v != %v", integer, 123)
	}
	if float != 1.23 {
		t.Errorf("float didn't have expected value: %v != %v", float, 1.23)
	}
}

func TestDefaultValues(t *testing.T) {
	var flag bool
	var str, str2 string
	var integer, integer2 int

	opt := New()
	opt.Bool("flag", false)
	opt.BoolVar(&flag, "varflag", false)
	opt.String("string", "")
	opt.String("string2", "default")
	str3 := opt.String("string3", "default")
	opt.StringVar(&str, "stringVar", "")
	opt.StringVar(&str2, "stringVar2", "default")
	opt.Int("int", 0)
	int2 := opt.Int("int2", 5)
	opt.IntVar(&integer, "intVar", 0)
	opt.IntVar(&integer2, "intVar2", 5)
	opt.StringSlice("string-repeat", 1, 1)
	opt.StringMap("string-map", 1, 1)

	_, err := opt.Parse([]string{})

	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}

	expected := map[string]interface{}{
		"flag":          false,
		"varflag":       false,
		"string":        "",
		"string2":       "default",
		"stringVar":     "",
		"stringVar2":    "default",
		"int":           0,
		"intVar":        0,
		"string-repeat": []string{},
		"string-map":    map[string]string{},
	}

	for k := range expected {
		if !reflect.DeepEqual(opt.Value(k), expected[k]) {
			t.Errorf("Wrong value: %s\n%v !=\n%v", k, opt.Value(k), expected[k])
		}
	}

	if flag != false {
		t.Errorf("flag didn't have expected value: %v != %v", flag, true)
	}
	if str != "" {
		t.Errorf("str didn't have expected value: %v != %v", str, "")
	}
	if str2 != "default" {
		t.Errorf("str2 didn't have expected value: %v != %v", str, "default")
	}
	if *str3 != "default" {
		t.Errorf("str didn't have expected value: %v != %v", str3, "default")
	}
	if integer != 0 {
		t.Errorf("integer didn't have expected value: %v != %v", integer, 123)
	}
	if integer2 != 5 {
		t.Errorf("integer2 didn't have expected value: %v != %v", integer2, 5)
	}
	if *int2 != 5 {
		t.Errorf("int2 didn't have expected value: %v != %v", int2, 5)
	}

	// Tested above, but it gives me a feel for how it would be used

	if opt.Value("flag").(bool) {
		t.Errorf("flag didn't have expected value: %v != %v", opt.Value("flag"), false)
	}
	if opt.Value("non-used-flag") != nil && opt.Value("non-used-flag").(bool) {
		t.Errorf("non-used-flag didn't have expected value: %v != %v", opt.Value("non-used-flag"), nil)
	}
	if opt.Value("string") != "" {
		t.Errorf("str didn't have expected value: %v != %v", opt.Value("string"), "")
	}
	if opt.Value("int") != 0 {
		t.Errorf("int didn't have expected value: %v != %v", opt.Value("int"), 0)
	}
}

func TestBundling(t *testing.T) {
	buf := setupLogging()
	var o, p bool
	var s string
	opt := New()
	opt.BoolVar(&o, "o", false)
	opt.BoolVar(&p, "p", false)
	opt.StringVar(&s, "t", "")
	opt.SetMode(Bundling)
	_, err := opt.Parse([]string{
		"-opt=arg",
	})
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if o != true {
		t.Errorf("o didn't have expected value: %v != %v", o, true)
	}
	if p != true {
		t.Errorf("p didn't have expected value: %v != %v", p, true)
	}
	if s != "arg" {
		t.Errorf("t didn't have expected value: %v != %v", s, "arg")
	}
	t.Log(buf.String())
}

func TestSingleDash(t *testing.T) {
	var o string
	var p bool
	var s string
	opt := New()
	opt.StringVar(&o, "o", "")
	opt.BoolVar(&p, "p", false)
	opt.StringVar(&s, "t", "")
	opt.SetMode(SingleDash)
	_, err := opt.Parse([]string{
		"-opt=arg",
	})
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if o != "pt=arg" {
		t.Errorf("o didn't have expected value: %v != %v", o, "pt=arg")
	}
	if opt.Called("p") || p != false {
		t.Errorf("p didn't have expected value: %v != %v", p, false)
	}
	if opt.Called("t") || s != "" {
		t.Errorf("t didn't have expected value: %v != %v", s, "")
	}
}

func TestIncrement(t *testing.T) {
	var i, j int
	opt := New()
	opt.IncrementVar(&i, "i", 0, opt.Alias("alias"))
	opt.IncrementVar(&j, "j", 0)
	ip := opt.Increment("ip", 0)
	_, err := opt.Parse([]string{
		"--i",
		"--ip",
	})
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if i != 1 {
		t.Errorf("i didn't have expected value: %v != %v", i, 1)
	}
	if j != 0 {
		t.Errorf("i didn't have expected value: %v != %v", j, 0)
	}
	if *ip != 1 {
		t.Errorf("ip didn't have expected value: %v != %v", *ip, 1)
	}
	opt = New()
	opt.IncrementVar(&i, "i", 0, opt.Alias("alias"))
	opt.IncrementVar(&j, "j", 0)
	ip = opt.Increment("ip", 0)
	_, err = opt.Parse([]string{
		"--i", "hello", "--i", "world", "--alias",
		"--ip", "--ip", "--ip", "--ip", "--ip",
	})
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if i != 3 {
		t.Errorf("i didn't have expected value: %v != %v", i, 3)
	}
	if j != 0 {
		t.Errorf("i didn't have expected value: %v != %v", j, 0)
	}
	if *ip != 5 {
		t.Errorf("ip didn't have expected value: %v != %v", *ip, 5)
	}
}

func TestLonesomeDash(t *testing.T) {
	var stdin bool
	opt := New()
	opt.BoolVar(&stdin, "-", false)
	_, err := opt.Parse([]string{
		"-",
	})
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if !opt.Called("-") || stdin != true {
		t.Errorf("stdin didn't have expected value: %v != %v", stdin, true)
	}
}

// TODO: Decide if I want to include sort just for stringer so the results are always the same for testing purposes.
func TestStringer(t *testing.T) {
	opt := New()
	opt.Bool("flag", false)
	opt.String("string", "")
	opt.Int("int", 0)
	_, err := opt.Parse([]string{
		"--flag",
		"--string", "hello",
		"--int", "123",
	})
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	opt.Stringer()
	// 	expected := `{
	// "flag":true,
	// "string":"hello",
	// "int":123,
	// }`
}

func TestSynopsis(t *testing.T) {
	opt := New()
	opt.Bool("flag", false, opt.Alias("f"))
	opt.String("string", "")
	opt.String("str", "str", opt.Required(), opt.GetEnv("_STR"))
	opt.Int("int", 0, opt.Required())
	opt.Float64("float", 0, opt.Alias("fl"))
	opt.StringSlice("strSlice", 1, 2, opt.ArgName("my_value"), opt.GetEnv("_STR_SLICE"))
	opt.StringSlice("list", 1, 1)
	opt.StringSlice("req-list", 1, 2, opt.Required(), opt.ArgName("item"))
	opt.IntSlice("intSlice", 1, 1, opt.Description("This option is using an int slice\nLets see how multiline works"))
	opt.StringMap("strMap", 1, 2, opt.Description("Hello world"))
	opt.NewCommand("log", "Log stuff")
	opt.NewCommand("show", "Show stuff")
	_, err := opt.Parse([]string{"--str", "a", "--int", "0", "--req-list", "a"})
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	name := opt.Help(HelpName)
	synopsis := opt.Help(HelpSynopsis)
	commandList := opt.Help(HelpCommandList)
	optionList := opt.Help(HelpOptionList)
	expectedName := `NAME:
    go-getoptions.test

`
	expectedSynopsis := `SYNOPSIS:
    go-getoptions.test --int <int> <--req-list <item>...>... --str <string>
                       [--flag|-f] [--float|--fl <float64>] [--intSlice <int>]...
                       [--list <string>]... [--strMap <key=value>...]...
                       [--strSlice <my_value>...]... [--string <string>]
                       <command> [<args>]

`
	expectedCommandList := `COMMANDS:
    log     Log stuff
    show    Show stuff

`
	expectedOptionList := `REQUIRED PARAMETERS:
    --int <int>

    --req-list <item>...

    --str <string>              (env: _STR)

OPTIONS:
    --flag|-f                   (default: false)

    --float|--fl <float64>      (default: 0.000000)

    --intSlice <int>            This option is using an int slice
                                Lets see how multiline works (default: [])

    --list <string>             (default: [])

    --strMap <key=value>...     Hello world (default: {})

    --strSlice <my_value>...    (default: [], env: _STR_SLICE)

    --string <string>           (default: "")

`

	if name != expectedName {
		fmt.Printf("got:\n%s\nexpected:\n%s\n", name, expectedName)
		t.Errorf("Unexpected name:\n%s", firstDiff(name, expectedName))
	}
	if synopsis != expectedSynopsis {
		fmt.Printf("got:\n%s\nexpected:\n%s\n", synopsis, expectedSynopsis)
		t.Errorf("Unexpected synopsis:\n%s", firstDiff(synopsis, expectedSynopsis))
	}
	if commandList != expectedCommandList {
		fmt.Printf("got:\n%s\nexpected:\n%s\n", commandList, expectedCommandList)
		t.Errorf("Unexpected commandList:\n%s", firstDiff(commandList, expectedCommandList))
	}
	if optionList != expectedOptionList {
		fmt.Printf("got:\n%s\nexpected:\n%s\n", optionList, expectedOptionList)
		t.Errorf("Unexpected option list:\n%s", firstDiff(optionList, expectedOptionList))
	}
	if opt.Help() != expectedSynopsis+expectedCommandList+expectedOptionList {
		t.Errorf("Unexpected help:\n---\n%s\n---\n", opt.Help())
	}

	opt = New()
	opt.NewCommand("log", "Log stuff")
	opt.NewCommand("show", "Show stuff")
	opt.Self("name", "description...")
	_, err = opt.Parse([]string{})
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	name = opt.Help(HelpName)
	synopsis = opt.Help(HelpSynopsis)
	expectedName = `NAME:
    name - description...

`
	expectedSynopsis = `SYNOPSIS:
    name <command> [<args>]

`
	if name != expectedName {
		fmt.Printf("got:\n%s\nexpected:\n%s\n", name, expectedName)
		t.Errorf("Unexpected name:\n%s", firstDiff(name, expectedName))
	}
	if synopsis != expectedSynopsis {
		fmt.Printf("got:\n%s\nexpected:\n%s\n", synopsis, expectedSynopsis)
		t.Errorf("Unexpected synopsis:\n%s", firstDiff(synopsis, expectedSynopsis))
	}
	if opt.Help() != expectedName+expectedSynopsis+opt.Help(HelpCommandList)+opt.Help(HelpOptionList) {
		t.Errorf("Unexpected help:\n---\n%s\n---\n", opt.Help())
	}

	opt = New()
	opt.HelpSynopsisArgs("[<filename>]")
	_, err = opt.Parse([]string{})
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	synopsis = opt.Help(HelpSynopsis)
	commandList = opt.Help(HelpCommandList)
	expectedSynopsis = `SYNOPSIS:
    go-getoptions.test [<filename>]

`
	expectedCommandList = ""
	if synopsis != expectedSynopsis {
		fmt.Printf("got:\n%s\nexpected:\n%s\n", synopsis, expectedSynopsis)
		t.Errorf("Unexpected synopsis:\n%s", firstDiff(synopsis, expectedSynopsis))
	}
	if commandList != expectedCommandList {
		fmt.Printf("got:\n%s\nexpected:\n%s\n", commandList, expectedCommandList)
		t.Errorf("Unexpected commandList:\n%s", firstDiff(commandList, expectedCommandList))
	}

	opt = New()
	logCmd := opt.NewCommand("log", "Log stuff")
	subLogCmd := logCmd.NewCommand("sublog", "Sub Log stuff")
	_, err = opt.Parse([]string{})
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	name = subLogCmd.Help(HelpName)
	synopsis = subLogCmd.Help(HelpSynopsis)
	expectedName = `NAME:
    go-getoptions.test log sublog - Sub Log stuff

`
	expectedSynopsis = `SYNOPSIS:
    go-getoptions.test log sublog [<args>]

`
	if name != expectedName {
		fmt.Printf("got:\n%s\nexpected:\n%s\n", name, expectedName)
		t.Errorf("Unexpected name:\n%s", firstDiff(name, expectedName))
	}
	if synopsis != expectedSynopsis {
		fmt.Printf("got:\n%s\nexpected:\n%s\n", synopsis, expectedSynopsis)
		t.Errorf("Unexpected synopsis:\n%s", firstDiff(synopsis, expectedSynopsis))
	}
	if subLogCmd.Help() != expectedName+expectedSynopsis+subLogCmd.Help(HelpCommandList)+subLogCmd.Help(HelpOptionList) {
		t.Errorf("Unexpected help:\n---\n%s\n---\n", opt.Help())
	}
}

func TestCompletion(t *testing.T) {
	called := false
	exitFn = func(code int) { called = true }
	opt := New()
	opt.Bool("flag", false, opt.Alias("f"))
	opt.NewCommand("help", "Show help").CustomCompletion([]string{"log", "show"})

	cleanup := func() {
		os.Setenv("COMP_LINE", "")
		completionWriter = os.Stdout
		called = false
	}

	tests := []struct {
		name     string
		setup    func()
		expected string
	}{
		{"option", func() { os.Setenv("COMP_LINE", "test --f") }, "--flag\n"},
		{"command", func() { os.Setenv("COMP_LINE", "test h") }, "help\n"},
		{"command", func() { os.Setenv("COMP_LINE", "test help ") }, "log\nshow\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			s := ""
			buf := bytes.NewBufferString(s)
			completionWriter = buf
			_, err := opt.Parse([]string{})
			if err != nil {
				t.Errorf("Unexpected error: %s", err)
			}
			if !called {
				t.Errorf("COMP_LINE set and exit wasn't called")
			}
			if buf.String() != tt.expected {
				t.Errorf("Error\ngot: '%s', expected: '%s'\n", buf.String(), tt.expected)
			}
			cleanup()
		})
	}
}

// Verifies that a panic is reached when Command is called with a getoptions without a name.
func TestCommandPanicWithNoNameInput(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("no name command didn't panic")
		}
	}()
	opt := New()
	opt.NewCommand("", "")
	opt.Parse([]string{})
}

// Verifies that a panic is reached when the same option is defined twice in the command.
func TestCommandDuplicateDefinition(t *testing.T) {
	s := ""
	buf := bytes.NewBufferString(s)
	Debug.SetOutput(buf)
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Duplicate definition did not panic")
		}
	}()
	opt := New()
	opt.String("profile", "", opt.Alias("p"))
	command := opt.NewCommand("command", "")
	command.String("password", "", command.Alias("p"))
	_, err := opt.Parse([]string{})
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	t.Log(buf.String())
}

func TestCommandDuplicateDefinition2(t *testing.T) {
	s := ""
	buf := bytes.NewBufferString(s)
	Debug.SetOutput(buf)
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Duplicate definition did not panic")
		}
	}()
	opt := New()
	opt.String("profile", "", opt.Alias("p"))
	command := opt.NewCommand("command", "")
	command.String("p", "")
	_, err := opt.Parse([]string{})
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	t.Log(buf.String())
}

// Make options unambiguous with subcomamnds.
// --profile at the parent was getting matched with the -p for --password at the child.
func TestCommandAmbiguosOption(t *testing.T) {
	t.Run("Should match parent", func(t *testing.T) {
		buf := setupLogging()
		var profile, password, password2 string
		opt := New()
		opt.SetUnknownMode(Pass)
		opt.StringVar(&profile, "profile", "")
		command := opt.NewCommand("command", "")
		command.StringVar(&password, "password", "")
		command2 := opt.NewCommand("command2", "")
		command2.StringVar(&password2, "password", "")
		remaining, err := opt.Parse([]string{"-pr", "hello"})
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		remaining, err = command.Parse(remaining)
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		if profile != "hello" {
			t.Errorf("Unexpected called option profile %s", profile)
		}
		if password != "" {
			t.Errorf("Unexpected called option password %s", password)
		}
		if password2 != "" {
			t.Errorf("Unexpected called option password %s", password2)
		}
		t.Log(buf.String())
	})

	t.Run("Should match command", func(t *testing.T) {
		buf := setupLogging()
		var profile, password, password2 string
		opt := New()
		opt.SetUnknownMode(Pass)
		opt.StringVar(&profile, "profile", "")
		command := opt.NewCommand("command", "")
		command.StringVar(&password, "password", "")
		command2 := opt.NewCommand("command2", "")
		command2.StringVar(&password2, "password", "")
		remaining, err := opt.Parse([]string{"-pa", "hello"})
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		if profile != "" {
			t.Errorf("Unexpected called option profile %s", profile)
		}
		if password != "" {
			t.Errorf("Unexpected called option password %s", password)
		}
		if password2 != "" {
			t.Errorf("Unexpected called option password %s", password2)
		}
		remaining, err = command.Parse(remaining)
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		if profile != "" {
			t.Errorf("Unexpected called option profile %s", profile)
		}
		if password != "hello" {
			t.Errorf("Unexpected called option password %s", password)
		}
		if password2 != "" {
			t.Errorf("Unexpected called option password %s", password2)
		}
		t.Log(buf.String())
	})

	t.Run("Should fail", func(t *testing.T) {
		buf := setupLogging()
		var profile, password string
		opt := New()
		opt.SetUnknownMode(Pass)
		opt.StringVar(&profile, "profile", "")
		command := opt.NewCommand("command", "")
		command.StringVar(&password, "password", "")
		_, err := opt.Parse([]string{"-p", "hello"})
		if err == nil {
			t.Errorf("Ambiguous argument didn't raise error")
		}
		if err != nil && err.Error() != fmt.Sprintf(text.ErrorAmbiguousArgument, "p", []string{"password", "profile"}) {
			t.Errorf("Error string didn't match expected value: %s", err)
		}
		t.Log(buf.String())
	})

	t.Run("Should match parent", func(t *testing.T) {
		buf := setupLogging()
		var profile, password, password2 string
		opt := New()
		opt.SetUnknownMode(Pass)
		opt.StringVar(&profile, "profile", "", opt.Alias("p"))
		command := opt.NewCommand("command", "")
		command.StringVar(&password, "password", "")
		command2 := opt.NewCommand("command2", "")
		command2.StringVar(&password2, "password", "")
		remaining, err := opt.Parse([]string{"-p", "hello"})
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		if profile != "hello" {
			t.Errorf("Unexpected called option profile %s", profile)
		}
		if password != "" {
			t.Errorf("Unexpected called option password %s", password)
		}
		if password2 != "" {
			t.Errorf("Unexpected called option password %s", password2)
		}
		remaining, err = command.Parse(remaining)
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		if profile != "hello" {
			t.Errorf("Unexpected called option profile %s", profile)
		}
		if password != "" {
			t.Errorf("Unexpected called option password %s", password)
		}
		if password2 != "" {
			t.Errorf("Unexpected called option password %s", password2)
		}
		t.Log(buf.String())
	})

	t.Run("Should match command", func(t *testing.T) {
		buf := setupLogging()
		var profile, password, password2 string
		opt := New()
		opt.SetUnknownMode(Pass)
		opt.StringVar(&profile, "profile", "")
		command := opt.NewCommand("command", "")
		command.StringVar(&password, "password", "", opt.Alias("p"))
		command2 := opt.NewCommand("command2", "")
		command2.StringVar(&password2, "password", "")
		remaining, err := opt.Parse([]string{"-p", "hello"})
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		if profile != "" {
			t.Errorf("Unexpected called option profile %s", profile)
		}
		if password != "" {
			t.Errorf("Unexpected called option password %s", password)
		}
		if password2 != "" {
			t.Errorf("Unexpected called option password %s", password2)
		}
		remaining, err = command.Parse(remaining)
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		if profile != "" {
			t.Errorf("Unexpected called option profile %s", profile)
		}
		if password != "hello" {
			t.Errorf("Unexpected called option password %s", password)
		}
		if password2 != "" {
			t.Errorf("Unexpected called option password %s", password2)
		}
		t.Log(buf.String())
	})

	t.Run("Should match command", func(t *testing.T) {
		buf := setupLogging()
		called := false
		fn := func(ctx context.Context, opt *GetOpt, args []string) error {
			called = true
			return nil
		}
		var profile, password, password2 string
		opt := New()
		opt.SetUnknownMode(Pass)
		opt.StringVar(&profile, "profile", "")
		command := opt.NewCommand("command", "")
		command.StringVar(&password, "password", "", opt.Alias("p"))
		command.SetCommandFn(fn)
		command2 := opt.NewCommand("command2", "")
		command2.StringVar(&password2, "password", "")
		remaining, err := opt.Parse([]string{"command", "-p", "hello"})
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		if profile != "" {
			t.Errorf("Unexpected called option profile %s", profile)
		}
		if password != "" {
			t.Errorf("Unexpected called option password %s", password)
		}
		if password2 != "" {
			t.Errorf("Unexpected called option password %s", password2)
		}
		err = opt.Dispatch(context.Background(), "help", remaining)
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		if !called {
			t.Errorf("not called")
		}
		if profile != "" {
			t.Errorf("Unexpected called option profile %s", profile)
		}
		if password != "hello" {
			t.Errorf("Unexpected called option password %s", password)
		}
		if password2 != "" {
			t.Errorf("Unexpected called option password %s", password2)
		}
		t.Log(buf.String())
	})

	t.Run("Should match parent at command", func(t *testing.T) {
		buf := setupLogging()
		called := false
		fn := func(ctx context.Context, opt *GetOpt, args []string) error {
			called = true
			return nil
		}
		var profile, password, password2 string
		opt := New()
		// opt.SetRequireOrder()
		opt.SetUnknownMode(Pass)
		opt.StringVar(&profile, "profile", "")
		command := opt.NewCommand("command", "")
		command.StringVar(&password, "password", "", opt.Alias("p"))
		command2 := opt.NewCommand("command2", "")
		command2.SetCommandFn(fn)
		remaining, err := opt.Parse([]string{"command2", "-p", "hello"})
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		if profile != "" {
			t.Errorf("Unexpected called option profile %s", profile)
		}
		if password != "" {
			t.Errorf("Unexpected called option password %s", password)
		}
		if password2 != "" {
			t.Errorf("Unexpected called option password %s", password2)
		}
		err = opt.Dispatch(context.Background(), "help", remaining)
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		if !called {
			t.Errorf("not called")
		}
		if profile != "hello" {
			t.Errorf("Unexpected called option profile %s", profile)
		}
		if password != "" {
			t.Errorf("Unexpected called option password %s", password)
		}
		if password2 != "" {
			t.Errorf("Unexpected called option password %s", password2)
		}
		t.Log(buf.String())
	})
}

func TestSetCommandFn(t *testing.T) {
	called := false
	fn := func(ctx context.Context, opt *GetOpt, args []string) error {
		called = true
		return nil
	}
	buf := setupLogging()
	opt := New()
	command := opt.NewCommand("command", "").SetCommandFn(fn)
	remaining, err := opt.Parse([]string{"command"})
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	command.CommandFn(context.Background(), command, remaining)
	if !called {
		t.Errorf("Function not called")
	}
	t.Log(buf.String())
}

func TestDispatch(t *testing.T) {
	t.Run("no args", func(t *testing.T) {
		helpBuf := new(bytes.Buffer)
		called := false
		fn := func(ctx context.Context, opt *GetOpt, args []string) error {
			return nil
		}
		exitFn = func(code int) { called = true }
		buf := setupLogging()
		opt := New()
		opt.Writer = helpBuf
		opt.Bool("help", false)
		opt.NewCommand("command", "").SetCommandFn(fn)
		opt.HelpCommand("")
		remaining, err := opt.Parse([]string{})
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		err = opt.Dispatch(context.Background(), "help", remaining)
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		if !called {
			t.Errorf("Exit not called")
		}
		expected := `SYNOPSIS:
    go-getoptions.test [--help] <command> [<args>]

COMMANDS:
    command    
    help       Use 'go-getoptions.test help <command>' for extra details.

OPTIONS:
    --help    (default: false)

Use 'go-getoptions.test help <command>' for extra details.
`
		if helpBuf.String() != expected {
			t.Errorf("Wrong output:\n%s\n", helpBuf.String())
		}
		t.Log(buf.String())
	})

	t.Run("help case", func(t *testing.T) {
		helpBuf := new(bytes.Buffer)
		called := false
		fn := func(ctx context.Context, opt *GetOpt, args []string) error {
			return nil
		}
		exitFn = func(code int) { called = true }
		buf := setupLogging()
		opt := New()
		opt.Writer = helpBuf
		opt.Bool("help", false)
		opt.NewCommand("command", "").SetCommandFn(fn)
		opt.HelpCommand("")
		remaining, err := opt.Parse([]string{"help"})
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		err = opt.Dispatch(context.Background(), "help", remaining)
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		if !called {
			t.Errorf("Exit not called")
		}
		expected := `SYNOPSIS:
    go-getoptions.test [--help] <command> [<args>]

COMMANDS:
    command    
    help       Use 'go-getoptions.test help <command>' for extra details.

OPTIONS:
    --help    (default: false)

Use 'go-getoptions.test help <command>' for extra details.
`
		if helpBuf.String() != expected {
			t.Errorf("Wrong output:\n%s\n", firstDiff(helpBuf.String(), expected))
		}
		t.Log(buf.String())
	})

	t.Run("help case command", func(t *testing.T) {
		helpBuf := new(bytes.Buffer)
		called := false
		fn := func(ctx context.Context, opt *GetOpt, args []string) error {
			return nil
		}
		exitFn = func(code int) { called = true }
		buf := setupLogging()
		opt := New()
		opt.Writer = helpBuf
		opt.Bool("help", false)
		opt.NewCommand("command", "").SetCommandFn(fn)
		opt.HelpCommand("")
		remaining, err := opt.Parse([]string{"xhelp", "command"})
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		err = opt.Dispatch(context.Background(), "xhelp", remaining)
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		if !called {
			t.Errorf("Exit not called")
		}
		expected := `NAME:
    go-getoptions.test command

SYNOPSIS:
    go-getoptions.test command [--help] [<args>]

OPTIONS:
    --help    (default: false)

`
		if helpBuf.String() != expected {
			t.Errorf("Wrong output:\n%s\n", helpBuf.String())
		}
		t.Log(buf.String())
	})

	t.Run("help case command", func(t *testing.T) {
		helpBuf := new(bytes.Buffer)
		called := false
		exitFn = func(code int) { called = true }
		fn := func(ctx context.Context, opt *GetOpt, args []string) error {
			if opt.Called("help") {
				fmt.Fprintf(helpBuf, opt.Help())
				exitFn(1)
			}
			return nil
		}
		buf := setupLogging()
		opt := New()
		opt.Writer = helpBuf
		opt.Bool("help", false)
		command := opt.NewCommand("command", "").SetCommandFn(fn)
		command.NewCommand("sub-command", "").SetCommandFn(fn)
		command.HelpCommand("")
		opt.HelpCommand("")
		remaining, err := opt.Parse([]string{"command", "--help"})
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		err = opt.Dispatch(context.Background(), "help", remaining)
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		if !called {
			t.Errorf("Exit not called")
		}
		expected := `NAME:
    go-getoptions.test command

SYNOPSIS:
    go-getoptions.test command [--help] <command> [<args>]

COMMANDS:
    help           Use 'go-getoptions.test command help <command>' for extra details.
    sub-command    

OPTIONS:
    --help    (default: false)

`
		if helpBuf.String() != expected {
			t.Errorf("Wrong output:\n%s\n", helpBuf.String())
		}
		t.Log(buf.String())
	})

	t.Run("help case sub-command", func(t *testing.T) {
		helpBuf := new(bytes.Buffer)
		called := false
		commandFn := func(ctx context.Context, opt *GetOpt, args []string) error {
			err := opt.Dispatch(context.Background(), "help", args)
			if err != nil {
				return err
			}
			return nil
		}
		fn := func(ctx context.Context, opt *GetOpt, args []string) error {
			called = true
			return nil
		}
		buf := setupLogging()
		opt := New()
		opt.Writer = helpBuf
		opt.Bool("help", false)
		command := opt.NewCommand("command", "").SetCommandFn(commandFn)
		command.NewCommand("sub-command", "").SetCommandFn(fn)
		command.HelpCommand("")
		opt.HelpCommand("")
		remaining, err := opt.Parse([]string{"command", "sub-command", "--help"})
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		err = opt.Dispatch(context.Background(), "help", remaining)
		if err != nil && !errors.Is(err, ErrorHelpCalled) {
			t.Errorf("Unexpected error: %s", err)
		}
		if called {
			t.Errorf("Fn was called")
		}
		expected := `NAME:
    go-getoptions.test command sub-command

SYNOPSIS:
    go-getoptions.test command sub-command [--help] [<args>]

OPTIONS:
    --help    (default: false)

`
		if helpBuf.String() != expected {
			t.Errorf("Wrong output:\n%s\n", helpBuf.String())
		}
		t.Log(buf.String())
	})

	t.Run("help case sub-command with required option", func(t *testing.T) {
		helpBuf := new(bytes.Buffer)
		called := false
		commandFn := func(ctx context.Context, opt *GetOpt, args []string) error {
			err := opt.Dispatch(context.Background(), "help", args)
			if err != nil {
				return err
			}
			return nil
		}
		fn := func(ctx context.Context, opt *GetOpt, args []string) error {
			called = true
			return nil
		}
		buf := setupLogging()
		opt := New()
		opt.Writer = helpBuf
		opt.Bool("help", false)
		command := opt.NewCommand("command", "").SetCommandFn(commandFn)
		subcommand := command.NewCommand("sub-command", "").SetCommandFn(fn)
		subcommand.String("required", "", opt.Required())
		command.HelpCommand("")
		opt.HelpCommand("")
		remaining, err := opt.Parse([]string{"command", "sub-command", "--help"})
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		err = opt.Dispatch(context.Background(), "help", remaining)
		if err != nil && !errors.Is(err, ErrorHelpCalled) {
			t.Errorf("Unexpected error: %s", err)
		}
		if called {
			t.Errorf("Fn was called")
		}
		expected := `NAME:
    go-getoptions.test command sub-command

SYNOPSIS:
    go-getoptions.test command sub-command --required <string> [--help] [<args>]

REQUIRED PARAMETERS:
    --required <string>

OPTIONS:
    --help                 (default: false)

`
		if helpBuf.String() != expected {
			t.Errorf("Wrong output:\n%s\n", helpBuf.String())
		}
		t.Log(buf.String())
	})

	t.Run("command", func(t *testing.T) {
		called := false
		fn := func(ctx context.Context, opt *GetOpt, args []string) error {
			called = true
			return nil
		}
		buf := setupLogging()
		opt := New()
		opt.Bool("help", false)
		opt.NewCommand("command", "").SetCommandFn(fn)
		opt.HelpCommand("")
		remaining, err := opt.Parse([]string{"command"})
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		err = opt.Dispatch(context.Background(), "help", remaining)
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		if !called {
			t.Errorf("Fn not called")
		}
		t.Log(buf.String())
	})

	t.Run("command multiple parse calls", func(t *testing.T) {
		called := false
		fn := func(ctx context.Context, opt *GetOpt, args []string) error {
			remaining, err := opt.Parse(args)
			if err != nil {
				return err
			}
			remaining, err = opt.Parse(remaining)
			if err != nil {
				return err
			}
			called = opt.Called("command-option")
			return nil
		}
		buf := setupLogging()
		opt := New()
		opt.SetUnknownMode(Pass)
		opt.Bool("help", false)
		cmd := opt.NewCommand("command", "").SetCommandFn(fn)
		cmd.Bool("command-option", false)
		opt.HelpCommand("")
		remaining, err := opt.Parse([]string{"command", "--command-option"})
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		err = opt.Dispatch(context.Background(), "help", remaining)
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		if !called {
			t.Errorf("Option not called")
		}
		t.Log(buf.String())
	})

	t.Run("command error", func(t *testing.T) {
		called := false
		fn := func(ctx context.Context, opt *GetOpt, args []string) error {
			called = true
			return fmt.Errorf("err")
		}
		buf := setupLogging()
		opt := New()
		opt.Bool("help", false)
		opt.NewCommand("command", "").SetCommandFn(fn)
		opt.HelpCommand("")
		remaining, err := opt.Parse([]string{"command"})
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		err = opt.Dispatch(context.Background(), "help", remaining)
		if err == nil {
			t.Errorf("Error not called")
		}
		if !called {
			t.Errorf("Fn not called")
		}
		t.Log(buf.String())
	})

	t.Run("command error", func(t *testing.T) {
		called := false
		fn := func(ctx context.Context, opt *GetOpt, args []string) error {
			called = true
			return fmt.Errorf("err")
		}
		buf := setupLogging()
		opt := New()
		opt.Bool("help", false)
		opt.NewCommand("command", "").SetCommandFn(fn)
		opt.HelpCommand("")
		remaining, err := opt.Parse([]string{"x"})
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		err = opt.Dispatch(context.Background(), "help", remaining)
		if err == nil {
			t.Errorf("Error not called")
		}
		if called {
			t.Errorf("Fn was called")
		}
		t.Log(buf.String())
	})

	t.Run("command error", func(t *testing.T) {
		called := false
		fn := func(ctx context.Context, opt *GetOpt, args []string) error {
			called = true
			return fmt.Errorf("err")
		}
		buf := setupLogging()
		opt := New()
		opt.Bool("help", false)
		opt.SetUnknownMode(Pass)
		opt.NewCommand("command", "").SetCommandFn(fn)
		opt.HelpCommand("")
		remaining, err := opt.Parse([]string{"-x"})
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		err = opt.Dispatch(context.Background(), "help", remaining)
		if err == nil {
			t.Errorf("Error not called")
		}
		if called {
			t.Errorf("Fn was called")
		}
		t.Log(buf.String())
	})

	t.Run("command parse error", func(t *testing.T) {
		called := false
		fn := func(ctx context.Context, opt *GetOpt, args []string) error {
			called = true
			return nil
		}
		buf := setupLogging()
		opt := New()
		opt.Bool("help", false)
		opt.SetUnknownMode(Pass)
		opt.NewCommand("command", "").SetCommandFn(fn)
		opt.HelpCommand("")
		remaining, err := opt.Parse([]string{"command", "-x"})
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		err = opt.Dispatch(context.Background(), "help", remaining)
		if err == nil {
			t.Errorf("Error not called")
		}
		if called {
			t.Errorf("Fn was called")
		}
		t.Log(buf.String())
	})

	t.Run("help error", func(t *testing.T) {
		fn := func(ctx context.Context, opt *GetOpt, args []string) error {
			return nil
		}
		buf := setupLogging()
		opt := New()
		opt.Bool("help", false)
		opt.NewCommand("command", "").SetCommandFn(fn)
		opt.HelpCommand("")
		remaining, err := opt.Parse([]string{"help", "x"})
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		err = opt.Dispatch(context.Background(), "help", remaining)
		if err == nil {
			t.Errorf("Error not called")
		}
		t.Log(buf.String())
	})
}

func TestGetEnv(t *testing.T) {
	setup := func(v string) {
		os.Setenv("_get_opt_env_test1", v)
		os.Setenv("_get_opt_env_test2", v)
	}
	cleanup := func() {
		os.Unsetenv("_get_opt_env_test1")
		os.Unsetenv("_get_opt_env_test2")
	}
	/////////////////////////////////////////////////////////////////////////////
	// Bool
	/////////////////////////////////////////////////////////////////////////////
	t.Run("bool no env", func(t *testing.T) {
		cleanup()
		buf := setupLogging()
		var v1 bool
		opt := New()
		opt.BoolVar(&v1, "opt1", false, opt.GetEnv("_get_opt_env_test1"))
		v2 := opt.Bool("opt2", false, opt.GetEnv("_get_opt_env_test2"))
		_, err := opt.Parse([]string{})
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		if v1 != false {
			t.Errorf("Unexpected value: %v", v1)
		}
		if *v2 != false {
			t.Errorf("Unexpected value: %v", *v2)
		}
		t.Log(buf.String())
	})
	t.Run("bool false env with option", func(t *testing.T) {
		// Ensures that the cli args always have precedence over env vars.
		setup("false")
		buf := setupLogging()
		var v1 bool
		opt := New()
		opt.BoolVar(&v1, "opt1", false, opt.GetEnv("_get_opt_env_test1"))
		v2 := opt.Bool("opt2", false, opt.GetEnv("_get_opt_env_test2"))
		_, err := opt.Parse([]string{"--opt1", "--opt2"})
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		if v1 != true {
			t.Errorf("Unexpected value: %v, %#v", v1, opt.Option("opt1"))
		}
		if *v2 != true {
			t.Errorf("Unexpected value: %v, %#v", *v2, opt.Option("opt2"))
		}
		t.Log(buf.String())
		cleanup()
	})
	t.Run("bool true env with option", func(t *testing.T) {
		setup("true")
		buf := setupLogging()
		var v1 bool
		opt := New()
		opt.BoolVar(&v1, "opt1", false, opt.GetEnv("_get_opt_env_test1"))
		v2 := opt.Bool("opt2", false, opt.GetEnv("_get_opt_env_test2"))
		_, err := opt.Parse([]string{"--opt1", "--opt2"})
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		if v1 != true {
			t.Errorf("Unexpected value: %v, %#v", v1, opt.Option("opt1"))
		}
		if *v2 != true {
			t.Errorf("Unexpected value: %v, %#v", *v2, opt.Option("opt2"))
		}
		t.Log(buf.String())
		cleanup()
	})
	t.Run("bool true env with option reverse", func(t *testing.T) {
		// Ensures that the cli args always have precedence over env vars.
		setup("true")
		buf := setupLogging()
		var v1 bool
		opt := New()
		opt.BoolVar(&v1, "opt1", true, opt.GetEnv("_get_opt_env_test1"))
		v2 := opt.Bool("opt2", true, opt.GetEnv("_get_opt_env_test2"))
		_, err := opt.Parse([]string{"--opt1", "--opt2"})
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		if v1 != false {
			t.Errorf("Unexpected value: %v, %#v", v1, opt.Option("opt1"))
		}
		if *v2 != false {
			t.Errorf("Unexpected value: %v, %#v", *v2, opt.Option("opt2"))
		}
		t.Log(buf.String())
		cleanup()
	})
	t.Run("bool env false", func(t *testing.T) {
		setup("fAlse")
		buf := setupLogging()
		var v1 bool
		opt := New()
		opt.BoolVar(&v1, "opt1", false, opt.GetEnv("_get_opt_env_test1"))
		v2 := opt.Bool("opt2", false, opt.GetEnv("_get_opt_env_test2"))
		_, err := opt.Parse([]string{})
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		if v1 != false {
			t.Errorf("Unexpected value: %v, %#v", v1, opt.Option("opt1"))
		}
		if *v2 != false {
			t.Errorf("Unexpected value: %v, %#v", *v2, opt.Option("opt2"))
		}
		t.Log(buf.String())
		cleanup()
	})
	t.Run("bool env true", func(t *testing.T) {
		setup("tRue")
		buf := setupLogging()
		var v1 bool
		opt := New()
		opt.BoolVar(&v1, "opt1", false, opt.GetEnv("_get_opt_env_test1"), opt.Description("opt1"))
		v2 := opt.Bool("opt2", false, opt.GetEnv("_get_opt_env_test2"))
		_, err := opt.Parse([]string{})
		if err != nil {
			t.Errorf("Unexpected error: %s\n", err)
		}
		if *v2 != true {
			t.Errorf("Unexpected value: %v\n%#v\n", *v2, opt.Option("opt2"))
		}
		if v1 != true {
			t.Errorf("Unexpected value: %v\n%#v\n%v", v1, opt.Option("opt1"), opt.Value("opt1"))
		}
		t.Log(buf.String())
		cleanup()
	})
	t.Run("bool env true reverse", func(t *testing.T) {
		setup("tRue")
		buf := setupLogging()
		var v1 bool
		opt := New()
		opt.BoolVar(&v1, "opt1", true, opt.GetEnv("_get_opt_env_test1"), opt.Description("opt1"))
		v2 := opt.Bool("opt2", true, opt.GetEnv("_get_opt_env_test2"))
		_, err := opt.Parse([]string{})
		if err != nil {
			t.Errorf("Unexpected error: %s\n", err)
		}
		if *v2 != true {
			t.Errorf("Unexpected value: %v\n%#v\n", *v2, opt.Option("opt2"))
		}
		if v1 != true {
			t.Errorf("Unexpected value: %v\n%#v\n%v", v1, opt.Option("opt1"), opt.Value("opt1"))
		}
		t.Log(buf.String())
		cleanup()
	})
	t.Run("bool env false reverse", func(t *testing.T) {
		setup("fAlse")
		buf := setupLogging()
		var v1 bool
		opt := New()
		opt.BoolVar(&v1, "opt1", true, opt.GetEnv("_get_opt_env_test1"), opt.Description("opt1"))
		v2 := opt.Bool("opt2", true, opt.GetEnv("_get_opt_env_test2"))
		_, err := opt.Parse([]string{})
		if err != nil {
			t.Errorf("Unexpected error: %s\n", err)
		}
		if *v2 != false {
			t.Errorf("Unexpected value: %v\n%#v\n", *v2, opt.Option("opt2"))
		}
		if v1 != false {
			t.Errorf("Unexpected value: %v\n%#v\n%v", v1, opt.Option("opt1"), opt.Value("opt1"))
		}
		t.Log(buf.String())
		cleanup()
	})
	/////////////////////////////////////////////////////////////////////////////
	// String
	/////////////////////////////////////////////////////////////////////////////
	t.Run("string no env", func(t *testing.T) {
		cleanup()
		buf := setupLogging()
		var v1 string
		opt := New()
		opt.StringVar(&v1, "opt1", "default1", opt.GetEnv("_get_opt_env_test1"))
		v2 := opt.String("opt2", "default2", opt.GetEnv("_get_opt_env_test2"))
		_, err := opt.Parse([]string{})
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		if v1 != "default1" {
			t.Errorf("Unexpected value: %s", v1)
		}
		if *v2 != "default2" {
			t.Errorf("Unexpected value: %s", *v2)
		}
		t.Log(buf.String())
	})
	t.Run("string env with option", func(t *testing.T) {
		setup("set-env")
		buf := setupLogging()
		var v1 string
		opt := New()
		opt.StringVar(&v1, "opt1", "default1", opt.GetEnv("_get_opt_env_test1"))
		v2 := opt.String("opt2", "default2", opt.GetEnv("_get_opt_env_test2"))
		_, err := opt.Parse([]string{"--opt1", "option1", "--opt2", "option2"})
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		if v1 != "option1" {
			t.Errorf("Unexpected value: %s, %#v", v1, opt.Option("opt1"))
		}
		if *v2 != "option2" {
			t.Errorf("Unexpected value: %s, %#v", *v2, opt.Option("opt2"))
		}
		t.Log(buf.String())
		cleanup()
	})
	t.Run("string env", func(t *testing.T) {
		setup("set-env")
		buf := setupLogging()
		var v1 string
		opt := New()
		opt.StringVar(&v1, "opt1", "default1", opt.GetEnv("_get_opt_env_test1"))
		v2 := opt.String("opt2", "default2", opt.GetEnv("_get_opt_env_test2"))
		_, err := opt.Parse([]string{})
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		if v1 != "set-env" {
			t.Errorf("Unexpected value: %s, %#v", v1, opt.Option("opt1"))
		}
		if *v2 != "set-env" {
			t.Errorf("Unexpected value: %s, %#v", *v2, opt.Option("opt2"))
		}
		t.Log(buf.String())
		cleanup()
	})
	t.Run("string optional env", func(t *testing.T) {
		setup("set-env")
		buf := setupLogging()
		var v1 string
		opt := New()
		opt.StringVarOptional(&v1, "opt1", "default1", opt.GetEnv("_get_opt_env_test1"))
		v2 := opt.StringOptional("opt2", "default2", opt.GetEnv("_get_opt_env_test2"))
		_, err := opt.Parse([]string{})
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		if v1 != "set-env" {
			t.Errorf("Unexpected value: %s, %#v", v1, opt.Option("opt1"))
		}
		if *v2 != "set-env" {
			t.Errorf("Unexpected value: %s, %#v", *v2, opt.Option("opt2"))
		}
		t.Log(buf.String())
		cleanup()
	})
	t.Run("string env required", func(t *testing.T) {
		setup("set-env")
		buf := setupLogging()
		var v1 string
		opt := New()
		opt.StringVar(&v1, "opt1", "default1", opt.GetEnv("_get_opt_env_test1"), opt.Required())
		v2 := opt.String("opt2", "default2", opt.GetEnv("_get_opt_env_test2"), opt.Required())
		_, err := opt.Parse([]string{})
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		if v1 != "set-env" {
			t.Errorf("Unexpected value: %s, %#v", v1, opt.Option("opt1"))
		}
		if *v2 != "set-env" {
			t.Errorf("Unexpected value: %s, %#v", *v2, opt.Option("opt2"))
		}
		if !opt.Called("opt1") {
			t.Errorf("Not called: %s, %#v", v1, opt.Option("opt1"))
		}
		if opt.CalledAs("opt1") != "_get_opt_env_test1" {
			t.Errorf("Not called as %s: %s, %#v", "_get_opt_env_test1", v1, opt.Option("opt1"))
		}
		t.Log(buf.String())
		cleanup()
	})
	/////////////////////////////////////////////////////////////////////////////
	// Int
	/////////////////////////////////////////////////////////////////////////////
	t.Run("int no env", func(t *testing.T) {
		cleanup()
		buf := setupLogging()
		var v1 int
		opt := New()
		opt.IntVar(&v1, "opt1", 123, opt.GetEnv("_get_opt_env_test1"))
		v2 := opt.Int("opt2", 123, opt.GetEnv("_get_opt_env_test2"))
		_, err := opt.Parse([]string{})
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		if v1 != 123 {
			t.Errorf("Unexpected value: %d", v1)
		}
		if *v2 != 123 {
			t.Errorf("Unexpected value: %d", *v2)
		}
		t.Log(buf.String())
	})
	t.Run("int env with option", func(t *testing.T) {
		setup("456")
		buf := setupLogging()
		var v1 int
		opt := New()
		opt.IntVar(&v1, "opt1", 123, opt.GetEnv("_get_opt_env_test1"))
		v2 := opt.Int("opt2", 123, opt.GetEnv("_get_opt_env_test2"))
		_, err := opt.Parse([]string{"--opt1", "789", "--opt2", "789"})
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		if v1 != 789 {
			t.Errorf("Unexpected value: %d, %#v", v1, opt.Option("opt1"))
		}
		if *v2 != 789 {
			t.Errorf("Unexpected value: %d, %#v", *v2, opt.Option("opt2"))
		}
		t.Log(buf.String())
		cleanup()
	})
	t.Run("int env", func(t *testing.T) {
		setup("456")
		buf := setupLogging()
		var v1 int
		opt := New()
		opt.IntVar(&v1, "opt1", 123, opt.GetEnv("_get_opt_env_test1"))
		v2 := opt.Int("opt2", 123, opt.GetEnv("_get_opt_env_test2"))
		_, err := opt.Parse([]string{})
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		if v1 != 456 {
			t.Errorf("Unexpected value: %d, %#v", v1, opt.Option("opt1"))
		}
		if *v2 != 456 {
			t.Errorf("Unexpected value: %d, %#v", *v2, opt.Option("opt2"))
		}
		t.Log(buf.String())
		cleanup()
	})
	t.Run("int optional env", func(t *testing.T) {
		setup("456")
		buf := setupLogging()
		var v1 int
		opt := New()
		opt.IntVarOptional(&v1, "opt1", 123, opt.GetEnv("_get_opt_env_test1"))
		v2 := opt.IntOptional("opt2", 123, opt.GetEnv("_get_opt_env_test2"))
		_, err := opt.Parse([]string{})
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		if v1 != 456 {
			t.Errorf("Unexpected value: %d, %#v", v1, opt.Option("opt1"))
		}
		if *v2 != 456 {
			t.Errorf("Unexpected value: %d, %#v", *v2, opt.Option("opt2"))
		}
		t.Log(buf.String())
		cleanup()
	})
	t.Run("int env error", func(t *testing.T) {
		setup("abc")
		buf := setupLogging()
		var v1 int
		opt := New()
		opt.IntVar(&v1, "opt1", 123, opt.GetEnv("_get_opt_env_test1"))
		v2 := opt.Int("opt2", 123, opt.GetEnv("_get_opt_env_test2"))
		_, err := opt.Parse([]string{})
		// TODO: Handle errors when env vars don't have proper types
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		if v1 != 123 {
			t.Errorf("Unexpected value: %d, %#v", v1, opt.Option("opt1"))
		}
		if *v2 != 123 {
			t.Errorf("Unexpected value: %d, %#v", *v2, opt.Option("opt2"))
		}
		t.Log(buf.String())
		cleanup()
	})
	/////////////////////////////////////////////////////////////////////////////
	// Float64
	/////////////////////////////////////////////////////////////////////////////
	t.Run("float64 env", func(t *testing.T) {
		setup("456.1")
		buf := setupLogging()
		var v1 float64
		opt := New()
		opt.Float64Var(&v1, "opt1", 123, opt.GetEnv("_get_opt_env_test1"))
		v2 := opt.Float64("opt2", 123, opt.GetEnv("_get_opt_env_test2"))
		_, err := opt.Parse([]string{})
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		if v1 != 456.1 {
			t.Errorf("Unexpected value: %f, %#v", v1, opt.Option("opt1"))
		}
		if *v2 != 456.1 {
			t.Errorf("Unexpected value: %f, %#v", *v2, opt.Option("opt2"))
		}
		t.Log(buf.String())
		cleanup()
	})
	t.Run("float64 optional env", func(t *testing.T) {
		setup("456.1")
		buf := setupLogging()
		var v1 float64
		opt := New()
		opt.Float64VarOptional(&v1, "opt1", 123, opt.GetEnv("_get_opt_env_test1"))
		v2 := opt.Float64Optional("opt2", 123, opt.GetEnv("_get_opt_env_test2"))
		_, err := opt.Parse([]string{})
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		if v1 != 456.1 {
			t.Errorf("Unexpected value: %f, %#v", v1, opt.Option("opt1"))
		}
		if *v2 != 456.1 {
			t.Errorf("Unexpected value: %f, %#v", *v2, opt.Option("opt2"))
		}
		t.Log(buf.String())
		cleanup()
	})
}

func TestAll(t *testing.T) {
	var flag bool
	var str string
	var integer int
	opt := New()
	opt.Bool("flag", false)
	opt.BoolVar(&flag, "varflag", false)
	opt.Bool("non-used-flag", false)
	opt.String("string", "")
	opt.StringVar(&str, "stringVar", "")
	opt.Int("int", 0)
	opt.IntVar(&integer, "intVar", 0)
	opt.StringSlice("string-repeat", 1, 1)
	opt.StringSlice("string-slice-multi", 1, 3)
	opt.StringMap("string-map", 1, 1)

	remaining, err := opt.Parse([]string{
		"hello",
		"--flag",
		"--varflag",
		"happy",
		"--string", "hello",
		"--stringVar", "hello",
		"--int", "123",
		"--intVar", "123",
		"--string-repeat", "hello", "--string-repeat", "world",
		"--string-slice-multi", "hello", "happy", "--string-slice-multi", "world",
		"--string-map", "hello=world", "--string-map", "server=name",
		"world",
	})

	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}

	remaining, err = opt.Parse(remaining)

	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}

	if !reflect.DeepEqual(remaining, []string{"hello", "happy", "world"}) {
		t.Errorf("remaining didn't have expected value: %v != %v", remaining, []string{"hello", "happy", "world"})
	}

	expected := map[string]interface{}{
		"flag":               true,
		"string":             "hello",
		"int":                123,
		"string-repeat":      []string{"hello", "world"},
		"string-slice-multi": []string{"hello", "happy", "world"},
		"string-map":         map[string]string{"hello": "world", "server": "name"},
	}

	for k := range expected {
		if !reflect.DeepEqual(opt.Value(k), expected[k]) {
			t.Errorf("Wrong value: %v != %v", opt.Value(k), expected[k])
		}
	}

	if flag != true {
		t.Errorf("flag didn't have expected value: %v != %v", flag, true)
	}
	if str != "hello" {
		t.Errorf("str didn't have expected value: %v != %v", str, "hello")
	}
	if integer != 123 {
		t.Errorf("int didn't have expected value: %v != %v", integer, 123)
	}

	// Tested above, but it gives me a feel for how it would be used

	if !opt.Value("flag").(bool) {
		t.Errorf("flag didn't have expected value: %v != %v", opt.Value("flag"), true)
	}
	if opt.Value("non-used-flag").(bool) {
		t.Errorf("non-used-flag didn't have expected value: %v != %v", opt.Value("non-used-flag"), false)
	}
	if opt.Value("string") != "hello" {
		t.Errorf("str didn't have expected value: %v != %v", opt.Value("string"), "hello")
	}
	if opt.Value("int") != 123 {
		t.Errorf("int didn't have expected value: %v != %v", opt.Value("int"), 123)
	}
}

func TestInterruptContext(t *testing.T) {
	iterations := 1000
	sum := 0
	called := false
	cleanupFn := func() { called = true }
	helpBuf := new(bytes.Buffer)
	buf := setupLogging()
	opt := New()
	opt.Writer = helpBuf
	ctx, cancel, done := opt.InterruptContext()
	defer func() {
		cancel()
		<-done
		if sum >= iterations {
			t.Errorf("Interrupt not captured: %d\n", sum)
		}
	}()

	for i := 0; i <= iterations; i++ {
		sum++
		select {
		case <-ctx.Done():
			cleanupFn()
			return
		default:
		}
		if i == 0 {
			id := os.Getpid()
			fmt.Printf("process id %d\n", id)
			p, err := os.FindProcess(id)
			if err != nil {
				t.Errorf("Unexpected error: %s\n", err)
				continue
			}
			fmt.Printf("process %v\n", p)
			err = p.Signal(os.Interrupt)
			if err != nil {
				t.Errorf("Unexpected error: %s\n", err)
				continue
			}
		}
		// Give the kernel time to process the signal
		time.Sleep(1 * time.Millisecond)
	}
	if !called {
		t.Errorf("Cleanup function not called")
	}
	if helpBuf.String() != text.MessageOnInterrupt+"\n" {
		t.Errorf("Wrong output: %s", helpBuf.String())
	}
	t.Log(buf.String())
}
