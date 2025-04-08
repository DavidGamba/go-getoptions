// This file is part of go-getoptions.
//
// Copyright (C) 2015-2025  David Gamba Rios
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package getoptions

import (
	"fmt"
	"strconv"

	"github.com/DavidGamba/go-getoptions/text"
)

// GetRequiredArg - Get the next argument from the args list and error if it doesn't exist.
// By default the error will include the HelpSynopsis section but it can be overriden with the list of sections or getoptions.HelpNone.
//
// If the arguments have been named with `opt.HelpSynopsisArg` then the error will include the argument name.
func (gopt *GetOpt) GetRequiredArg(args []string, sections ...HelpSection) (string, []string, error) {
	if len(args) < 1 {
		if len(gopt.programTree.SynopsisArgs) > gopt.programTree.SynopsisArgsIdx {
			argName := gopt.programTree.SynopsisArgs[gopt.programTree.SynopsisArgsIdx].Arg
			fmt.Fprintf(Writer, text.ErrorMissingRequiredNamedArgument+"\n", argName)
		} else {
			fmt.Fprintf(Writer, "%s\n", text.ErrorMissingRequiredArgument)
		}
		if sections != nil {
			fmt.Fprintf(Writer, "%s", gopt.Help(sections...))
		} else {
			fmt.Fprintf(Writer, "%s", gopt.Help(HelpSynopsis))
		}
		gopt.programTree.SynopsisArgsIdx++
		return "", args, ErrorHelpCalled
	}
	gopt.programTree.SynopsisArgsIdx++
	return args[0], args[1:], nil
}

// Same as GetRequiredArg but converts the argument to an int.
func (gopt *GetOpt) GetRequiredArgInt(args []string, sections ...HelpSection) (int, []string, error) {
	arg, args, err := gopt.GetRequiredArg(args, sections...)
	if err != nil {
		return 0, args, err
	}
	i, err := strconv.Atoi(arg)
	if err != nil {
		return 0, args, fmt.Errorf(text.ErrorConvertArgumentToInt, arg)
	}
	return i, args, nil
}

// Same as GetRequiredArg but converts the argument to a float64.
func (gopt *GetOpt) GetRequiredArgFloat64(args []string, sections ...HelpSection) (float64, []string, error) {
	arg, args, err := gopt.GetRequiredArg(args, sections...)
	if err != nil {
		return 0, args, err
	}
	f, err := strconv.ParseFloat(arg, 64)
	if err != nil {
		return 0, args, fmt.Errorf(text.ErrorConvertArgumentToFloat64, arg)
	}
	return f, args, nil
}
