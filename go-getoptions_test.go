// This file is part of go-getoptions.
//
// Copyright (C) 2015-2017  David Gamba Rios
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
package getoptions

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"testing"
)

func TestIsOption(t *testing.T) {
	Debug.SetOutput(os.Stderr)
	Debug.SetOutput(ioutil.Discard)

	cases := []struct {
		in       string
		mode     string
		options  []string
		argument string
	}{
		{"opt", "bundling", []string{}, ""},
		{"--opt", "bundling", []string{"opt"}, ""},
		{"--opt=arg", "bundling", []string{"opt"}, "arg"},
		{"-opt", "bundling", []string{"o", "p", "t"}, ""},
		{"-opt=arg", "bundling", []string{"o", "p", "t"}, "arg"},
		{"-", "bundling", []string{"-"}, ""},
		{"--", "bundling", []string{"--"}, ""},

		{"opt", "singleDash", []string{}, ""},
		{"--opt", "singleDash", []string{"opt"}, ""},
		{"--opt=arg", "singleDash", []string{"opt"}, "arg"},
		{"-opt", "singleDash", []string{"o"}, "pt"},
		{"-opt=arg", "singleDash", []string{"o"}, "pt=arg"},
		{"-", "singleDash", []string{"-"}, ""},
		{"--", "singleDash", []string{"--"}, ""},

		{"opt", "normal", []string{}, ""},
		{"--opt", "normal", []string{"opt"}, ""},
		{"--opt=arg", "normal", []string{"opt"}, "arg"},
		{"-opt", "normal", []string{"opt"}, ""},
		{"-", "normal", []string{"-"}, ""},
		{"--", "normal", []string{"--"}, ""},
	}
	for _, c := range cases {
		options, argument := isOption(c.in, c.mode)
		if !reflect.DeepEqual(options, c.options) || argument != c.argument {
			t.Errorf("isOption(%q, %q) == (%q, %q), want (%q, %q)",
				c.in, c.mode, options, argument, c.options, c.argument)
		}
	}
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
	opt.Bool("flag", false, "t")
	opt.Bool("bool", false, "t")
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
	opt.Bool("bool", false, "flag")
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
		t.Errorf("Error string didn't match expected value")
	}

	opt = New()
	opt.SetUnknownMode("fail")
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
	opt.SetUnknownMode("warn")
	remaining, err := opt.Parse([]string{"--flags"})
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if buf.String() != fmt.Sprintf(MessageOnUnknown, "flags") {
		t.Errorf("Warning message didn't match expected value: %s", buf.String())
	}
	if !reflect.DeepEqual(remaining, []string{"--flags"}) {
		t.Errorf("remaining didn't have expected value: %v != %v", remaining, []string{"--flags"})
	}

	// Tests first unknown argument as a passthrough
	buf = new(bytes.Buffer)
	opt = New()
	opt.Writer = buf
	opt.SetUnknownMode("pass")
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
	opt.SetUnknownMode("pass")
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
	opt.SetUnknownMode("pass")
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
	opt.SetUnknownMode("pass")
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
	if err != nil && err.Error() != fmt.Sprintf(ErrorMissingArgument, "string") {
		t.Errorf("Error string didn't match expected value")
	}

	// Missing argument with default
	opt = New()
	opt.StringOptional("string", "default")
	_, err = opt.Parse([]string{"--string"})
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if opt.Option("string") != "default" {
		t.Errorf("Default value not set for 'string'")
	}

	opt = New()
	opt.IntOptional("int", 123)
	_, err = opt.Parse([]string{"--int"})
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if opt.Option("int") != 123 {
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
	if opt.Option("string") != "default" {
		t.Errorf("Default value not set for 'string'")
	}
	opt = New()
	opt.StringOptional("string", "default")
	opt.IntOptional("int", 123)
	_, err = opt.Parse([]string{"--int", "--string"})
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if opt.Option("int") != 123 {
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
	if opt.Option("string") != "arg" {
		t.Errorf("string Optional didn't take argument")
	}
	if opt.Option("int") != 456 {
		t.Errorf("int Optional didn't take argument")
	}
	opt = New()
	opt.StringOptional("string", "default")
	opt.IntOptional("int", 123)
	_, err = opt.Parse([]string{"--string", "arg", "--int", "456"})
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if opt.Option("string") != "arg" {
		t.Errorf("string Optional didn't take argument")
	}
	if opt.Option("int") != 456 {
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
	if err != nil && err.Error() != fmt.Sprintf(ErrorConvertToInt, "int", "hello") {
		t.Errorf("Error string didn't match expected value '%s'", err)
	}

	opt = New()
	opt.IntOptional("int", 0)
	_, err = opt.Parse([]string{"--int", "hello"})
	if err == nil {
		t.Errorf("Int cast didn't raise errors")
	}
	if err != nil && err.Error() != fmt.Sprintf(ErrorConvertToInt, "int", "hello") {
		t.Errorf("Error string didn't match expected value '%s'", err)
	}
}

func TestGetOptBool(t *testing.T) {
	setup := func() *GetOpt {
		opt := New()
		opt.Bool("flag", false)
		opt.NBool("nflag", false)
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
			"nflag",
			[]string{"--nflag"},
			true,
		},
		{setup(),
			"nflag",
			[]string{"--no-nflag"},
			false,
		},
	}
	for _, c := range cases {
		_, err := c.opt.Parse(c.input)
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		if c.opt.Option(c.option) != c.value {
			t.Errorf("Wrong value: %v != %v", c.opt.Option(c.option), c.value)
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
		opt.Bool("flag", false, "f", "h")
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
		if c.opt.Option(c.option) != c.value {
			t.Errorf("Wrong value: %v != %v", c.opt.Option(c.option), c.value)
		}
	}

	opt := New()
	opt.Bool("flag", false)
	opt.Bool("fleg", false)
	_, err := opt.Parse([]string{"--fl"})
	if err == nil {
		t.Errorf("Ambiguous argument 'fl' didn't raise unknown option error")
	}
	if err != nil && err.Error() != "Unknown option 'fl'" {
		t.Errorf("Error string didn't match expected value")
	}

	// Bug: Startup panic when alias matches the beginning of preexisting option
	// https://github.com/DavidGamba/go-getoptions/issues/1
	opt = New()
	opt.Bool("fleg", false)
	opt.Bool("flag", false, "f")
	_, err = opt.Parse([]string{"f"})
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if opt.Called("flag") {
		t.Errorf("flag not called")
	}
}

func TestGetOptString(t *testing.T) {
	setup := func() *GetOpt {
		opt := New()
		opt.String("string", "")
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
			[]string{"--string", "hello", "world"},
			"hello",
		},
		// String should only accept an option looking string as an argument when passed after =
		{setup(),
			"string",
			[]string{"--string=--hello", "world"},
			"--hello",
		},
		// TODO: Set up a flag to decide wheter or not to err on this
		// To have the definition of string overriden. This should probably fail since it is most likely not what the user intends.
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
		if c.opt.Option(c.option) != c.value {
			t.Errorf("Wrong value: %v != %v", c.opt.Option(c.option), c.value)
		}
	}

	opt := New()
	opt.String("string", "")
	_, err := opt.Parse([]string{"--string", "--hello"})
	if err == nil {
		t.Errorf("Passing option where argument expected didn't raise error")
	}
	if err != nil && err.Error() != fmt.Sprintf(ErrorArgumentWithDash, "string") {
		t.Errorf("Error string didn't match expected value")
	}
}

func TestGetOptInt(t *testing.T) {
	setup := func() *GetOpt {
		opt := New()
		opt.Int("int", 0)
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
			[]string{"--int", "123", "world"},
			123,
		},
	}
	for _, c := range cases {
		_, err := c.opt.Parse(c.input)
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		if c.opt.Option(c.option) != c.value {
			t.Errorf("Wrong value: %v != %v", c.opt.Option(c.option), c.value)
		}
	}

	// Missing Argument errors
	opt := New()
	opt.Int("int", 0)
	_, err := opt.Parse([]string{"--int"})
	if err == nil {
		t.Errorf("Int didn't raise errors")
	}
	if err != nil && err.Error() != fmt.Sprintf(ErrorMissingArgument, "int") {
		t.Errorf("Error string didn't match expected value '%s'", err)
	}

	// Cast errors
	opt = New()
	opt.Int("int", 0)
	_, err = opt.Parse([]string{"--int=hello"})
	if err == nil {
		t.Errorf("Int cast didn't raise errors")
	}
	if err != nil && err.Error() != fmt.Sprintf(ErrorConvertToInt, "int", "hello") {
		t.Errorf("Error string didn't match expected value '%s'", err)
	}

	opt = New()
	opt.Int("int", 0)
	_, err = opt.Parse([]string{"--int", "hello"})
	if err == nil {
		t.Errorf("Int cast didn't raise errors")
	}
	if err != nil && err.Error() != fmt.Sprintf(ErrorConvertToInt, "int", "hello") {
		t.Errorf("Error string didn't match expected value '%s'", err)
	}

	opt = New()
	opt.Int("int", 0)
	_, err = opt.Parse([]string{"--int", "-123"})
	if err == nil {
		t.Errorf("Passing option where argument expected didn't raise error")
	}
	if err != nil && err.Error() != fmt.Sprintf(ErrorArgumentWithDash, "int") {
		t.Errorf("Error string didn't match expected value: %s", err.Error())
	}
}

func TestGetOptFloat64(t *testing.T) {
	setup := func() *GetOpt {
		opt := New()
		opt.Float64("float", 0)
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
			[]string{"--float", "1.23", "world"},
			1.23,
		},
	}
	for _, c := range cases {
		_, err := c.opt.Parse(c.input)
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		if c.opt.Option(c.option) != c.value {
			t.Errorf("Wrong value: %v != %v", c.opt.Option(c.option), c.value)
		}
	}

	// Missing Argument errors
	opt := New()
	opt.Float64("float", 0)
	_, err := opt.Parse([]string{"--float"})
	if err == nil {
		t.Errorf("Float64 didn't raise errors")
	}
	if err != nil && err.Error() != fmt.Sprintf(ErrorMissingArgument, "float") {
		t.Errorf("Error string didn't match expected value '%s'", err)
	}

	// Cast errors
	opt = New()
	opt.Float64("float", 0)
	_, err = opt.Parse([]string{"--float=hello"})
	if err == nil {
		t.Errorf("Float cast didn't raise errors")
	}
	if err != nil && err.Error() != fmt.Sprintf(ErrorConvertToFloat64, "float", "hello") {
		t.Errorf("Error string didn't match expected value '%s'", err)
	}

	opt = New()
	opt.Float64("float", 0)
	_, err = opt.Parse([]string{"--float", "hello"})
	if err == nil {
		t.Errorf("Int cast didn't raise errors")
	}
	if err != nil && err.Error() != fmt.Sprintf(ErrorConvertToFloat64, "float", "hello") {
		t.Errorf("Error string didn't match expected value '%s'", err)
	}

	opt = New()
	opt.Float64("float", 0)
	_, err = opt.Parse([]string{"--float", "-123"})
	if err == nil {
		t.Errorf("Passing option where argument expected didn't raise error")
	}
	if err != nil && err.Error() != fmt.Sprintf(ErrorArgumentWithDash, "float") {
		t.Errorf("Error string didn't match expected value: %s", err.Error())
	}
}

func TestGetOptStringRepeat(t *testing.T) {
	setup := func() *GetOpt {
		opt := New()
		opt.StringSlice("string")
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
			[]string{"--string=hello"},
			[]string{"hello"},
		},
		{setup(),
			"string",
			[]string{"--string=hello", "world"},
			[]string{"hello"},
		},
		{setup(),
			"string",
			[]string{"--string", "hello"},
			[]string{"hello"},
		},
		{setup(),
			"string",
			[]string{"--string", "hello", "world"},
			[]string{"hello"},
		},
		{setup(),
			"string",
			[]string{"--string", "hello", "--string", "happy", "--string", "world"},
			[]string{"hello", "happy", "world"},
		},
	}
	for _, c := range cases {
		_, err := c.opt.Parse(c.input)
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		if !reflect.DeepEqual(c.opt.Option(c.option), c.value) {
			t.Errorf("Wrong value: %v != %v", c.opt.Option(c.option), c.value)
		}
	}
	opt := New()
	opt.StringSlice("string")
	_, err := opt.Parse([]string{"--string"})
	if err == nil {
		t.Errorf("Passing option where argument expected didn't raise error")
	}
	if err != nil && err.Error() != fmt.Sprintf(ErrorMissingArgument, "string") {
		t.Errorf("Error string didn't match expected value: %s", err.Error())
	}

	opt = New()
	opt.StringSlice("string")
	_, err = opt.Parse([]string{"--string", "--hello", "world"})
	if err == nil {
		t.Errorf("Passing option where argument expected didn't raise error")
	}
	if err != nil && err.Error() != fmt.Sprintf(ErrorArgumentWithDash, "string") {
		t.Errorf("Error string didn't match expected value: %s", err.Error())
	}

	opt = New()
	ss := opt.StringSlice("string")
	_, err = opt.Parse([]string{"--string", "hello", "--string", "world"})
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if !reflect.DeepEqual(*ss, []string{"hello", "world"}) {
		t.Errorf("Wrong value: %v != %v", *ss, []string{"hello", "world"})
	}
}

// TODO: Allow passig : as the map divider
func TestGetOptStringMap(t *testing.T) {
	setup := func() *GetOpt {
		opt := New()
		opt.StringMap("string")
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
			[]string{"--string=hello=happy", "world"},
			map[string]string{"hello": "happy"},
		},
		{setup(),
			"string",
			[]string{"--string", "hello=world"},
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
	}
	for _, c := range cases {
		_, err := c.opt.Parse(c.input)
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		if !reflect.DeepEqual(c.opt.Option(c.option), c.value) {
			t.Errorf("Wrong value: %v != %v", c.opt.Option(c.option), c.value)
		}
	}
	opt := New()
	opt.StringMap("string")
	_, err := opt.Parse([]string{"--string", "hello"})
	if err != nil && err.Error() != fmt.Sprintf(ErrorArgumentIsNotKeyValue, "string") {
		t.Errorf("Error string didn't match expected value: %s", err.Error())
	}
	opt = New()
	opt.StringMap("string")
	_, err = opt.Parse([]string{"--string=hello"})
	if err != nil && err.Error() != fmt.Sprintf(ErrorArgumentIsNotKeyValue, "string") {
		t.Errorf("Error string didn't match expected value: %s", err.Error())
	}
	opt = New()
	opt.StringMap("string")
	_, err = opt.Parse([]string{"--string"})
	if err == nil {
		t.Errorf("Missing argument for option 'string' didn't raise error")
	}
	if err != nil && err.Error() != fmt.Sprintf(ErrorMissingArgument, "string") {
		t.Errorf("Error string didn't match expected value")
	}
	opt = New()
	opt.StringMap("string")
	_, err = opt.Parse([]string{"--string", "--hello=happy", "world"})
	if err == nil {
		t.Errorf("Missing argument for option 'string' didn't raise error")
	}
	if err != nil && err.Error() != fmt.Sprintf(ErrorArgumentWithDash, "string") {
		t.Errorf("Error string didn't match expected value")
	}

	opt = New()
	sm := opt.StringMap("string")
	_, err = opt.Parse([]string{"--string", "hello=world", "--string", "key=value", "--string", "key2=value2"})
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if !reflect.DeepEqual(map[string]string{"hello": "world", "key": "value", "key2": "value2"}, sm) {
		t.Errorf("Wrong value: %v != %v", map[string]string{"hello": "world", "key": "value", "key2": "value2"}, sm)
	}
	if sm["hello"] != "world" || sm["key"] != "value" || sm["key2"] != "value2" {
		t.Errorf("Wrong value: %v", sm)
	}
}

func TestGetOptStringMapMulti(t *testing.T) {
	setup := func() *GetOpt {
		opt := New()
		opt.StringMapMulti("string", 1, 3)
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
	}
	for _, c := range cases {
		_, err := c.opt.Parse(c.input)
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		if !reflect.DeepEqual(c.opt.Option(c.option), c.value) {
			t.Errorf("Wrong value: %v != %v", c.opt.Option(c.option), c.value)
		}
	}
	opt := New()
	opt.StringMapMulti("string", 1, 3)
	_, err := opt.Parse([]string{"--string", "hello"})
	if err != nil && err.Error() != fmt.Sprintf(ErrorArgumentIsNotKeyValue, "string") {
		t.Errorf("Error string didn't match expected value: %s", err.Error())
	}
	opt = New()
	opt.StringMapMulti("string", 1, 3)
	_, err = opt.Parse([]string{"--string=hello"})
	if err != nil && err.Error() != fmt.Sprintf(ErrorArgumentIsNotKeyValue, "string") {
		t.Errorf("Error string didn't match expected value: %s", err.Error())
	}
	opt = New()
	opt.StringMapMulti("string", 1, 3)
	_, err = opt.Parse([]string{"--string"})
	if err == nil {
		t.Errorf("Missing argument for option 'string' didn't raise error")
	}
	if err != nil && err.Error() != fmt.Sprintf(ErrorMissingArgument, "string") {
		t.Errorf("Error string didn't match expected value: %s", err.Error())
	}
	opt = New()
	opt.StringMapMulti("string", 1, 3)
	_, err = opt.Parse([]string{"--string", "--hello=happy", "world"})
	if err == nil {
		t.Errorf("Missing argument for option 'string' didn't raise error")
	}
	if err != nil && err.Error() != fmt.Sprintf(ErrorArgumentWithDash, "string") {
		t.Errorf("Error string didn't match expected value: %s", err.Error())
	}

	opt = New()
	sm := opt.StringMapMulti("string", 1, 3)
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
	opt.StringMapMulti("string", 2, 3)
	_, err = opt.Parse([]string{"--string", "hello=world"})
	if err == nil {
		t.Errorf("Passing less than min didn't raise error")
	}
	if err != nil && err.Error() != fmt.Sprintf(ErrorMissingArgument, "string") {
		t.Errorf("Error string didn't match expected value: %s", err.Error())
	}

	opt = New()
	opt.StringMapMulti("string", 2, 3)
	_, err = opt.Parse([]string{"--string", "hello=world", "happy"})
	if err == nil {
		t.Errorf("Passing less than min didn't raise error")
	}
	if err != nil && err.Error() != fmt.Sprintf(ErrorArgumentIsNotKeyValue, "string") {
		t.Errorf("Error string didn't match expected value: %s", err.Error())
	}
}

func TestGetOptStringSliceMulti(t *testing.T) {
	setup := func() *GetOpt {
		opt := New()
		opt.StringSliceMulti("string", 1, 3)
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
		if !reflect.DeepEqual(c.opt.Option(c.option), c.value) {
			t.Errorf("Wrong value: %v != %v", c.opt.Option(c.option), c.value)
		}
	}

	opt := New()
	opt.StringSliceMulti("string", 2, 3)
	_, err := opt.Parse([]string{"--string", "hello"})
	if err == nil {
		t.Errorf("Passing less than min didn't raise error")
	}
	if err != nil && err.Error() != fmt.Sprintf(ErrorMissingArgument, "string") {
		t.Errorf("Error string didn't match expected value")
	}
}

func TestGetOptIntSliceMulti(t *testing.T) {
	setup := func() *GetOpt {
		opt := New()
		opt.IntSliceMulti("int", 1, 3)
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
		if !reflect.DeepEqual(c.opt.Option(c.option), c.value) {
			t.Errorf("Wrong value: %v != %v", c.opt.Option(c.option), c.value)
		}
	}

	opt := New()
	opt.IntSliceMulti("int", 2, 3)
	_, err := opt.Parse([]string{"--int", "123"})
	if err == nil {
		t.Errorf("Passing less than min didn't raise error")
	}
	if err != nil && err.Error() != fmt.Sprintf(ErrorMissingArgument, "int") {
		t.Errorf("Error int didn't match expected value")
	}

	opt = New()
	opt.IntSliceMulti("int", 1, 3)
	_, err = opt.Parse([]string{"--int", "hello"})
	if err == nil {
		t.Errorf("Passing string didn't raise error")
	}
	if err != nil && err.Error() != fmt.Sprintf(ErrorConvertToInt, "int", "hello") {
		t.Errorf("Error int didn't match expected value: %s", err)
	}

	opt = New()
	opt.IntSliceMulti("int", 1, 3)
	_, err = opt.Parse([]string{"--int", "hello..3"})
	if err == nil {
		t.Errorf("Passing string didn't raise error")
	}
	if err != nil && err.Error() != fmt.Sprintf(ErrorConvertToInt, "int", "hello..3") {
		t.Errorf("Error int didn't match expected value: %s", err)
	}

	opt = New()
	opt.IntSliceMulti("int", 1, 3)
	_, err = opt.Parse([]string{"--int", "1..hello"})
	if err == nil {
		t.Errorf("Passing string didn't raise error")
	}
	if err != nil && err.Error() != fmt.Sprintf(ErrorConvertToInt, "int", "1..hello") {
		t.Errorf("Error int didn't match expected value: %s", err)
	}

	opt = New()
	opt.IntSliceMulti("int", 1, 3)
	_, err = opt.Parse([]string{"--int", "3..1"})
	if err == nil {
		t.Errorf("Passing string didn't raise error")
	}
	if err != nil && err.Error() != fmt.Sprintf(ErrorConvertToInt, "int", "3..1") {
		t.Errorf("Error int didn't match expected value: %s", err)
	}
}

// Verifies that a panic is reached when StringSliceMulti has wrong min
func TestGetOptStringSliceMultiPanicWithWrongMin(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Wrong min didn't panic")
		}
	}()
	opt := New()
	opt.StringSliceMulti("string", 0, 1)
	opt.Parse([]string{})
}

// Verifies that a panic is reached when StringMapMulti has wrong min
func TestGetOptStringMapMultiPanicWithWrongMin(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Wrong min didn't panic")
		}
	}()
	opt := New()
	opt.StringMapMulti("string", 0, 1)
	opt.Parse([]string{})
}

// Verifies that a panic is reached when IntSliceMulti has wrong min
func TestGetOptIntSliceMultiPanicWithWrongMin(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Wrong min didn't panic")
		}
	}()
	opt := New()
	opt.IntSliceMulti("int", 0, 1)
	opt.Parse([]string{})
}

// Verifies that a panic is reached when StringSliceMulti has wrong max
func TestGetOptStringSliceMultiPanicWithWrongMax(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Wrong max didn't panic")
		}
	}()
	opt := New()
	opt.StringSliceMulti("string", 2, 1)
	opt.Parse([]string{})
}

// Verifies that a panic is reached when StringMapMulti has wrong max
func TestGetOptStringMapMultiPanicWithWrongMax(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Wrong max didn't panic")
		}
	}()
	opt := New()
	opt.StringMapMulti("string", 2, 1)
	opt.Parse([]string{})
}

// Verifies that a panic is reached when IntSliceMulti has wrong max
func TestGetOptIntSliceMultiPanicWithWrongMax(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Wrong max didn't panic")
		}
	}()
	opt := New()
	opt.IntSliceMulti("int", 2, 1)
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

	var nflag, nflag2 bool
	opt.NBoolVar(&nflag, "nflag", false)
	opt.NBoolVar(&nflag2, "n2", false)

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
		"-nf",
		"--no-n2",
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

	if nflag != true {
		t.Errorf("nflag didn't have expected value: %v != %v", nflag, true)
	}
	if nflag2 != false {
		t.Errorf("nflag2 didn't have expected value: %v != %v", nflag2, false)
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
	var flag, nflag bool
	var str, str2 string
	var integer, integer2 int

	opt := New()
	opt.Bool("flag", false)
	opt.BoolVar(&flag, "varflag", false)
	opt.NBool("nflag", false)
	opt.NBoolVar(&nflag, "varnflag", false)
	opt.String("string", "")
	opt.String("string2", "default")
	str3 := opt.String("string3", "default")
	opt.StringVar(&str, "stringVar", "")
	opt.StringVar(&str2, "stringVar2", "default")
	opt.Int("int", 0)
	int2 := opt.Int("int2", 5)
	opt.IntVar(&integer, "intVar", 0)
	opt.IntVar(&integer2, "intVar2", 5)
	opt.StringSlice("string-repeat")
	opt.StringMap("string-map")

	_, err := opt.Parse([]string{})

	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}

	expected := map[string]interface{}{
		"flag":          false,
		"varflag":       false,
		"nflag":         false,
		"varnflag":      false,
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
		if !reflect.DeepEqual(opt.Option(k), expected[k]) {
			t.Errorf("Wrong value: %s\n%v !=\n%v", k, opt.Option(k), expected[k])
		}
	}

	if flag != false {
		t.Errorf("flag didn't have expected value: %v != %v", flag, true)
	}
	if nflag != false {
		t.Errorf("nflag didn't have expected value: %v != %v", nflag, true)
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

	if opt.Option("flag").(bool) {
		t.Errorf("flag didn't have expected value: %v != %v", opt.Option("flag"), false)
	}
	if opt.Option("non-used-flag") != nil && opt.Option("non-used-flag").(bool) {
		t.Errorf("non-used-flag didn't have expected value: %v != %v", opt.Option("non-used-flag"), nil)
	}
	if opt.Option("flag") != nil && opt.Option("nflag").(bool) {
		t.Errorf("nflag didn't have expected value: %v != %v", opt.Option("nflag"), nil)
	}
	if opt.Option("string") != "" {
		t.Errorf("str didn't have expected value: %v != %v", opt.Option("string"), "")
	}
	if opt.Option("int") != 0 {
		t.Errorf("int didn't have expected value: %v != %v", opt.Option("int"), 0)
	}
}

func TestBundling(t *testing.T) {
	var o, p bool
	var s string
	opt := New()
	opt.BoolVar(&o, "o", false)
	opt.BoolVar(&p, "p", false)
	opt.StringVar(&s, "t", "")
	opt.SetMode("bundling")
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
}

func TestSingleDash(t *testing.T) {
	var o string
	var p bool
	var s string
	opt := New()
	opt.StringVar(&o, "o", "")
	opt.BoolVar(&p, "p", false)
	opt.StringVar(&s, "t", "")
	opt.SetMode("singleDash")
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
	opt.IncrementVar(&i, "i", 0)
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
	opt.IncrementVar(&i, "i", 0)
	opt.IncrementVar(&j, "j", 0)
	ip = opt.Increment("ip", 0)
	_, err = opt.Parse([]string{
		"--i", "hello", "--i", "world", "--i",
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

func TestAll(t *testing.T) {
	var flag, nflag, nflag2 bool
	var str string
	var integer int
	opt := New()
	opt.Bool("flag", false)
	opt.BoolVar(&flag, "varflag", false)
	opt.Bool("non-used-flag", false)
	opt.NBool("nflag", false)
	opt.NBool("nftrue", false)
	opt.NBool("nfnil", false)
	opt.NBoolVar(&nflag, "varnflag", false)
	opt.NBoolVar(&nflag2, "varnflag2", false)
	opt.String("string", "")
	opt.StringVar(&str, "stringVar", "")
	opt.Int("int", 0)
	opt.IntVar(&integer, "intVar", 0)
	opt.StringSlice("string-repeat")
	opt.StringSliceMulti("string-slice-multi", 1, 3)
	opt.StringMap("string-map")

	remaining, err := opt.Parse([]string{
		"hello",
		"--flag",
		"--varflag",
		"--no-nflag",
		"--nft",
		"happy",
		"--varnflag",
		"--no-varnflag2",
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

	if !reflect.DeepEqual(remaining, []string{"hello", "happy", "world"}) {
		t.Errorf("remaining didn't have expected value: %v != %v", remaining, []string{"hello", "happy", "world"})
	}

	expected := map[string]interface{}{
		"flag":               true,
		"nflag":              false,
		"nftrue":             true,
		"string":             "hello",
		"int":                123,
		"string-repeat":      []string{"hello", "world"},
		"string-slice-multi": []string{"hello", "happy", "world"},
		"string-map":         map[string]string{"hello": "world", "server": "name"},
	}

	for k := range expected {
		if !reflect.DeepEqual(opt.Option(k), expected[k]) {
			t.Errorf("Wrong value: %v != %v", opt.Option(k), expected[k])
		}
	}

	if flag != true {
		t.Errorf("flag didn't have expected value: %v != %v", flag, true)
	}
	if nflag != true {
		t.Errorf("nflag didn't have expected value: %v != %v", nflag, true)
	}
	if nflag2 != false {
		t.Errorf("nflag2 didn't have expected value: %v != %v", nflag2, false)
	}
	if str != "hello" {
		t.Errorf("str didn't have expected value: %v != %v", str, "hello")
	}
	if integer != 123 {
		t.Errorf("int didn't have expected value: %v != %v", integer, 123)
	}

	// Tested above, but it gives me a feel for how it would be used

	if !opt.Option("flag").(bool) {
		t.Errorf("flag didn't have expected value: %v != %v", opt.Option("flag"), true)
	}
	if opt.Option("non-used-flag").(bool) {
		t.Errorf("non-used-flag didn't have expected value: %v != %v", opt.Option("non-used-flag"), false)
	}
	if opt.Option("nflag").(bool) {
		t.Errorf("nflag didn't have expected value: %v != %v", opt.Option("nflag"), true)
	}
	if opt.Option("string") != "hello" {
		t.Errorf("str didn't have expected value: %v != %v", opt.Option("string"), "hello")
	}
	if opt.Option("int") != 123 {
		t.Errorf("int didn't have expected value: %v != %v", opt.Option("int"), 123)
	}
}
