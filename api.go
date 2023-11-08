// This file is part of go-getoptions.
//
// Copyright (C) 2015-2024  David Gamba Rios
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package getoptions

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/DavidGamba/go-getoptions/internal/help"
	"github.com/DavidGamba/go-getoptions/internal/option"
	"github.com/DavidGamba/go-getoptions/internal/sliceiterator"
	"github.com/DavidGamba/go-getoptions/text"
)

type programTree struct {
	Type            argType
	Name            string
	Description     string
	SynopsisArgs    []help.SynopsisArg
	SynopsisArgsIdx int // idx for the GetRequiredArg helper
	ChildCommands   map[string]*programTree
	ChildOptions    map[string]*option.Option
	UnknownOptions  []*option.Option // Track unknown options in order in case they need to be passed to the remaining array.
	ChildText       []string
	Parent          *programTree
	Level           int
	CommandFn       CommandFn
	HelpCommandName string
	mode            Mode
	unknownMode     UnknownMode    // Unknown option mode
	requireOrder    bool           // stop parsing args as soon as an unknown is found
	skipOptionsCopy bool           // skips copying options from parent to child. Required when doing wrapper commands.
	Suggestions     []string       // Suggestions used for completions
	SuggestionFns   []CompletionFn // SuggestionsFns used for completions

	mapKeysToLower bool // controls wether or not map keys are normalized to lowercase

	command
}

type strProgramTree struct {
	Name          string
	Parent        string
	Type          string
	ChildOptions  map[string]strChildOption
	ChildCommands map[string]*strProgramTree
}

type strChildOption struct {
	Aliases   []string
	Value     string
	UsedAlias string
	Called    bool
}

func (n *programTree) str() *strProgramTree {
	str := &strProgramTree{}
	str.ChildOptions = make(map[string]strChildOption)
	str.ChildCommands = make(map[string]*strProgramTree)

	str.Name = n.Name
	if n.Parent != nil {
		str.Parent = n.Parent.Name
	}
	str.Type = fmt.Sprintf("%v", n.Type)

	// options
	var options []string
	for k := range n.ChildOptions {
		options = append(options, k)
	}
	sort.Strings(options)
	for _, e := range options {
		str.ChildOptions[e] = strChildOption{
			Aliases:   n.ChildOptions[e].Aliases,
			Value:     fmt.Sprintf("%v", n.ChildOptions[e].Value()),
			UsedAlias: n.ChildOptions[e].UsedAlias,
			Called:    n.ChildOptions[e].Called,
		}
	}

	// commands
	var commands []string
	for k := range n.ChildCommands {
		commands = append(commands, k)
	}
	sort.Strings(commands)
	for _, k := range commands {
		str.ChildCommands[k] = n.ChildCommands[k].str()
	}
	return str
}

// AddChildOption - Adds child options to programTree and runs validations.
func (n *programTree) AddChildOption(name string, opt *option.Option) {
	// Design choice:
	// 1. Create a flat structure where aliases are part of the map and they point to the option.
	// 2. Create a layered structure where the ChildOptions point to the name of
	//    the option and to get the alias we need to traverse all options.
	//
	// 1 seems simpler to work with long term. It is easy to determine it is an alias because key != value.Name

	if name == "" {
		panic("Option/Alias name can't be empty")
	}

	if v, ok := n.ChildOptions[name]; ok {
		panic(fmt.Sprintf("Option/Alias '%s' is already defined in option '%s'", name, v.Name))
	}

	switch opt.OptType {
	case option.StringRepeatType, option.IntRepeatType, option.Float64RepeatType, option.StringMapType:
		err := opt.ValidateMinMaxArgs()
		if err != nil {
			panic(fmt.Sprintf("%s definition error: %s", name, err))
		}
	}

	n.ChildOptions[name] = opt
}

// AddChildOption - Adds child commands to programTree and runs validations.
func (n *programTree) AddChildCommand(name string, cmd *programTree) {
	if name == "" {
		panic("Command name can't be empty")
	}

	if v, ok := n.ChildCommands[name]; ok {
		panic(fmt.Sprintf("Command '%s' is already defined in command '%s'", name, v.Name))
	}
	n.ChildCommands[name] = cmd
}

type argType int

// command - Fields that only make sense for a command
type command struct {
	CommandFn CommandFn
}

// TODO: Make this a method of tree so we can add parent information.
// Maybe not a good idea? Would it complicate testing?
// newUnknownCLIOption - attaches a new CLI option to the parent that is labelled as unknown for later handling.
func newUnknownCLIOption(parent *programTree, name, verbatim string, args ...string) *option.Option {
	data := []string{}
	data = append(data, args...)
	arg := option.New(name, option.StringRepeatType, &data)
	arg.Unknown = true
	arg.Verbatim = verbatim
	return arg
}

type completions []string

// parseCLIArgs - Given the root node tree and the cli args it returns a populated tree of the node that was called.
// For example, if a command is called, then the returned node is that of the command with the options that were set updated with their values.
// Additionally, when in completion mode, it returns the possible completions
func parseCLIArgs(completionMode string, tree *programTree, args []string, mode Mode) (*programTree, completions, error) {
	// Design: This function could return an array or CLIargs as a parse result
	// or I could do one level up and have a root CLIarg type with the name of
	// the program.  Having the root level might be helpful with help generation.

	// The current implementation expects os.Args[1:] as an argument so this
	// can't expect to receive the os.Args[0] as the first argument.

	// CLI arguments are split by spaces by the shell and passed as individual
	// strings.  In most cases, a cli argument (one string) represents one option
	// or one argument, however, in the case of bundling mode a single string can
	// represent multiple options.

	// Ensure consistent response for empty and nil slices
	if args == nil {
		args = []string{}
	}

	currentProgramNode := tree

	iterator := sliceiterator.New(&args)

ARGS_LOOP:
	for iterator.Next() ||
		(completionMode != "" && len(args) == 0) { // enter at least once if running in completion mode.

		///////////////////////////////////
		// Completions
		///////////////////////////////////

		// We only generate completions when we reached the end of the provided args
		if completionMode != "" && (iterator.IsLast() || len(args) == 0) {
			completions := []string{}

			// Options
			{
				if strings.HasPrefix(iterator.Value(), "-") {
					var lastOpt *option.Option

					// Options are stored without leading dashes, remove them to compare
					// TODO: Also remove the / when dealing with windows.
					partialOption := strings.TrimPrefix(strings.TrimPrefix(iterator.Value(), "-"), "-")
					// value = strings.SplitN(value, "=", 2)[0]
					for k, v := range currentProgramNode.ChildOptions {
						// handle lonesome dash
						if k == "-" {
							if iterator.Value() == "-" {
								completions = append(completions, k)
							}
							continue
						}
						// The entry is not fully complete here
						if strings.HasPrefix(k, partialOption) {
							lastOpt = v
							if currentProgramNode.ChildOptions[k].OptType != option.BoolType {
								completions = append(completions, "--"+k+`=`)
							} else {
								completions = append(completions, "--"+k)
							}
						}
						// The entry is complete here and has suggestions
						if strings.Contains(partialOption, "=") && strings.HasPrefix(partialOption, k) {
							lastOpt = v
							if lastOpt.SuggestedValues != nil && len(lastOpt.SuggestedValues) > 0 {
								for _, e := range lastOpt.SuggestedValues {
									c := fmt.Sprintf("--%s=%s", k, e)
									if strings.HasPrefix(c, iterator.Value()) {
										// NOTE: Bash completions have = as a special char and results should be trimmed form the = on.
										if completionMode == "bash" {
											tc := strings.SplitN(c, "=", 2)[1]
											completions = append(completions, tc)
										} else {
											completions = append(completions, c)
										}
									}
								}
							}
						}
					}
					sort.Strings(completions)

					// If there is a single completion and it expects an argument, add an
					// extra completion so there is no trailing space automatically
					// inserted by bash.
					// This extra completion has nice documentation on what the option expects.
					if len(completions) == 1 && strings.HasSuffix((completions)[0], "=") {
						if lastOpt.SuggestedValues != nil && len(lastOpt.SuggestedValues) > 0 {
							for _, e := range lastOpt.SuggestedValues {
								completions = append(completions, completions[0]+e)
							}
						} else {
							valueStr := "<value>"
							if lastOpt.HelpArgName != "" {
								valueStr = "<" + lastOpt.HelpArgName + ">"
							}
							completions = append(completions, completions[0]+valueStr)
						}
					}

					sort.Strings(completions)
					return currentProgramNode, completions, nil
				}
			}

			// Commands
			{
				// Iterate over commands and check prefix to see if we offer command completion
				for k := range currentProgramNode.ChildCommands {
					if strings.HasPrefix(k, iterator.Value()) {
						completions = append(completions, k)
					}
				}
			}

			// Suggestions
			{
				for _, e := range currentProgramNode.Suggestions {
					if strings.HasPrefix(e, iterator.Value()) {
						completions = append(completions, e)
					}
				}
			}
			// SuggestionFns
			{
				for _, fn := range currentProgramNode.SuggestionFns {
					completions = append(completions, fn(completionMode, iterator.Value())...)
				}
			}

			// Provide other kinds of completions, like file completions.

			sort.Strings(completions)
			// Add trailing space to force next completion, makes for nicer UI when there is a single result.
			// In most cases this is not required but sometimes the compspec just seems to get stuck.
			if len(completions) == 1 && completionMode == "bash" {
				(completions)[0] = completions[0] + " "
			}
			return currentProgramNode, completions, nil
		}

		///////////////////////////////////
		// Normal parsing
		///////////////////////////////////

		// handle terminator
		if iterator.Value() == "--" {
			// iterate over --
			if iterator.Next() {
				storeRemainingAsText(iterator, currentProgramNode)
			}
			break ARGS_LOOP
		}

		// TODO: Handle unknown option.
		// It basically needs to be copied down to the command every time we find a command and it has to be validated against aliases and option name.
		// If we were to check on require order and other modes without doing that work, passing --help after passing an unknown option would return an unknown option error and it would be annoying to the user.

		// TODO: Handle case where option has an argument
		// check for option

		// isOption should check if a cli argument starts with -.
		// If it does, we validate that it matches an option.
		// If it does we update the option with the values that might have been provided on the CLI.
		//
		// We almost need to build a separate option tree which allows unknown options and then update the main tree when we are done parsing cli args.
		//
		// Currently go-getoptions has no knowledge of command options at the
		// parents so it marks them as an unknown option that needs to be used at a
		// different level. It is as if it was ignoring getoptions.Pass.
		if optPair, is := isOption(iterator.Value(), mode, false); is {

			// iterate over the possible cli args and try matching against expectations
			for _, p := range optPair {
				// handle full option match
				optionMatches := getAliasNameFromPartialEntry(currentProgramNode, p.Option)
				if len(optionMatches) > 1 {
					sort.Strings(optionMatches)
					err := fmt.Errorf(text.ErrorAmbiguousArgument, iterator.Value(), optionMatches)
					return currentProgramNode, []string{}, err
				}

				if len(optionMatches) == 0 {
					if currentProgramNode.requireOrder {
						storeRemainingAsText(iterator, currentProgramNode)
						break ARGS_LOOP
					}
					// TODO: This shouldn't append new children but update existing ones and isOption needs to be able to check if the option expects a follow up argument.
					opt := newUnknownCLIOption(currentProgramNode, p.Option, iterator.Value(), p.Args...)
					currentProgramNode.UnknownOptions = append(currentProgramNode.UnknownOptions, opt)

					switch currentProgramNode.unknownMode {
					case Pass, Warn:
						currentProgramNode.ChildText = append(currentProgramNode.ChildText, iterator.Value())
					}
					continue
				}
				// TODO: Check min, check max and keep ingesting until something starts with `-` or matches a command.

				if cOpt, ok := currentProgramNode.ChildOptions[optionMatches[0]]; ok {
					cOpt.Called = true
					cOpt.UsedAlias = optionMatches[0]
					cOpt.MapKeysToLower = tree.mapKeysToLower
					err := cOpt.Save(p.Args...)
					if err != nil {
						return currentProgramNode, []string{}, err
					}
					// TODO: Handle option having a minimum bigger than 1

					// Validate minimum
					i := len(p.Args) // if the value is part of the option, for example --opt=value then the minimum of 1 is already met.
					for ; i < cOpt.MinArgs; i++ {
						if !iterator.ExistsNext() && !cOpt.IsOptional {
							err := fmt.Errorf(text.ErrorMissingArgument+"%w", cOpt.UsedAlias, ErrorParsing)
							return currentProgramNode, []string{}, err
						}
						iterator.Next()
						if _, is := isOption(iterator.Value(), mode, false); is && !cOpt.IsOptional {
							err := fmt.Errorf(text.ErrorArgumentWithDash+"%w", cOpt.UsedAlias, ErrorParsing)
							return currentProgramNode, []string{}, err
						}
						err := cOpt.Save(iterator.Value())
						if err != nil {
							return currentProgramNode, []string{}, err
						}
					}

				MAX_LOOP:
					// Run maximun
					for ; i < cOpt.MaxArgs; i++ {
						if !iterator.ExistsNext() {
							break
						}
						value, _ := iterator.PeekNextValue()
						if _, is := isOption(value, mode, false); is {
							break
						}

						// Validate that value matches expected format
						switch cOpt.OptType {
						case option.StringRepeatType:
						// TODO: Should we validate that argument doesn't match a command?
						// nothing to do here
						case option.IntRepeatType:
							// Next Value is not an int entry, break the max feed.
							_, err := strconv.Atoi(value)
							if err != nil {
								break MAX_LOOP
							}
						case option.Float64RepeatType:
							// Next Value is not a float64 entry, break the max feed.
							_, err := strconv.ParseFloat(value, 64)
							if err != nil {
								break MAX_LOOP
							}
						case option.StringMapType:
							// Next Value is not a key=value entry, break the max feed.
							if !strings.Contains(value, "=") {
								break MAX_LOOP
							}
						}

						iterator.Next()
						err := cOpt.Save(iterator.Value())
						if err != nil {
							return currentProgramNode, []string{}, err
						}
					}
				}
			}
			continue ARGS_LOOP
		}

		// When handling options out of order, iterate over all possible options for all the children and set them if they match.
		// That means that the option has to match the alias and aliases need to be non ambiguous with the parent.
		// partial options can only be applied if they match a single possible option in the tree.
		// Since at the end we return the programTree node, we will only care about handling the options at one single level.

		// handle commands and subcommands
		for k, v := range currentProgramNode.ChildCommands {
			if k == iterator.Value() {
				currentProgramNode = v
				continue ARGS_LOOP
			}
		}

		// handle text
		if currentProgramNode.requireOrder {
			storeRemainingAsText(iterator, currentProgramNode)
			break ARGS_LOOP
		}
		value := iterator.Value()
		currentProgramNode.ChildText = append(currentProgramNode.ChildText, value)
	}

	// TODO: Before returning the current node, parse EnvVars and update the values.

	// TODO: After being done parsing everything validate for errors
	// Errors can be unknown options, options without values, etc

	return currentProgramNode, []string{}, nil
}

func storeRemainingAsText(iterator *sliceiterator.Iterator, n *programTree) {
	value := iterator.Value()
	n.ChildText = append(n.ChildText, value)
	for iterator.Next() {
		value := iterator.Value()
		n.ChildText = append(n.ChildText, value)
	}
}

func getAliasNameFromPartialEntry(n *programTree, entry string) []string {
	// Attempt to fully match node option
	if _, ok := n.ChildOptions[entry]; ok {
		return []string{entry}
	}
	// Attempt to match initial chars of node option
	matches := []string{}
	for k := range n.ChildOptions {
		if strings.HasPrefix(k, entry) {
			matches = append(matches, k)
		}
	}
	return matches
}
