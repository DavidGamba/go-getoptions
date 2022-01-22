// This file is part of go-getoptions.
//
// Copyright (C) 2015-2022  David Gamba Rios
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package getoptions

import (
	"regexp"
	"strings"
)

// 1: leading dashes
// 2: option
// 3: =arg
var isOptionRegex = regexp.MustCompile(`^(--?)([^=]+)(.*?)$`)

// 1: leading dashes or /
// 2: option
// 3: =arg or :arg
var isOptionRegexWindows = regexp.MustCompile(`^(--?|/)([^=:]+)(.*?)$`)

type optionPair struct {
	Option string
	// We allow multiple args in case of splitting on comma.
	// TODO: Verify where we are handling integer ranges (1..10) and maybe move that logic here as well.
	Args []string
}

/*
isOption - Check if the given string is an option (starts with - or --, or / when windows support mode is set).
Return the option(s) without the starting dash and their argument if the string contained one.
The behaviour changes depending on the mode: normal, bundling or singleDash.
Also, handle the single dash '-' especial option.

The options are returned as pairs of options and arguments
At this level we don't aggregate results in case we have -- and then other options, basically we can parse one option at a time.
This makes the caller have to aggregate multiple calls to the same option.

When windows support mode is set all short options `-`, long options `--` and Windows `/` options are allowed.
Windows support mode also adds : as a valid argument indicator.
For example: /baudrate:115200 /baudrate=115200 --baudrate=115200 --baudrate:115200 are all valid.
*/
func isOption(s string, mode Mode, windows bool) ([]optionPair, bool) {
	// Handle especial cases
	switch s {
	case "--":
		// Option parsing termination (--) is not identified by isOption as an option.
		// It is the caller's responsibility.
		return []optionPair{{Option: "--"}}, false
	case "-":
		return []optionPair{{Option: "-"}}, true
	}

	var match []string
	if windows {
		match = isOptionRegexWindows.FindStringSubmatch(s)
	} else {
		match = isOptionRegex.FindStringSubmatch(s)
	}
	if len(match) > 0 {
		// check long option
		if match[1] == "--" || match[1] == "/" {
			opt := optionPair{}
			opt.Option = match[2]
			var args string
			if strings.HasPrefix(match[3], "=") {
				args = strings.TrimPrefix(match[3], "=")
			} else if strings.HasPrefix(match[3], ":") {
				args = strings.TrimPrefix(match[3], ":")
			}
			if args != "" {
				// TODO: Here is where we could split on comma
				opt.Args = []string{args}
			}
			return []optionPair{opt}, true
		}
		// check short option
		switch mode {
		case Bundling:
			opts := []optionPair{}
			for _, option := range strings.Split(match[2], "") {
				opt := optionPair{}
				opt.Option = option
				opts = append(opts, opt)
			}
			if len(opts) > 0 {
				var args string
				if strings.HasPrefix(match[3], "=") {
					args = strings.TrimPrefix(match[3], "=")
				} else if strings.HasPrefix(match[3], ":") {
					args = strings.TrimPrefix(match[3], ":")
				}
				if args != "" {
					opts[len(opts)-1].Args = []string{args}
				}
			}
			return opts, true
		case SingleDash:
			opts := []optionPair{{Option: string([]rune(match[2])[0])}}
			if len(match[2]) > 1 || len(match[3]) > 0 {
				args := string([]rune(match[2])[1:]) + match[3]
				opts[0].Args = []string{args}
			}
			return opts, true
		default:
			opt := optionPair{}
			opt.Option = match[2]
			var args string
			if strings.HasPrefix(match[3], "=") {
				args = strings.TrimPrefix(match[3], "=")
			} else if strings.HasPrefix(match[3], ":") {
				args = strings.TrimPrefix(match[3], ":")
			}
			if args != "" {
				opt.Args = []string{args}
			}
			return []optionPair{opt}, true
		}
	}
	return []optionPair{}, false
}
