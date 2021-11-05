// This file is part of go-getoptions.
//
// Copyright (C) 2015-2021  David Gamba Rios
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package getoptions

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/DavidGamba/go-getoptions/option"
)

var Logger = log.New(ioutil.Discard, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)

// exitFn - This variable allows to test os.Exit calls
var exitFn = os.Exit

// completionWriter - Writer where the completion results will be written to.
// Set as a variable to allow for easy testing.
var completionWriter io.Writer = os.Stdout

type GetOpt struct {
	programTree *programTree
}

// Mode - Operation mode for short options
type Mode int

// Operation modes
const (
	Normal Mode = iota
	Bundling
	SingleDash
)

// UnknownMode - Unknown option mode
type UnknownMode int

// Unknown option modes - Action taken when an unknown option is encountered.
const (
	Fail UnknownMode = iota
	Warn
	Pass
)

// CommandFn - Function signature for commands
type CommandFn func(context.Context, *GetOpt, []string) error

// ModifyFn - Function signature for functions that modify an option.
type ModifyFn func(parent *GetOpt, option *option.Option)

// ModifyFn has to include the parent information because We want alias to be a
// global option. That is, that the user can call the top level opt.Alias from
// an option that belongs to a command or a subcommand. The problem with that
// is that if the ModifyFn signature doesn't provide information about the
// current parent we loose information about where the alias belongs to.
//
// The other complication with aliases becomes validation. Ideally, due to the
// tree nature of the command/option definition, you might want to define the
// same option with the same alias for two commands and they could do different
// things. That means that, without parent information, to write validation for
// aliases one has to navigate all leafs of the tree and validate that
// duplicates don't exist and limit functionality.

func New() *GetOpt {
	gopt := &GetOpt{}
	gopt.programTree = &programTree{
		Type:          argTypeProgname,
		Name:          os.Args[0],
		ChildCommands: map[string]*programTree{},
		ChildOptions:  map[string]*option.Option{},
		Level:         0,
	}
	return gopt
}

func (gopt *GetOpt) NewCommand(name string, description string) *GetOpt {
	cmd := &GetOpt{}
	command := &programTree{
		Type:          argTypeCommand,
		Name:          name,
		ChildCommands: map[string]*programTree{},
		ChildOptions:  map[string]*option.Option{},
		Parent:        gopt.programTree,
		Level:         gopt.programTree.Level + 1,
	}

	// Copy option definitions from parent to child
	for k, v := range gopt.programTree.ChildOptions {
		// The option parent doesn't match properly here.
		// I should in a way create a copy of the option but I still want a pointer to the data.

		// c := v.Copy() // copy that maintains a pointer to the underlying data
		// c.SetParent(command)

		// TODO: This is doing an overwrite, ensure it doesn't exist
		// command.ChildOptions[k] = c
		command.ChildOptions[k] = v
	}
	cmd.programTree = command
	gopt.programTree.AddChildCommand(name, command)
	return cmd
}

// SetCommandFn - Defines the command entry point function.
func (gopt *GetOpt) SetCommandFn(fn CommandFn) *GetOpt {
	gopt.programTree.CommandFn = fn
	return gopt
}

func (gopt *GetOpt) Parse(args []string) ([]string, error) {
	compLine := os.Getenv("COMP_LINE")
	if compLine != "" {
		// COMP_LINE has a single trailing space when the completion isn't complete and 2 when it is
		re := regexp.MustCompile(`\s+`)
		compLineParts := re.Split(compLine, -1)
		// Drop the trailing "" part if the second argument is not "". COMP_LINE alone isn't enough to tell if we are triggering a completion or not.
		if len(compLineParts) > 0 && compLineParts[len(compLineParts)-1] == "" && len(args) > 2 && args[1] != "" {
			compLineParts = compLineParts[:len(compLineParts)-1]
		}
		// Only pass an empty arg to parse when we have 2 trailing spaces indicating we are ready for the next completion.
		// if !strings.HasSuffix(compLine, "  ") && len(compLineParts) > 0 && compLineParts[len(compLineParts)-1] == "" {
		// 	compLineParts = compLineParts[:len(compLineParts)-1]
		// }
		// In some cases, the first completion only gets one space
		Logger.SetPrefix("\n")
		Logger.Printf("COMP_LINE: '%s', parts: %#v, args: %#v\n", compLine, compLineParts, args)
		_, completions, err := parseCLIArgs(true, gopt.programTree, compLineParts, Normal)
		if err != nil {
			exitFn(124) // programmable completion restarts from the beginning, with an attempt to find a new compspec for that command.

			// Ignore errors in completion mode
			return nil, nil
		}
		fmt.Fprintln(completionWriter, strings.Join(*completions, "\n"))
		exitFn(124) // programmable completion restarts from the beginning, with an attempt to find a new compspec for that command.
	}

	// WIP:
	// After we are done parsing, we know what node in the tree we are.
	// I could easily dispatch from here.
	// Think about whether or not there is value in dispatching directly from parse or if it is better to call the dispatch function.

	// TODO: parseCLIArgs needs to return the remaining array
	node, _, err := parseCLIArgs(false, gopt.programTree, args, Normal)
	if err != nil {
		return nil, err
	}

	remaining := node.ChildText

	if node.CommandFn != nil {
		err = node.CommandFn(context.Background(), &GetOpt{node}, remaining)
		if err != nil {
			return remaining, err
		}
	}

	return remaining, nil
}
