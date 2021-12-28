package getoptions

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/DavidGamba/go-getoptions/internal/option"
	"github.com/DavidGamba/go-getoptions/internal/sliceiterator"
	"github.com/DavidGamba/go-getoptions/text"
)

var ErrorMissingArgument = errors.New("")

type programTree struct {
	Type            argType
	Name            string
	Description     string
	SynopsisArgs    string
	ChildCommands   map[string]*programTree
	ChildOptions    map[string]*option.Option
	UnknownOptions  []*option.Option // Track unknown options in order in case they need to be passed to the remaining array.
	ChildText       []string
	Parent          *programTree
	Level           int
	CommandFn       CommandFn
	HelpCommandName string
	mode            Mode
	unknownMode     UnknownMode // Unknown option mode
	// Used to track option names and aliases at a global level.
	// So for example, we can have an alias p that globally refers to profile but
	// once set, no matter the level, p can't refer to password.
	// Aliases have to be globally consistent.
	GlobalOptionMap map[string]string // map[option/alias]option
	Suggestions     []string          // Suggestions used for completions

	mapKeysToLower bool // controls wether or not map keys are normalized to lowercase

	command
}

func (n *programTree) String() string {
	return n.Str()
}

// Str - not String so it doesn't get called automatically by Spew.
func (n *programTree) Str() string {
	level := n.Level
	if n.Type == argTypeOption {
		if n.Parent != nil {
			level = n.Parent.Level + 1
		}
	}
	padding := func(n int) string {
		return strings.Repeat("  ", n)
	}
	out := padding(level) + fmt.Sprintf("Name: %v, Type: %v", n.Name, n.Type)
	if n.Parent != nil {
		out += fmt.Sprintf(", Parent: %v", n.Parent.Name)
	}
	if len(n.ChildOptions) > 0 {
		var keys []string
		for k := range n.ChildOptions {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		out += ", child options: [\n"
		for _, k := range keys {
			out += padding(level+1) + fmt.Sprintf("Name: %s, Aliases: %v, Values: %v\n", n.ChildOptions[k].Name, n.ChildOptions[k].Aliases, n.ChildOptions[k].Value())
		}
		out += padding(level) + "]"
	} else {
		out += ", child options: []"
	}
	if len(n.ChildCommands) > 0 {
		var keys []string
		for k := range n.ChildCommands {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		out += ", child commands: [\n"
		for _, k := range keys {
			out += n.ChildCommands[k].Str()
		}
		out += padding(level) + "]"
	} else {
		out += ", child commands: []"
	}
	if len(n.GlobalOptionMap) > 0 {
		var keys []string
		for k := range n.GlobalOptionMap {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		out += ", option map: [\n"
		for _, k := range keys {
			out += padding(level) + fmt.Sprintf("  { %s = %s }\n", k, n.GlobalOptionMap[k])
		}
		out += padding(level) + "]"
	}
	out += "\n"
	return out
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

	if v, ok := n.GlobalOptionMap[name]; ok && v != opt.Name {
		panic(fmt.Sprintf("Option/Alias '%s' is already globally defined in option '%s'", name, v))
	}

	switch opt.OptType {
	case option.StringRepeatType, option.IntRepeatType, option.Float64RepeatType, option.StringMapType:
		err := opt.ValidateMinMaxArgs()
		if err != nil {
			panic(fmt.Sprintf("%s definition error: %s", name, err))
		}
	}

	n.ChildOptions[name] = opt
	n.GlobalOptionMap[name] = opt.Name
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

// Copy - Returns a copy of programTree that maintains a pointer to the underlying data
func (n *programTree) Copy() *programTree {
	// a := *n
	// c := &a
	parent := *n.Parent
	c := &programTree{
		Type:          n.Type,
		Name:          n.Name,
		ChildCommands: n.ChildCommands,
		ChildOptions:  n.ChildOptions,
		ChildText:     n.ChildText,
		Parent:        &parent,
	}
	return c
}

func (n *programTree) SetParent(p *programTree) *programTree {
	n.Parent = p
	return n
}

func getNode(tree *programTree, element ...string) (*programTree, error) {
	if len(element) == 0 {
		return tree, nil
	}
	if child, ok := tree.ChildCommands[element[0]]; ok {
		return getNode(child, element[1:]...)
	}
	return tree, fmt.Errorf("not found")
}

type argType int

const (
	argTypeProgname   argType = iota // The root node type
	argTypeCommand                   // The node type used for commands and subcommands
	argTypeOption                    // The node type used for options
	argTypeText                      // The node type used for regular cli arguments
	argTypeTerminator                // --
)

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
func parseCLIArgs(completionMode bool, tree *programTree, args []string, mode Mode) (*programTree, completions, error) {
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
		(completionMode && len(args) == 0) { // enter at least once if running in completion mode.

		// We only generate completions when we reached the end of the provided args
		if completionMode && (iterator.IsLast() || len(args) == 0) {
			completions := []string{}

			// Options
			{
				if strings.HasPrefix(iterator.Value(), "-") {
					var lastOpt *option.Option

					// Options are stored without leading dashes, remove them to compare
					// TODO: Also remove the / when dealing with windows.
					value := strings.TrimPrefix(strings.TrimPrefix(iterator.Value(), "-"), "-")
					for k, v := range currentProgramNode.ChildOptions {
						// handle lonesome dash
						if k == "-" {
							if iterator.Value() == "-" {
								completions = append(completions, k)
							}
							continue
						}
						if strings.HasPrefix(k, value) {
							lastOpt = v
							if currentProgramNode.ChildOptions[k].OptType != option.BoolType {
								completions = append(completions, "--"+k+"=")
							} else {
								completions = append(completions, "--"+k)
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

			// Provide other kinds of completions, like file completions.

			sort.Strings(completions)
			// Add trailing space to force next completion, makes for nicer UI when there is a single result.
			if len(completions) == 1 {
				(completions)[0] = completions[0] + " "
			}
			return currentProgramNode, completions, nil
		}

		// handle terminator
		if iterator.Value() == "--" {
			for iterator.Next() {
				value := iterator.Value()
				currentProgramNode.ChildText = append(currentProgramNode.ChildText, value)
			}
			break ARGS_LOOP
		}

		// Handle lonesome dash
		if iterator.Value() == "-" {
			for _, v := range currentProgramNode.ChildOptions {
				// handle full option match, this allows to have - defined as an alias
				if _, ok := stringSliceIndex(v.Aliases, "-"); ok {
					v.Called = true
					v.UsedAlias = "-"
					err := v.Save()
					if err != nil {
						return currentProgramNode, []string{}, err
					}
					continue ARGS_LOOP
				}
			}
			opt := newUnknownCLIOption(currentProgramNode, "-", iterator.Value())
			currentProgramNode.UnknownOptions = append(currentProgramNode.UnknownOptions, opt)
			continue ARGS_LOOP
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
					// TODO: This shouldn't append new children but update existing ones and isOption needs to be able to check if the option expects a follow up argument.
					opt := newUnknownCLIOption(currentProgramNode, p.Option, iterator.Value(), p.Args...)
					currentProgramNode.UnknownOptions = append(currentProgramNode.UnknownOptions, opt)
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
							err := fmt.Errorf(text.ErrorMissingArgument+"%w", cOpt.UsedAlias, ErrorMissingArgument)
							return currentProgramNode, []string{}, err
						}
						iterator.Next()
						if _, is := isOption(iterator.Value(), mode, false); is && !cOpt.IsOptional {
							err := fmt.Errorf(text.ErrorArgumentWithDash+"%w", cOpt.UsedAlias, ErrorMissingArgument)
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
		value := iterator.Value()
		currentProgramNode.ChildText = append(currentProgramNode.ChildText, value)
	}

	// TODO: Before returning the current node, parse EnvVars and update the values.

	// TODO: After being done parsing everything validate for errors
	// Errors can be unknown options, options without values, etc

	return currentProgramNode, []string{}, nil
}

func getAliasNameFromPartialEntry(n *programTree, entry string) []string {
	// Attempt to fully match node option
	if _, ok := n.GlobalOptionMap[entry]; ok {
		return []string{entry}
	}
	// Attempt to match initial chars of node option
	matches := []string{}
	for k := range n.GlobalOptionMap {
		if strings.HasPrefix(k, entry) {
			matches = append(matches, k)
		}
	}
	return matches
}

// TODO:
// suggestCompletions -
func suggestCompletions(tree *programTree, args []string, mode Mode) {}
