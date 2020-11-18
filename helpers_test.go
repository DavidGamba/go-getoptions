// This file is part of go-getoptions.
//
// Copyright (C) 2015-2020  David Gamba Rios
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
package getoptions

import (
	"reflect"
	"testing"
)

func TestIsOption(t *testing.T) {
	cases := []struct {
		name     string
		in       string
		mode     Mode
		options  []string
		argument string
	}{
		{"", "opt", Bundling, []string{}, ""},
		{"", "--opt", Bundling, []string{"opt"}, ""},
		{"", "--opt=arg", Bundling, []string{"opt"}, "arg"},
		{"", "-opt", Bundling, []string{"o", "p", "t"}, ""},
		{"", "-opt=arg", Bundling, []string{"o", "p", "t"}, "arg"},
		{"", "-", Bundling, []string{"-"}, ""},
		{"", "--", Bundling, []string{"--"}, ""},

		{"", "opt", SingleDash, []string{}, ""},
		{"", "--opt", SingleDash, []string{"opt"}, ""},
		{"", "--opt=arg", SingleDash, []string{"opt"}, "arg"},
		{"", "-opt", SingleDash, []string{"o"}, "pt"},
		{"", "-opt=arg", SingleDash, []string{"o"}, "pt=arg"},
		{"", "-", SingleDash, []string{"-"}, ""},
		{"", "--", SingleDash, []string{"--"}, ""},

		{"", "opt", Normal, []string{}, ""},
		{"", "--opt", Normal, []string{"opt"}, ""},
		{"", "--opt=arg", Normal, []string{"opt"}, "arg"},
		{"", "-opt", Normal, []string{"opt"}, ""},
		{"", "-", Normal, []string{"-"}, ""},
		{"", "--", Normal, []string{"--"}, ""},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			buf := setupLogging()
			options, argument, _ := isOption(tt.in, tt.mode)
			if !reflect.DeepEqual(options, tt.options) || argument != tt.argument {
				t.Errorf("isOption(%q, %q) == (%q, %q), want (%q, %q)",
					tt.in, tt.mode, options, argument, tt.options, tt.argument)
			}
			t.Log(buf.String())
		})
	}
}

func TestIsOptionV2(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		mode    Mode
		optPair []optionPair
		is      bool
	}{
		{"lone dash", "-", Normal, []optionPair{{Option: "-"}}, true},
		{"lone dash", "-", Bundling, []optionPair{{Option: "-"}}, true},
		{"lone dash", "-", SingleDash, []optionPair{{Option: "-"}}, true},

		{"double dash", "--", Normal, []optionPair{{Option: "--"}}, false},
		{"double dash", "--", Bundling, []optionPair{{Option: "--"}}, false},
		{"double dash", "--", SingleDash, []optionPair{{Option: "--"}}, false},

		{"no option", "opt", Normal, []optionPair{}, false},
		{"no option", "opt", Bundling, []optionPair{}, false},
		{"no option", "opt", SingleDash, []optionPair{}, false},

		{"Long option", "--opt", Normal, []optionPair{{Option: "opt"}}, true},
		{"Long option", "--opt", Bundling, []optionPair{{Option: "opt"}}, true},
		{"Long option", "--opt", SingleDash, []optionPair{{Option: "opt"}}, true},

		{"Long option with arg", "--opt=arg", Normal, []optionPair{{Option: "opt", Args: []string{"arg"}}}, true},
		{"Long option with arg", "--opt=arg", Bundling, []optionPair{{Option: "opt", Args: []string{"arg"}}}, true},
		{"Long option with arg", "--opt=arg", SingleDash, []optionPair{{Option: "opt", Args: []string{"arg"}}}, true},

		{"short option", "-opt", Normal, []optionPair{{Option: "opt"}}, true},
		{"short option", "-opt", Bundling, []optionPair{{Option: "o"}, {Option: "p"}, {Option: "t"}}, true},
		{"short option", "-opt", SingleDash, []optionPair{{Option: "o", Args: []string{"pt"}}}, true},

		{"short option with arg", "-opt=arg", Normal, []optionPair{{Option: "opt", Args: []string{"arg"}}}, true},
		{"short option with arg", "-opt=arg", Bundling, []optionPair{{Option: "o"}, {Option: "p"}, {Option: "t", Args: []string{"arg"}}}, true},
		{"short option with arg", "-opt=arg", SingleDash, []optionPair{{Option: "o", Args: []string{"pt=arg"}}}, true},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			buf := setupLogging()
			optPair, is := isOptionV2(tt.in, tt.mode)
			if !reflect.DeepEqual(optPair, tt.optPair) || is != tt.is {
				t.Errorf("isOption(%q, %s) == (%q, %v), want (%q, %v)",
					tt.in, tt.mode, optPair, is, tt.optPair, tt.is)
			}
			t.Log(buf.String())
		})
	}
}
