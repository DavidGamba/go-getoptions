// This file is part of go-getoptions.
//
// Copyright (C) 2015-2019  David Gamba Rios
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package help

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/DavidGamba/go-getoptions/option"
	"github.com/DavidGamba/go-getoptions/text"
)

// Padding -
var Padding = 4

// TODO: This refers to os.Args[0]
// HelpName -
func HelpName(scriptName, name, description string) string {
	// scriptName := "    " + filepath.Base(os.Args[0])
	out := scriptName
	if name != "" {
		out += fmt.Sprintf(" %s", name)
	}
	if description != "" {
		out += fmt.Sprintf(" - %s", description)
	}
	return fmt.Sprintf("%s:\n%s%s\n", text.HelpNameHeader, strings.Repeat(" ", Padding), out)
}

// HelpSynopsis - Return a default synopsis.
// option list should be sorted.
// TODO: Sort list by Name
func HelpSynopsis(scriptName, name string, options []*option.Option, commands []string) string {
	// 4 spaces padding
	scriptName = "    " + scriptName
	if name != "" {
		scriptName += " " + name
	}
	normalOptions := []*option.Option{}
	requiredOptions := []*option.Option{}
	for _, option := range options {
		if option.IsRequired {
			requiredOptions = append(requiredOptions, option)
		} else {
			normalOptions = append(normalOptions, option)
		}
	}
	option.Sort(normalOptions)
	option.Sort(requiredOptions)
	optSynopsis := func(opt *option.Option) string {
		txt := ""
		aliases := []string{}
		for _, alias := range opt.Aliases {
			if len(alias) > 1 {
				aliases = append(aliases, fmt.Sprintf("--%s", alias))
			} else {
				aliases = append(aliases, fmt.Sprintf("-%s", alias))
			}
		}
		aliasStr := strings.Join(aliases, "|")
		open := ""
		close := ""
		if !opt.IsRequired {
			open = "["
			close = "]"
		}
		argName := opt.HelpArgName
		switch opt.OptType {
		case option.BoolType:
			txt += fmt.Sprintf("%s%s%s", open, aliasStr, close)
		case option.StringType, option.IntType, option.Float64Type:
			txt += fmt.Sprintf("%s%s <%s>%s", open, aliasStr, argName, close)
		case option.StringRepeatType, option.IntRepeatType, option.StringMapType:
			if opt.IsRequired {
				open = "<"
				close = ">"
			}
			repeat := ""
			if opt.MaxArgs > 1 {
				repeat = "..."
			}
			txt += fmt.Sprintf("%s%s <%s>%s%s...", open, aliasStr, argName, repeat, close)
		}
		return txt
	}
	var out string
	line := scriptName
	for _, option := range append(requiredOptions, normalOptions...) {
		syn := optSynopsis(option)
		// fmt.Printf("%d - %d - %d | %s | %s\n", len(line), len(syn), len(line)+len(syn), syn, line)
		if len(line)+len(syn) > 80 {
			out += line + "\n"
			line = fmt.Sprintf("%s %s", strings.Repeat(" ", len(scriptName)), syn)
		} else {
			line += fmt.Sprintf(" %s", syn)
		}
	}
	if len(commands) > 0 {
		syn := "<command> [<args>]"
		if len(line)+len(syn) > 80 {
			out += line + "\n"
			line = fmt.Sprintf("%s %s", strings.Repeat(" ", len(scriptName)), syn)
		} else {
			line += fmt.Sprintf(" %s", syn)
		}
	}
	out += line
	return fmt.Sprintf("%s:\n%s\n", text.HelpSynopsisHeader, out)
}

// HelpCommandList -
// commandMap => name: description
func HelpCommandList(commandMap map[string]string) string {
	if len(commandMap) <= 0 {
		return ""
	}
	names := []string{}
	for name := range commandMap {
		names = append(names, name)
	}
	sort.Strings(names)
	factor := longestStringLen(names)
	out := ""
	for _, command := range names {
		out += fmt.Sprintf("    %s    %s\n", pad(command, factor), commandMap[command])
	}
	return fmt.Sprintf("%s:\n%s", text.HelpCommandsHeader, out)
}

// longestStringLen - Given a slice of strings it returns the length of the longest string in the slice
func longestStringLen(s []string) int {
	i := 0
	for _, e := range s {
		if len(e) > i {
			i = len(e)
		}
	}
	return i
}

// pad - Given a string and a padding factor it will return the string padded with spaces.
func pad(s string, factor int) string {
	return fmt.Sprintf("%-"+strconv.Itoa(factor)+"s", s)
}

// HelpOptionList - Return a formatted list of options and their descriptions.
func HelpOptionList(options []*option.Option) string {
	aliasListLength := 0
	normalOptions := []*option.Option{}
	requiredOptions := []*option.Option{}
	for _, option := range options {
		l := len(option.Aliases)
		for _, alias := range option.Aliases {
			// --alias || -a
			l += len(alias) + 1
			if len(alias) > 1 {
				l++
			}
		}
		if l > aliasListLength {
			aliasListLength = l
		}
		if option.IsRequired {
			requiredOptions = append(requiredOptions, option)
		} else {
			normalOptions = append(normalOptions, option)
		}
	}
	option.Sort(normalOptions)
	option.Sort(requiredOptions)
	helpString := func(opt *option.Option) string {
		txt := ""
		aliases := []string{}
		for _, alias := range opt.Aliases {
			if len(alias) > 1 {
				aliases = append(aliases, fmt.Sprintf("--%s", alias))
			} else {
				aliases = append(aliases, fmt.Sprintf("-%s", alias))
			}
		}
		aliasStr := strings.Join(aliases, "|")
		// TODO: Calculate argName length.
		// 16: Longest default argName is <key=value> plus space plus 4 spaces.
		factor := aliasListLength + 16
		padding := strings.Repeat(" ", factor)
		argName := opt.HelpArgName
		switch opt.OptType {
		case option.BoolType:
			txt += fmt.Sprintf("    %s", pad(aliasStr+"", factor))
		case option.StringType, option.IntType, option.Float64Type:
			txt += fmt.Sprintf("    %s", pad(aliasStr+" <"+argName+">", factor))
		case option.StringRepeatType, option.IntRepeatType, option.StringMapType:
			txt += fmt.Sprintf("    %s", pad(aliasStr+" <"+argName+">...", factor))
		}
		if opt.Description != "" {
			description := strings.Replace(opt.Description, "\n", "\n    "+padding, -1)
			txt += fmt.Sprintf("%s ", description)
		}
		if !opt.IsRequired {
			txt += fmt.Sprintf("(default: %s)\n\n", opt.DefaultStr)
		} else {
			txt += "\n\n"
		}
		return txt
	}
	out := ""
	if len(requiredOptions) > 0 {
		out += fmt.Sprintf("%s:\n", text.HelpRequiredOptionsHeader)
		for _, option := range requiredOptions {
			out += helpString(option)
		}
	}
	if len(normalOptions) > 0 {
		out += fmt.Sprintf("%s:\n", text.HelpOptionsHeader)
		for _, option := range normalOptions {
			out += helpString(option)
		}
	}
	return out
}
