// This file is part of go-getoptions.
//
// Copyright (C) 2015-2020  David Gamba Rios
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

/*
isOption - Check if the given string is an option (starts with - or --).
Return the option(s) without the starting dash and an argument if the string contained one.
The behaviour changes depending on the mode: normal, bundling or singleDash.
Also, handle the single dash '-' and double dash '--' especial options.
*/
func isOption(s string, mode Mode) (options []string, argument string, is bool) {
	// Handle especial cases
	if s == "--" {
		return []string{"--"}, "", false
	} else if s == "-" {
		return []string{"-"}, "", true
	}

	match := isOptionRegex.FindStringSubmatch(s)
	if len(match) > 0 {
		// check long option
		if match[1] == "--" {
			options = []string{match[2]}
			argument = strings.TrimPrefix(match[3], "=")
			is = true
			return
		}
		switch mode {
		case Bundling:
			options = strings.Split(match[2], "")
			argument = strings.TrimPrefix(match[3], "=")
			is = true
		case SingleDash:
			options = []string{strings.Split(match[2], "")[0]}
			argument = strings.Join(strings.Split(match[2], "")[1:], "") + match[3]
			is = true
		default:
			options = []string{match[2]}
			argument = strings.TrimPrefix(match[3], "=")
			is = true
		}
		return
	}
	return []string{}, "", false
}

type optionPair struct {
	Option string
	// We allow multiple args in case of splitting on comma.
	Args []string
}

// isOptionV2 - Enhanced version of isOption, this one returns pairs of options and arguments
// At this level we don't agregate results in case we have -- and then other options, basically we can parse one option at a time.
// This makes the caller have to agregate multiple calls to the same option.
func isOptionV2(s string, mode Mode) ([]optionPair, bool) {
	// Handle especial cases
	if s == "--" {
		return []optionPair{{Option: "--"}}, false
	} else if s == "-" {
		return []optionPair{{Option: "-"}}, true
	}

	match := isOptionRegex.FindStringSubmatch(s)
	if len(match) > 0 {
		// check long option
		if match[1] == "--" {
			opt := optionPair{}
			opt.Option = match[2]
			args := strings.TrimPrefix(match[3], "=")
			if args != "" {
				opt.Args = []string{args}
			}
			return []optionPair{opt}, true
		}
		switch mode {
		case Bundling:
			opts := []optionPair{}
			for _, option := range strings.Split(match[2], "") {
				opt := optionPair{}
				opt.Option = option
				opts = append(opts, opt)
			}
			if len(opts) > 0 {
				args := strings.TrimPrefix(match[3], "=")
				if args != "" {
					opts[len(opts)-1].Args = []string{args}
				}
			}
			return opts, true
		case SingleDash:
			opts := []optionPair{}
			for _, option := range []string{strings.Split(match[2], "")[0]} {
				opt := optionPair{}
				opt.Option = option
				opts = append(opts, opt)
			}
			if len(opts) > 0 {
				args := strings.Join(strings.Split(match[2], "")[1:], "") + match[3]
				opts[len(opts)-1].Args = []string{args}
			}
			return opts, true
		default:
			opt := optionPair{}
			opt.Option = match[2]
			args := strings.TrimPrefix(match[3], "=")
			if args != "" {
				opt.Args = []string{args}
			}
			return []optionPair{opt}, true
		}
	}
	return []optionPair{}, false
}
