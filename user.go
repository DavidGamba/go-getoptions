// This file is part of go-getoptions.
//
// Copyright (C) 2015-2021  David Gamba Rios
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// TODO: Handle uncomplete options

package getoptions

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/DavidGamba/go-getoptions/internal/option"
	"github.com/DavidGamba/go-getoptions/text"
)

// Logger instance set to `ioutil.Discard` by default.
// Enable debug logging by setting: `Logger.SetOutput(os.Stderr)`.
var Logger = log.New(ioutil.Discard, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)

var Writer io.Writer = os.Stderr // io.Writer to write warnings to. Defaults to os.Stderr.

// exitFn - This variable allows to test os.Exit calls
var exitFn = os.Exit

// completionWriter - Writer where the completion results will be written to.
// Set as a variable to allow for easy testing.
var completionWriter io.Writer = os.Stdout

// GetOpt - main object.
type GetOpt struct {
	// This is the main tree structure that gets build during the option and command definition
	programTree *programTree

	// This is the node that gets selected after parsing the CLI args.
	//
	// NOTE: When calling dispatch the programTree above is overwritten to be finalNode.
	//       This finalNode shouldn't be used downstream.
	finalNode *programTree
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

// New returns an empty object of type GetOpt.
// This is the starting point when using go-getoptions.
// For example:
//
//   opt := getoptions.New()
func New() *GetOpt {
	gopt := &GetOpt{}
	gopt.programTree = &programTree{
		Type:          argTypeProgname,
		Name:          filepath.Base(os.Args[0]),
		ChildCommands: map[string]*programTree{},
		ChildOptions:  map[string]*option.Option{},
		Level:         0,
	}
	return gopt
}

// TODO: Get rid of self and instead have a NewDetailed(name, description)

// Self - Set a custom name and description that will show in the automated help.
// If name is an empty string, it will only use the description and use the name as the executable name.
func (gopt *GetOpt) Self(name string, description string) *GetOpt {
	// TODO: Should this only be allowed at the root node level
	gopt.programTree.Name = name
	gopt.programTree.Description = description
	return gopt
}

// SetMode - Sets the Operation Mode.
// The operation mode only affects options starting with a single dash '-'.
// The available operation modes are: normal, bundling or singleDash.
//
// The following table shows the different operation modes given the string "-opt=arg".
//
//     .Operation Modes for string "-opt=arg"
//     |===
//     |Mode             |Description
//
//     |normal           |option: opt
//                         argument: arg
//
//     |bundling         |option: o
//                         argument: nil
//                        option: p
//                         argument: nil
//                        option: t
//                         argument: arg
//
//     |singleDash       |option: o
//                         argument: pt=arg
//
//     |===
//
// See https://github.com/DavidGamba/go-getoptions#operation_modes for more details.
func (gopt *GetOpt) SetMode(mode Mode) *GetOpt {
	gopt.programTree.mode = mode
	return gopt
}

// SetUnknownMode - Determines how to behave when encountering an unknown option.
//
// • 'fail' (default) will make 'Parse' return an error with the unknown option information.
//
// • 'warn' will make 'Parse' print a user warning indicating there was an unknown option.
// The unknown option will be left in the remaining array.
//
// • 'pass' will make 'Parse' ignore any unknown options and they will be passed onto the 'remaining' slice.
// This allows for subcommands.
// TODO: Add aliases
func (gopt *GetOpt) SetUnknownMode(mode UnknownMode) *GetOpt {
	gopt.programTree.unknownMode = mode
	return gopt
}

// SetRequireOrder - Stop parsing options when a subcommand is passed.
// Put every remaining argument, including the subcommand, in the `remaining` slice.
//
// A subcommand is assumed to be the first argument that is not an option or an argument to an option.
// When a subcommand is found, stop parsing arguments and let a subcommand handler handle the remaining arguments.
// For example:
//
//     command --opt arg subcommand --subopt subarg
//
// In the example above, `--opt` is an option and `arg` is an argument to an option, making `subcommand` the first non option argument.
//
// This method is useful when both the command and the subcommand have option handlers for the same option.
//
// For example, with:
//
//     command --help
//
// `--help` is handled by `command`, and with:
//
//     command subcommand --help
//
// `--help` is not handled by `command` since there was a subcommand that caused the parsing to stop.
// In this case, the `remaining` slice will contain `['subcommand', '--help']` and that can be passed directly to a subcommand's option parser.
// func (gopt *GetOpt) SetRequireOrder() *GetOpt {
// 	gopt.programTree.requireOrder = true
// 	return gopt
// }

// HelpSynopsisArgs - Defines the help synopsis args description.
// Defaults to: [<args>]
func (gopt *GetOpt) HelpSynopsisArgs(args string) *GetOpt {
	gopt.programTree.SynopsisArgs = args
	return gopt
}

// NewCommand - Returns a new GetOpt object representing a new command.
//
// NOTE: commands must be declared after all options are declared.
func (gopt *GetOpt) NewCommand(name string, description string) *GetOpt {
	cmd := &GetOpt{}
	command := &programTree{
		Type:            argTypeCommand,
		Name:            name,
		Description:     description,
		HelpCommandName: gopt.programTree.HelpCommandName,
		ChildCommands:   map[string]*programTree{},
		ChildOptions:    map[string]*option.Option{},
		Parent:          gopt.programTree,
		Level:           gopt.programTree.Level + 1,
		mapKeysToLower:  gopt.programTree.mapKeysToLower,
	}

	// TODO: Copying options from parent to child can't be done on declaration
	// because if an option is declared after the command then it is not part of
	// the tree.
	// However, the other side of the coin, is that if we do it in the parse call
	// then I have to wait until parse to find duplicates and panics.

	// // Copy option definitions from parent to child
	// for k, v := range gopt.programTree.ChildOptions {
	// 	// The option parent doesn't match properly here.
	// 	// I should in a way create a copy of the option but I still want a pointer to the data.
	//
	// 	// c := v.Copy() // copy that maintains a pointer to the underlying data
	// 	// c.SetParent(command)
	//
	// 	// TODO: This is doing an overwrite, ensure it doesn't exist
	// 	// command.ChildOptions[k] = c
	// 	command.ChildOptions[k] = v
	// }

	cmd.programTree = command
	gopt.programTree.AddChildCommand(name, command)
	copyOptionsFromParent(gopt.programTree, false)
	return cmd
}

func copyOptionsFromParent(parent *programTree, fail bool) {
	for k, v := range parent.ChildOptions {
		for _, command := range parent.ChildCommands {
			// don't copy options to help command
			if command.Name == parent.HelpCommandName {
				continue
			}
			if fail {
				if _, ok := command.ChildOptions[k]; ok {
					panic("duplicate option definition")
				}
			}
			command.ChildOptions[k] = v
		}
	}
	for _, command := range parent.ChildCommands {
		copyOptionsFromParent(command, fail)
	}
}

// SetCommandFn - Defines the command entry point function.
func (gopt *GetOpt) SetCommandFn(fn CommandFn) *GetOpt {
	gopt.programTree.CommandFn = fn
	return gopt
}

// Parse - Call the parse method when done describing.
// It will operate on any given slice of strings and return the remaining (non
// used) command line arguments.
// This allows to easily subcommand.
//
// Parsing style is controlled by the `Set` methods (SetMode, SetRequireOrder, etc).
//     // Declare the GetOptions object
//     opt := getoptions.New()
//     ...
//     // Parse cmdline arguments or any provided []string
//     remaining, err := opt.Parse(os.Args[1:])
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
			fmt.Fprintf(Writer, "\nERROR: %s\n", err)
			exitFn(124) // programmable completion restarts from the beginning, with an attempt to find a new compspec for that command.

			// Ignore errors in completion mode
			return nil, nil
		}
		fmt.Fprintln(completionWriter, strings.Join(completions, "\n"))
		exitFn(124) // programmable completion restarts from the beginning, with an attempt to find a new compspec for that command.
	}

	// WIP:
	// After we are done parsing, we know what node in the tree we are.
	// I could easily dispatch from here.
	// Think about whether or not there is value in dispatching directly from parse or if it is better to call the dispatch function.
	// I came up with the conclusion that dispatch provides a bunch of flexibility and explicitness.

	// TODO: parseCLIArgs needs to return the remaining array
	node, _, err := parseCLIArgs(false, gopt.programTree, args, gopt.programTree.mode)
	if err != nil {
		return nil, err
	}
	gopt.finalNode = node

	// Only validate required options at the parse call when the final node is the parent
	// This to enable handling the help option in a command
	if gopt.finalNode.Parent == nil {
		// Validate required options
		for _, option := range node.ChildOptions {
			err := option.CheckRequired()
			if err != nil {
				return nil, err
			}
		}
	}

	for _, option := range node.UnknownOptions {
		switch gopt.programTree.unknownMode {
		case Fail:
			return nil, fmt.Errorf(text.MessageOnUnknown, option.Name)
		case Warn:
			fmt.Fprintf(Writer, text.WarningOnUnknown+"\n", option.Name)
		}
		node.ChildText = append(node.ChildText, option.Verbatim)
	}

	// TODO: Validate unknown options

	return node.ChildText, nil
}

// Dispatch - Handles calling commands and subcommands after the call to Parse.
func (gopt *GetOpt) Dispatch(ctx context.Context, remaining []string) error {
	if gopt.finalNode.HelpCommandName != "" && gopt.Called(gopt.finalNode.HelpCommandName) {
		fmt.Fprint(Writer, helpOutput(gopt.finalNode))
		return ErrorHelpCalled
	}
	// Validate required options
	for _, option := range gopt.finalNode.ChildOptions {
		err := option.CheckRequired()
		if err != nil {
			return err
		}
	}
	if gopt.finalNode.CommandFn != nil {
		return gopt.finalNode.CommandFn(ctx, &GetOpt{gopt.finalNode, gopt.finalNode}, remaining)
	}
	if gopt.finalNode.Parent != nil {
		// TODO: This should probably panic at the parse call with validation instead of waiting for a runtime error.
		// Then in that case it makes sense to just make it as a required argument in the constructor.
		// Though empty could be considered as a help landing to avoid having to create one...
		return fmt.Errorf("command '%s' has no defined CommandFn", gopt.finalNode.Name)
	}
	fmt.Fprint(Writer, gopt.Help())
	return nil
}
