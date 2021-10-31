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
	"os/signal"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"

	"github.com/DavidGamba/go-getoptions/completion"
	"github.com/DavidGamba/go-getoptions/help"
	"github.com/DavidGamba/go-getoptions/option"
	"github.com/DavidGamba/go-getoptions/text"
)

// Debug Logger instance set to `ioutil.Discard` by default.
// Enable debug logging by setting: `Debug.SetOutput(os.Stderr)`.
var Debug = log.New(ioutil.Discard, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)

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

// Unknown option modes
const (
	Fail UnknownMode = iota
	Warn
	Pass
)

// HelpSection - Indicates what portion of the help to return.
type HelpSection int

// Help Output Types
const (
	helpDefaultName HelpSection = iota
	HelpName
	HelpSynopsis
	HelpCommandList
	HelpOptionList
)

// ErrorHelpCalled - Indicates the help has been handled.
var ErrorHelpCalled = fmt.Errorf("help called")

// exitFn - This variable allows to test os.Exit calls
var exitFn = os.Exit

// completionWriter - Writer where the completion results will be written to.
// Set as a variable to allow for easy testing.
var completionWriter io.Writer = os.Stdout

// GetOpt - main object.
type GetOpt struct {
	// Help fields
	name         string
	description  string
	synopsisArgs string
	selfCalled   bool

	// isCommand
	isCommand bool
	// CommandFn
	CommandFn CommandFn
	// Parent object
	parent *GetOpt

	// Option handling
	// TODO: Option handling should trickle down to commands.
	mode           Mode        // Operation mode for short options: normal, bundling, singleDash
	unknownMode    UnknownMode // Unknown option mode
	requireOrder   bool        // Stop parsing on non option
	mapKeysToLower bool        // Set Map keys lower case

	// Debugging
	Writer io.Writer // io.Writer to write warnings to. Defaults to os.Stderr.

	// Data
	obj        map[string]*option.Option // indexed options
	commands   map[string]*GetOpt
	args       *argList
	completion *completion.Node
}

// ModifyFn - Function signature for functions that modify an option.
type ModifyFn func(*option.Option)

// CommandFn - Function signature for commands
type CommandFn func(context.Context, *GetOpt, []string) error

// New returns an empty object of type GetOpt.
// This is the starting point when using go-getoptions.
// For example:
//
//   opt := getoptions.New()
func New() *GetOpt {
	root := completion.NewNode("root", completion.Root, nil)
	root.AddChild(completion.NewNode("options", completion.OptionsNode, nil))
	root.AddChild(completion.NewNode("options-with-arg", completion.OptionsWithCompletion, nil))
	gopt := &GetOpt{
		name:       filepath.Base(os.Args[0]),
		obj:        make(map[string]*option.Option),
		commands:   make(map[string]*GetOpt),
		Writer:     os.Stderr,
		completion: root,
	}
	return gopt
}

// NewCommand - Returns a new GetOpt object representing a new command.
func (gopt *GetOpt) NewCommand(name string, description string) *GetOpt {
	if name == "" {
		panic("NewCommand name must not be empty!")
	}
	cmd := New()
	cmd.isCommand = true
	cmd.name = name
	cmd.description = description
	cmd.parent = gopt

	// TODO: Ensure aliases are gettint validated

	// Completion
	node := cmd.completion
	node.Kind = completion.CommandNode
	node.Name = cmd.name
	gopt.completion.AddChild(node)
	gopt.commands[cmd.name] = cmd

	return cmd
}

// SetCommandFn - Defines the command entry point function.
func (gopt *GetOpt) SetCommandFn(fn CommandFn) *GetOpt {
	gopt.CommandFn = fn
	return gopt
}

func (gopt *GetOpt) completionAppendAliases(aliases []string) {
	node := gopt.completion.GetChildByName("options")
	for _, alias := range aliases {
		if len(alias) == 1 {
			node.Entries = append(node.Entries, "-"+alias)
		} else {
			node.Entries = append(node.Entries, "--"+alias)
		}
	}
}

func (gopt *GetOpt) completionWithArgAppendAliases(aliases []string) {
	node := gopt.completion.GetChildByName("options-with-arg")
	for _, alias := range aliases {
		if len(alias) == 1 {
			node.Entries = append(node.Entries, "-"+alias)
		} else {
			node.Entries = append(node.Entries, "--"+alias)
		}
	}
}

// Self - Set a custom name and description that will show in the automated help.
// If name is an empty string, it will only use the description and use the name as the executable name.
func (gopt *GetOpt) Self(name string, description string) *GetOpt {
	if name != "" {
		gopt.name = name
	}
	gopt.description = description
	gopt.selfCalled = true
	return gopt
}

func (gopt *GetOpt) extraDetails() string {
	scriptName := filepath.Base(os.Args[0])
	description := ""
	// TODO: Expose string as var?
	if gopt.isCommand {
		description = fmt.Sprintf("Use '%s %s help <command>' for extra details.", scriptName, gopt.name)
	} else {
		description = fmt.Sprintf("Use '%s help <command>' for extra details.", scriptName)
	}
	return description
}

// Dispatch - Call CommandFn for the program commands based on the contents of the args slice.
// By default, if given the helpCommandName (normally just "help") as the first argument, it will print the help for the parent.
// If given helpCommandName plus the name of the command, it will print the help for the command.
func (gopt *GetOpt) Dispatch(ctx context.Context, helpCommandName string, args []string) error {
	Debug.Printf("Dispatch %v\n", args)
	if len(args) == 0 {
		fmt.Fprint(gopt.Writer, gopt.Help())
		fmt.Fprint(gopt.Writer, gopt.extraDetails()+"\n")
		exitFn(1)
		return nil
	}
	switch args[0] {
	case helpCommandName:
		if len(args) > 1 {
			commandName := args[1]
			for name, v := range gopt.commands {
				if commandName == name {
					fmt.Fprint(gopt.Writer, v.Help())
					exitFn(1)
					return nil
				}
			}
			// TODO: Expose string as var?
			return fmt.Errorf("unkown help entry '%s'", commandName)
		}
		fmt.Fprint(gopt.Writer, gopt.Help())
		fmt.Fprint(gopt.Writer, gopt.extraDetails()+"\n")
		exitFn(1)
		return nil
	default:
		commandName := args[0]
		for name, v := range gopt.commands {
			if commandName == name {
				if v.CommandFn != nil {
					remaining, err := v.Parse(args[1:])
					if len(v.commands) == 0 {
						if v.Called(helpCommandName) {
							fmt.Fprint(gopt.Writer, v.Help())
							return ErrorHelpCalled
						}
					}
					if err != nil {
						return err
					}
					err = v.CommandFn(ctx, v, remaining)
					if err != nil {
						return err
					}
				}
				return nil
			}
		}
		if strings.HasPrefix(args[0], "-") {
			// TODO: Expose string as var?
			return fmt.Errorf(`not a command or a valid option: '%s'
       Did you mean to pass it after the command?`, args[0])
		}
		// TODO: Expose string as var?
		return fmt.Errorf("not a command: '%s'", args[0])
	}
}

// TODO: Consider extracting, gopt.obj can be passed as an arg.

// failIfDefined will *panic* if an option is defined twice.
// This is not an error because the programmer has to fix this!
func (gopt *GetOpt) failIfDefined(aliases []string) {
	for _, a := range aliases {
		for _, option := range gopt.obj {
			for _, v := range option.Aliases {
				if v == a {
					panic(fmt.Sprintf("Option/Alias '%s' is already defined in option '%s'", a, option.Name))
				}
			}
		}
		if gopt.parent != nil {
			for _, option := range gopt.parent.obj {
				for _, v := range option.Aliases {
					if v == a {
						panic(fmt.Sprintf("Option/Alias '%s' is already defined", a))
					}
				}
			}
		}
	}
}

// Called - Indicates if the option was passed on the command line.
// If the `name` is an option that wasn't declared it will return false.
func (gopt *GetOpt) Called(name string) bool {
	if v, ok := gopt.obj[name]; ok {
		return v.Called
	}
	return false
}

// CalledAs - Returns the alias used to call the option.
// Empty string otherwise.
//
// If the `name` is an option that wasn't declared it will return an empty string.
//
// For options that can be called multiple times, the last alias used is returned.
func (gopt *GetOpt) CalledAs(name string) string {
	if v, ok := gopt.obj[name]; ok {
		return v.UsedAlias
	}
	return ""
}

// Value - Returns the value of the given option.
//
// Type assertions are required in cases where the compiler can't determine the type by context.
// For example: `opt.Value("flag").(bool)`.
func (gopt *GetOpt) Value(name string) interface{} {
	opt := gopt.Option(name)
	if opt != nil {
		return opt.Value()
	}
	return nil
}

// Option - Returns the *option.Option for name.
func (gopt *GetOpt) Option(name string) *option.Option {
	if value, ok := gopt.obj[name]; ok {
		return value
	}
	return nil
}

// setOption - Internal only
func (gopt *GetOpt) setOption(opts ...*option.Option) *GetOpt {
	node := gopt.completion.GetChildByName("options")
	nodeWithArg := gopt.completion.GetChildByName("options-with-arg")
	for _, opt := range opts {
		gopt.obj[opt.Name] = opt
		if opt.OptType == option.BoolType {
			// TODO: Add aliases
			node.Entries = append(node.Entries, opt.Name)
		} else {
			nodeWithArg.Entries = append(nodeWithArg.Entries, opt.Name)
		}
	}
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
	gopt.mode = mode
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
	gopt.unknownMode = mode
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
func (gopt *GetOpt) SetRequireOrder() *GetOpt {
	gopt.requireOrder = true
	return gopt
}

// SetMapKeysToLower - StringMap keys captured from StringMap are lower case.
// For example:
//
//     command --opt key=value
//
// And:
//
//     command --opt KEY=value
//
// Would both return `map[string]string{"key":"value"}`.
func (gopt *GetOpt) SetMapKeysToLower() *GetOpt {
	gopt.mapKeysToLower = true
	return gopt
}

// Alias - Adds aliases to an option.
func (gopt *GetOpt) Alias(alias ...string) ModifyFn {
	gopt.failIfDefined(alias)
	return func(opt *option.Option) {
		opt.SetAlias(alias...)
	}
}

// Required - Automatically return an error if the option is not called.
// Optionally provide an error message if the option is not called.
// A default error message will be used otherwise.
func (gopt *GetOpt) Required(msg ...string) ModifyFn {
	var errTxt string
	if len(msg) >= 1 {
		errTxt = msg[0]
	}
	return func(opt *option.Option) {
		opt.SetRequired(errTxt)
	}
}

// GetEnv - Will read an environment variable if set.
// Precedence higher to lower: CLI option, environment variable, option default.
//
// Currently, only `opt.Bool`, `opt.BoolVar`, `opt.String`, and `opt.StringVar` are supported.
//
// When an environment variable that matches the variable from opt.GetEnv is
// set, opt.GetEnv will set opt.Called(name) to true and will set
// opt.CalledAs(name) to the name of the environment variable used.
// In other words, when an option is required (opt.Required is set) opt.GetEnv
// satisfies that requirement.
//
// When using `opt.GetEnv` with `opt.Bool` or `opt.BoolVar`, only the words
// "true" or "false" are valid.  They can be provided in any casing, for
// example: "true", "True" or "TRUE".
//
// NOTE: Non supported option types behave with a No-Op when `opt.GetEnv` is defined.
func (gopt *GetOpt) GetEnv(name string) ModifyFn {
	return func(opt *option.Option) {
		opt.SetEnvVar(name)
		value := os.Getenv(name)
		if value != "" {
			switch opt.OptType {
			case option.BoolType:
				v := strings.ToLower(value)
				if v == "true" || v == "false" {
					opt.Save(v)
					opt.SetCalled(name)
				}
			case option.StringType, option.IntType, option.Float64Type:
				opt.Save(value)
				opt.SetCalled(name)
			}
		}
	}
}

// Description - Add a description to an option for use in automated help.
func (gopt *GetOpt) Description(msg string) ModifyFn {
	return func(opt *option.Option) {
		opt.Description = msg
	}
}

// ArgName - Add an argument name to an option for use in automated help.
// For example, by default a string option will have a default synopsis as follows:
//
//     --host <string>
//
// If ArgName("hostname") is used, the synopsis will read:
//
//     --host <hostname>
func (gopt *GetOpt) ArgName(name string) ModifyFn {
	return func(opt *option.Option) {
		opt.SetHelpArgName(name)
	}
}

// HelpSynopsisArgs - Defines the help synopsis args description.
// Defaults to: [<args>]
func (gopt *GetOpt) HelpSynopsisArgs(args string) *GetOpt {
	gopt.synopsisArgs = args
	return gopt
}

func getCommandName(opt *GetOpt) string {
	if opt.isCommand {
		name := getCommandName(opt.parent)
		return fmt.Sprintf("%s %s", name, opt.name)
	}
	return opt.name
}

// Help - Default help string that is composed of the HelpSynopsis and HelpOptionList.
func (gopt *GetOpt) Help(sections ...HelpSection) string {
	if len(sections) == 0 {
		// Print all in the following order
		sections = []HelpSection{helpDefaultName, HelpSynopsis, HelpCommandList, HelpOptionList}
	}
	helpTxt := ""
	var scriptName string
	if gopt.isCommand {
		scriptName = getCommandName(gopt.parent)
	}
	for _, section := range sections {
		switch section {
		// Default name only prints name if the name or description is set.
		// The explicit type always prints it.
		case helpDefaultName:
			if gopt.selfCalled || gopt.isCommand {
				helpTxt += help.Name(scriptName, gopt.name, gopt.description)
				helpTxt += "\n"
			}
		case HelpName:
			helpTxt += help.Name(scriptName, gopt.name, gopt.description)
			helpTxt += "\n"
		case HelpSynopsis:
			options := []*option.Option{}
			commands := []string{}
			for _, option := range gopt.obj {
				options = append(options, option)
			}
			for _, command := range gopt.commands {
				commands = append(commands, command.name)
			}
			helpTxt += help.Synopsis(scriptName, gopt.name, gopt.synopsisArgs, options, commands)
			helpTxt += "\n"
		case HelpCommandList:
			m := make(map[string]string)
			for _, command := range gopt.commands {
				m[command.name] = command.description
			}
			commands := help.CommandList(m)
			if commands != "" {
				helpTxt += commands
				helpTxt += "\n"
			}
		case HelpOptionList:
			options := []*option.Option{}
			for _, option := range gopt.obj {
				options = append(options, option)
			}
			helpTxt += help.OptionList(options)
		}
	}
	return helpTxt
}

// HelpCommand - Adds a help command with completion for all other commands.
//
// NOTE: Define after all other commands have been defined.
func (gopt *GetOpt) HelpCommand(description string) *GetOpt {
	if description == "" {
		description = gopt.extraDetails()
	}
	// TODO: "help" is hardcoded
	opt := gopt.NewCommand("help", description)
	commands := []string{}
	for name := range gopt.commands {
		commands = append(commands, name)
	}
	opt.CustomCompletion(commands)
	return opt
}

// CustomCompletion - Add a custom completion list.
func (gopt *GetOpt) CustomCompletion(list []string) *GetOpt {
	gopt.completion.AddChild(completion.NewNode("custom", completion.CustomNode, list))
	return gopt
}

// BoolVar - define a `bool` option and its aliases.
// The result will be available through the variable marked by the given pointer.
// If the option is found, the result will be the opposite of the provided default.
func (gopt *GetOpt) BoolVar(p *bool, name string, def bool, fns ...ModifyFn) {
	gopt.failIfDefined([]string{name})
	*p = def
	opt := option.New(name, option.BoolType, p)
	opt.DefaultStr = fmt.Sprintf("%t", def)
	opt.Handler = gopt.handleBool
	for _, fn := range fns {
		fn(opt)
	}
	gopt.completionAppendAliases(opt.Aliases)
	gopt.setOption(opt)
}

// Bool - define a `bool` option and its aliases.
// It returns a `*bool` pointing to the variable holding the result.
// If the option is found, the result will be the opposite of the provided default.
func (gopt *GetOpt) Bool(name string, def bool, fns ...ModifyFn) *bool {
	gopt.BoolVar(&def, name, def, fns...)
	return &def
}

func (gopt *GetOpt) handleBool(name string, argument string, usedAlias string) error {
	Debug.Println("handleBool")
	opt := gopt.Option(name)
	opt.SetCalled(usedAlias)
	opt.SetBoolAsOppositeToDefault()
	return nil
}

func (gopt *GetOpt) handleSingleOption(name string, argument string, usedAlias string) error {
	Debug.Printf("handleSingleOption %s, %s\n", name, argument)
	opt := gopt.Option(name)
	opt.SetCalled(usedAlias)
	if argument != "" {
		return opt.Save(argument)
	}
	if !gopt.args.existsNext() {
		Debug.Printf("handleSingleOption %v %v\n", gopt.args.remaining(), gopt.args.existsNext())
		if opt.IsOptional {
			return nil
		}
		return fmt.Errorf(text.ErrorMissingArgument, usedAlias)
	}
	// Check if next arg is option
	if optPair, _ := isOption(gopt.args.peekNextValue(), gopt.mode, false); len(optPair) > 0 {
		if opt.IsOptional {
			return nil
		}
		return fmt.Errorf(text.ErrorArgumentWithDash, usedAlias)
	}
	gopt.args.next()
	return opt.Save(gopt.args.value())
}

// StringVar - define a `string` option and its aliases.
// The result will be available through the variable marked by the given pointer.
// If not called, the return value will be that of the given default `def`.
func (gopt *GetOpt) StringVar(p *string, name, def string, fns ...ModifyFn) {
	gopt.failIfDefined([]string{name})
	opt := option.New(name, option.StringType, p)
	opt.SetString(def)
	opt.DefaultStr = fmt.Sprintf(`"%s"`, def)
	opt.Handler = gopt.handleSingleOption
	opt.SetHelpArgName("string")

	for _, fn := range fns {
		fn(opt)
	}
	gopt.completionWithArgAppendAliases(opt.Aliases)
	gopt.setOption(opt)
}

// String - define a `string` option and its aliases.
// If not called, the return value will be that of the given default `def`.
func (gopt *GetOpt) String(name, def string, fns ...ModifyFn) *string {
	gopt.StringVar(&def, name, def, fns...)
	return &def
}

// StringVarOptional - define a `string` option and its aliases.
// The result will be available through the variable marked by the given pointer.
//
// StringVarOptional will set the string to the provided default value when no value is given.
// For example, when called with `--strOpt value`, the value is `value`.
// when called with `--strOpt` the value is the given default.
func (gopt *GetOpt) StringVarOptional(p *string, name, def string, fns ...ModifyFn) {
	gopt.failIfDefined([]string{name})
	opt := option.New(name, option.StringType, p)
	opt.SetString(def)
	opt.DefaultStr = fmt.Sprintf(`"%s"`, def)
	opt.Handler = gopt.handleSingleOption
	opt.SetHelpArgName("string")

	// TODO: The only  difference with StringVar is this line
	opt.IsOptional = true

	for _, fn := range fns {
		fn(opt)
	}
	gopt.completionAppendAliases(opt.Aliases)
	gopt.setOption(opt)
}

// StringOptional - define a `string` option and its aliases.
//
// StringOptional will set the string to the provided default value when no value is given.
// For example, when called with `--strOpt value`, the value is `value`.
// when called with `--strOpt` the value is the given default.
func (gopt *GetOpt) StringOptional(name string, def string, fns ...ModifyFn) *string {
	gopt.StringVarOptional(&def, name, def, fns...)
	return &def
}

// IntVar - define an `int` option and its aliases.
// The result will be available through the variable marked by the given pointer.
func (gopt *GetOpt) IntVar(p *int, name string, def int, fns ...ModifyFn) {
	gopt.failIfDefined([]string{name})
	opt := option.New(name, option.IntType, p)
	opt.SetInt(def)
	opt.DefaultStr = fmt.Sprintf("%d", def)
	opt.Handler = gopt.handleSingleOption
	opt.SetHelpArgName("int")

	for _, fn := range fns {
		fn(opt)
	}
	gopt.completionWithArgAppendAliases(opt.Aliases)
	gopt.setOption(opt)
}

// Int - define an `int` option and its aliases.
func (gopt *GetOpt) Int(name string, def int, fns ...ModifyFn) *int {
	gopt.IntVar(&def, name, def, fns...)
	return &def
}

// IntVarOptional - define a `int` option and its aliases.
// The result will be available through the variable marked by the given pointer.
//
// IntOptional will set the int to the provided default value when no value is given.
// For example, when called with `--intOpt 123`, the value is `123`.
// when called with `--intOpt` the value is the given default.
func (gopt *GetOpt) IntVarOptional(p *int, name string, def int, fns ...ModifyFn) {
	gopt.failIfDefined([]string{name})
	opt := option.New(name, option.IntType, p)
	opt.SetInt(def)
	opt.DefaultStr = fmt.Sprintf("%d", def)
	opt.Handler = gopt.handleSingleOption
	opt.SetHelpArgName("int")

	// TODO: The only  difference with IntVar is this line
	opt.IsOptional = true

	for _, fn := range fns {
		fn(opt)
	}
	gopt.completionAppendAliases(opt.Aliases)
	gopt.setOption(opt)
}

// IntOptional - define a `int` option and its aliases.
//
// IntOptional will set the int to the provided default value when no value is given.
// For example, when called with `--intOpt 123`, the value is `123`.
// when called with `--intOpt` the value is the given default.
func (gopt *GetOpt) IntOptional(name string, def int, fns ...ModifyFn) *int {
	gopt.IntVarOptional(&def, name, def, fns...)
	return &def
}

// Float64Var - define an `float64` option and its aliases.
// The result will be available through the variable marked by the given pointer.
func (gopt *GetOpt) Float64Var(p *float64, name string, def float64, fns ...ModifyFn) {
	gopt.failIfDefined([]string{name})
	opt := option.New(name, option.Float64Type, p)
	opt.SetFloat64(def)
	opt.DefaultStr = fmt.Sprintf("%f", def)
	opt.Handler = gopt.handleSingleOption
	opt.SetHelpArgName("float64")

	for _, fn := range fns {
		fn(opt)
	}
	gopt.completionWithArgAppendAliases(opt.Aliases)
	gopt.setOption(opt)
}

// Float64 - define an `float64` option and its aliases.
func (gopt *GetOpt) Float64(name string, def float64, fns ...ModifyFn) *float64 {
	gopt.Float64Var(&def, name, def, fns...)
	return &def
}

// Float64VarOptional - define an `float64` option and its aliases.
// The result will be available through the variable marked by the given pointer.
func (gopt *GetOpt) Float64VarOptional(p *float64, name string, def float64, fns ...ModifyFn) {
	gopt.failIfDefined([]string{name})
	opt := option.New(name, option.Float64Type, p)
	opt.SetFloat64(def)
	opt.DefaultStr = fmt.Sprintf("%f", def)
	opt.Handler = gopt.handleSingleOption
	opt.SetHelpArgName("float64")

	// TODO: The only  difference with Float64Var is this line
	opt.IsOptional = true

	for _, fn := range fns {
		fn(opt)
	}
	gopt.completionAppendAliases(opt.Aliases)
	gopt.setOption(opt)
}

// Float64Optional - define an `float64` option and its aliases.
func (gopt *GetOpt) Float64Optional(name string, def float64, fns ...ModifyFn) *float64 {
	gopt.Float64VarOptional(&def, name, def, fns...)
	return &def
}

// StringSliceVar - define a `[]string` option and its aliases.
//
// StringSliceVar will accept multiple calls to the same option and append them
// to the `[]string`.
// For example, when called with `--strRpt 1 --strRpt 2`, the value is `[]string{"1", "2"}`.
//
// Additionally, StringSliceVar will allow to define a min and max amount of
// arguments to be passed at once.
// For example, when min is 1 and max is 3 and called with `--strRpt 1 2 3`,
// the value is `[]string{"1", "2", "3"}`.
// It could also be called with `--strRpt 1 --strRpt 2 --strRpt 3` for the same result.
//
// When min is bigger than 1, it is required to pass the amount of arguments defined by min at once.
// For example: with `min = 2`, you at least require `--strRpt 1 2 --strRpt 3`
func (gopt *GetOpt) StringSliceVar(p *[]string, name string, min, max int, fns ...ModifyFn) {
	gopt.failIfDefined([]string{name})
	opt := option.New(name, option.StringRepeatType, p)
	opt.DefaultStr = "[]"
	opt.Handler = gopt.handleSliceMultiOption
	opt.MinArgs = min
	opt.MaxArgs = max
	opt.SetHelpArgName("string")
	if min <= 0 {
		panic(fmt.Sprintf("%s min should be > 0", name))
	}
	if max <= 0 || max < min {
		panic(fmt.Sprintf("%s max should be > 0 and > min", name))
	}
	for _, fn := range fns {
		fn(opt)
	}
	Debug.Printf("StringMulti return: %v\n", *p)
	gopt.completionWithArgAppendAliases(opt.Aliases)
	gopt.setOption(opt)
}

// StringSlice - define a `[]string` option and its aliases.
//
// StringSlice will accept multiple calls to the same option and append them
// to the `[]string`.
// For example, when called with `--strRpt 1 --strRpt 2`, the value is `[]string{"1", "2"}`.
//
// Additionally, StringSlice will allow to define a min and max amount of
// arguments to be passed at once.
// For example, when min is 1 and max is 3 and called with `--strRpt 1 2 3`,
// the value is `[]string{"1", "2", "3"}`.
// It could also be called with `--strRpt 1 --strRpt 2 --strRpt 3` for the same result.
//
// When min is bigger than 1, it is required to pass the amount of arguments defined by min at once.
// For example: with `min = 2`, you at least require `--strRpt 1 2 --strRpt 3`
func (gopt *GetOpt) StringSlice(name string, min, max int, fns ...ModifyFn) *[]string {
	s := []string{}
	gopt.StringSliceVar(&s, name, min, max, fns...)
	return &s
}

// IntSliceVar - define a `[]int` option and its aliases.
//
// IntSliceVar will accept multiple calls to the same option and append them
// to the `[]int`.
// For example, when called with `--intRpt 1 --intRpt 2`, the value is `[]int{1, 2}`.
//
// Additionally, IntSliceVar will allow to define a min and max amount of
// arguments to be passed at once.
// For example, when min is 1 and max is 3 and called with `--strRpt 1 2 3`,
// the value is `[]int{1, 2, 3}`.
// It could also be called with `--strRpt 1 --strRpt 2 --strRpt 3` for the same result.
//
// When min is bigger than 1, it is required to pass the amount of arguments defined by min at once.
// For example: with `min = 2`, you at least require `--strRpt 1 2 --strRpt 3`
//
// Finally, positive integer ranges are allowed.
// For example, Instead of writing: `csv --columns 1 2 3` or
// `csv --columns 1 --columns 2 --columns 3`
// The input could be: `csv --columns 1..3`.
func (gopt *GetOpt) IntSliceVar(p *[]int, name string, min, max int, fns ...ModifyFn) {
	gopt.failIfDefined([]string{name})
	opt := option.New(name, option.IntRepeatType, p)
	opt.DefaultStr = "[]"
	opt.Handler = gopt.handleSliceMultiOption
	opt.MinArgs = min
	opt.MaxArgs = max
	opt.SetHelpArgName("int")
	if min <= 0 {
		panic(fmt.Sprintf("%s min should be > 0", name))
	}
	if max <= 0 || max < min {
		panic(fmt.Sprintf("%s max should be > 0 and > min", name))
	}
	for _, fn := range fns {
		fn(opt)
	}
	Debug.Printf("IntMulti return: %v\n", *p)
	gopt.completionWithArgAppendAliases(opt.Aliases)
	gopt.setOption(opt)
}

// IntSlice - define a `[]int` option and its aliases.
//
// IntSlice will accept multiple calls to the same option and append them
// to the `[]int`.
// For example, when called with `--intRpt 1 --intRpt 2`, the value is `[]int{1, 2}`.
//
// Additionally, IntSlice will allow to define a min and max amount of
// arguments to be passed at once.
// For example, when min is 1 and max is 3 and called with `--strRpt 1 2 3`,
// the value is `[]int{1, 2, 3}`.
// It could also be called with `--strRpt 1 --strRpt 2 --strRpt 3` for the same result.
//
// When min is bigger than 1, it is required to pass the amount of arguments defined by min at once.
// For example: with `min = 2`, you at least require `--strRpt 1 2 --strRpt 3`
//
// Finally, positive integer ranges are allowed.
// For example, Instead of writing: `csv --columns 1 2 3` or
// `csv --columns 1 --columns 2 --columns 3`
// The input could be: `csv --columns 1..3`.
func (gopt *GetOpt) IntSlice(name string, min, max int, fns ...ModifyFn) *[]int {
	s := []int{}
	gopt.IntSliceVar(&s, name, min, max, fns...)
	return &s
}

// StringMapVar - define a `map[string]string` option and its aliases.
//
// StringMapVar will accept multiple calls of `key=value` type to the same option
// and add them to the `map[string]string` result.
// For example, when called with `--strMap k=v --strMap k2=v2`, the value is
// `map[string]string{"k":"v", "k2": "v2"}`.
//
// Additionally, StringMapVar will allow to define a min and max amount of
// arguments to be passed at once.
// For example, when min is 1 and max is 3 and called with `--strMap k=v k2=v2 k3=v3`,
// the value is `map[string]string{"k":"v", "k2": "v2", "k3": "v3"}`.
// It could also be called with `--strMap k=v --strMap k2=v2 --strMap k3=v3` for the same result.
//
// When min is bigger than 1, it is required to pass the amount of arguments defined by min at once.
// For example: with `min = 2`, you at least require `--strMap k=v k2=v2 --strMap k3=v3`
func (gopt *GetOpt) StringMapVar(m *map[string]string, name string, min, max int, fns ...ModifyFn) {
	// TODO: panic if m is nil

	// check that the map has been initialized
	if *m == nil {
		*m = make(map[string]string)
	}
	gopt.failIfDefined([]string{name})
	opt := option.New(name, option.StringMapType, m)
	opt.DefaultStr = "{}"
	opt.Handler = gopt.handleSliceMultiOption
	opt.MinArgs = min
	opt.MaxArgs = max
	opt.SetHelpArgName("key=value")
	if min <= 0 {
		panic(fmt.Sprintf("%s min should be > 0", name))
	}
	if max <= 0 || max < min {
		panic(fmt.Sprintf("%s max should be > 0 and > min", name))
	}
	for _, fn := range fns {
		fn(opt)
	}
	Debug.Printf("StringMulti return: %v\n", *m)
	gopt.completionWithArgAppendAliases(opt.Aliases)
	gopt.setOption(opt)
}

// StringMap - define a `map[string]string` option and its aliases.
//
// StringMap will accept multiple calls of `key=value` type to the same option
// and add them to the `map[string]string` result.
// For example, when called with `--strMap k=v --strMap k2=v2`, the value is
// `map[string]string{"k":"v", "k2": "v2"}`.
//
// Additionally, StringMap will allow to define a min and max amount of
// arguments to be passed at once.
// For example, when min is 1 and max is 3 and called with `--strMap k=v k2=v2 k3=v3`,
// the value is `map[string]string{"k":"v", "k2": "v2", "k3": "v3"}`.
// It could also be called with `--strMap k=v --strMap k2=v2 --strMap k3=v3` for the same result.
//
// When min is bigger than 1, it is required to pass the amount of arguments defined by min at once.
// For example: with `min = 2`, you at least require `--strMap k=v k2=v2 --strMap k3=v3`
func (gopt *GetOpt) StringMap(name string, min, max int, fns ...ModifyFn) map[string]string {
	m := map[string]string{}
	gopt.StringMapVar(&m, name, min, max, fns...)
	return m
}

// NOTE: Options that can be called multiple times and thus modify the used
// alias, don't use usedAlias for their errors because the error is used to
// check the min, max args.
// TODO: Do a regex instead of matching the full error to enable usedAlias in errors.
func (gopt *GetOpt) handleSliceMultiOption(name string, argument string, usedAlias string) error {
	Debug.Printf("handleStringSlice\n")
	opt := gopt.Option(name)
	opt.SetCalled(usedAlias)
	opt.MapKeysToLower = gopt.mapKeysToLower
	argCounter := 0

	if argument != "" {
		argCounter++
		err := opt.Save(argument)
		if err != nil {
			return err
		}
	}
	// Function to handle one arg at a time
	next := func(required bool) error {
		Debug.Printf("total arguments: %d, index: %d, counter %d", gopt.args.size(), gopt.args.index(), argCounter)
		if !gopt.args.existsNext() {
			if required {
				return fmt.Errorf(text.ErrorMissingArgument, name)
			}
			return fmt.Errorf("NoMoreArguments")
		}
		// Check if next arg is option
		if optPair, _ := isOption(gopt.args.peekNextValue(), gopt.mode, false); len(optPair) > 0 {
			Debug.Printf("Next arg is option: %s\n", gopt.args.peekNextValue())
			return fmt.Errorf(text.ErrorArgumentWithDash, name)
		}
		// Check if next arg is not key=value
		if opt.OptType == option.StringMapType && !strings.Contains(gopt.args.peekNextValue(), "=") {
			if required {
				return fmt.Errorf(text.ErrorArgumentIsNotKeyValue, name)
			}
			return nil
		}
		if opt.OptType == option.IntRepeatType {
			_, err := strconv.Atoi(gopt.args.peekNextValue())
			if !required && err != nil {
				return nil
			}
		}
		gopt.args.next()
		return opt.Save(gopt.args.value())
	}

	// Go through the required and optional iterations
	for argCounter < opt.MaxArgs {
		argCounter++
		err := next(argCounter <= opt.MinArgs)
		Debug.Printf("counter: %d, value: %v, err %v", argCounter, opt.Value(), err)
		if err != nil {
			if err.Error() == "NoMoreArguments" {
				Debug.Printf("return value: %v", opt.Value())
				return nil
			}
			// always fail if errors under min args
			// After min args, skip missing arg errors
			if argCounter <= opt.MinArgs ||
				(err.Error() != fmt.Sprintf(text.ErrorMissingArgument, name) &&
					err.Error() != fmt.Sprintf(text.ErrorArgumentWithDash, name)) {
				Debug.Printf("return value: %v, err: %v", opt.Value(), err)
				return err
			}
			Debug.Printf("return value: %v", opt.Value())
			return nil
		}
	}
	Debug.Printf("return value: %v", opt.Value())
	return nil
}

// IncrementVar - When called multiple times it increments the provided int.
func (gopt *GetOpt) IncrementVar(p *int, name string, def int, fns ...ModifyFn) {
	gopt.failIfDefined([]string{name})
	opt := option.New(name, option.IntType, p)
	opt.SetInt(def)
	opt.DefaultStr = fmt.Sprintf("%d", def)
	opt.Handler = gopt.handleIncrement
	for _, fn := range fns {
		fn(opt)
	}
	gopt.completionAppendAliases(opt.Aliases)
	gopt.setOption(opt)
}

// Increment - When called multiple times it increments the int counter defined by this option.
func (gopt *GetOpt) Increment(name string, def int, fns ...ModifyFn) *int {
	gopt.IncrementVar(&def, name, def, fns...)
	return &def
}

func (gopt *GetOpt) handleIncrement(name string, argument string, usedAlias string) error {
	Debug.Println("handleIncrement")
	opt := gopt.Option(name)
	opt.SetCalled(usedAlias)
	opt.SetInt(opt.Int() + 1)
	return nil
}

// func (opt *GetOpt) StringMulti(name string, def []string, min int, max int, fns ...ModifyFn) {}
// func (opt *GetOpt) StringMap(name string, def map[string]string, min int, max int, fns ...ModifyFn) {}
// func (opt *GetOpt) Procedure(name string, lambda_func int, fns ...ModifyFn) {}

// Stringer - print a nice looking representation of the resulting `Option` map.
func (gopt *GetOpt) Stringer() string {
	s := "{\n"
	for name, opt := range gopt.obj {
		s += fmt.Sprintf("\"%s\":", name)
		switch v := opt.Value().(type) {
		case bool, int, float64:
			s += fmt.Sprintf("%v,\n", v)
		default:
			s += fmt.Sprintf("\"%v\",\n", v)
		}
	}
	s += "}"
	Debug.Printf("stringer: %s\n", s)
	return s
}

// TODO: Add case insensitive matching.
func (gopt *GetOpt) getOptionFromAliases(alias string) (optName, usedAlias string, found bool, err error) {
	Debug.Printf("getOptionFromAliases: %s\n", gopt.name)

	// Attempt to fully match node option
	found = false
	for name, option := range gopt.obj {
		for _, v := range option.Aliases {
			Debug.Printf("Trying to match '%s' against '%s' alias for '%s'\n", alias, v, name)
			if v == alias {
				Debug.Printf("found: %s, %s\n", v, alias)
				found = true
				optName = name
				usedAlias = v
				break
			}
		}
	}

	// Attempt to fully match command option
	matches := []string{}
	for _, command := range gopt.commands {
		for name, option := range command.obj {
			for _, v := range option.Aliases {
				Debug.Printf("Trying to match '%s' against '%s' alias for command option '%s'\n", alias, v, name)
				if v == alias {
					Debug.Printf("found: %s, %s\n", v, alias)
					matches = append(matches, v)
					continue
				}
			}
		}
	}

	// If there are full matches of the command return with an empty match at the parent.
	// There is no case in which a match could be found at the parent because aliases are checked.
	if len(matches) >= 1 {
		Debug.Printf("getOptionFromAliases return: %s, %s, %v\n", optName, usedAlias, found)
		return optName, usedAlias, found, nil
	}

	// Attempt to match initial chars of node option
	if !found {
		matches := []string{}
		for name, option := range gopt.obj {
			for _, v := range option.Aliases {
				Debug.Printf("Trying to lazy match '%s' against '%s' alias for '%s'\n", alias, v, name)
				if strings.HasPrefix(v, alias) {
					Debug.Printf("found: %s, %s\n", v, alias)
					matches = append(matches, name)
					usedAlias = v
					continue
				}
			}
		}
		Debug.Printf("matches: %v(%d), %s\n", matches, len(matches), alias)

		// Attempt to match initial chars of command option
		commandMatches := []string{}
		for _, command := range gopt.commands {
			for name, option := range command.obj {
				for _, v := range option.Aliases {
					Debug.Printf("Trying to lazy match '%s' against '%s' alias for command option '%s'\n", alias, v, name)
					if strings.HasPrefix(v, alias) {
						Debug.Printf("found: %s, %s\n", v, alias)
						commandMatches = append(commandMatches, v)
						continue
					}
				}
			}
		}
		Debug.Printf("commandMatches: %v(%d), %s\n", commandMatches, len(commandMatches), alias)

		dedup := func(s []string) []string {
			m := map[string]struct{}{}
			for _, e := range s {
				m[e] = struct{}{}
			}
			r := []string{}
			for k := range m {
				r = append(r, k)
			}
			return r
		}
		matches = dedup(matches)
		commandMatches = dedup(commandMatches)
		combined := dedup(append(matches, commandMatches...))

		if len(combined) >= 2 {
			sort.Strings(combined)
			return optName, usedAlias, found, fmt.Errorf(text.ErrorAmbiguousArgument, alias, combined)
		}
		if len(matches) == 1 {
			found = true
			optName = matches[0]
		}
	}
	Debug.Printf("getOptionFromAliases return: %s, %s, %v\n", optName, usedAlias, found)
	return optName, usedAlias, found, nil
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
	gopt.passOptionsToChildren()
	return gopt.parse(args)
}

func (gopt *GetOpt) passOptionsToChildren() error {
	Debug.Printf("passOptionsToChildren %s\n", gopt.name)
	for _, commandOpt := range gopt.commands {
		// pass writer to child
		commandOpt.Writer = gopt.Writer

		// pass options to child
		for optName, opt := range gopt.obj {
			commandOpt.obj[optName] = opt

			parentNode := gopt.completion.GetChildByName("options")
			node := commandOpt.completion.GetChildByName("options")
			node.Entries = append(node.Entries, parentNode.Entries...)

			parentNodeWithArg := gopt.completion.GetChildByName("options-with-arg")
			nodeWithArg := commandOpt.completion.GetChildByName("options-with-arg")
			nodeWithArg.Entries = append(nodeWithArg.Entries, parentNodeWithArg.Entries...)
		}
		// Once we are done passing the options to the command, pass them along to its children.
		commandOpt.passOptionsToChildren()
	}
	return nil
}

func (gopt *GetOpt) passArgsToParent() {
	Debug.Printf("passArgsToParent %s\n", gopt.name)
	if parent := gopt.parent; parent != nil {
		parent.args = gopt.args
		parent.passArgsToParent()
	}
}

func (gopt *GetOpt) parse(args []string) ([]string, error) {
	compLine := os.Getenv("COMP_LINE")
	// https://stackoverflow.com/a/33396628
	if compLine != "" {
		fmt.Fprintln(completionWriter, strings.Join(gopt.completion.CompLineComplete(false, compLine), "\n"))
		exitFn(124) // programmable completion restarts from the beginning, with an attempt to find a new compspec for that command.
	}
	al := newArgList(args)
	gopt.args = al
	Debug.Printf("parse %s\n", gopt.name)
	Debug.Printf("Parse args: %v(%d)\n", args, len(args))
	var remaining []string
	// opt.argsIndex is the index in the opt.args slice.
	// Option handlers will have to know about it, to ask for the next element.
	for gopt.args.next() {
		arg := gopt.args.value()
		Debug.Printf("Parse input arg: %s\n", arg)
		if optPair, _ := isOption(arg, gopt.mode, false); len(optPair) > 0 {
			Debug.Printf("Parse optPair: %v\n", optPair)
			// Check for termination: '--'
			if optPair[0].Option == "--" {
				Debug.Printf("Parse -- found\n")
				// move index to next position (to not include '--') and return remaining.
				gopt.args.next()
				remaining = append(remaining, gopt.args.remaining()...)
				Debug.Printf("return %v, %v", remaining, nil)
				return remaining, nil
			}
			Debug.Printf("Parse continue\n")
			for _, optElement := range optPair {
				Debug.Printf("Parse optElement: %s\n", optElement)
				optName, usedAlias, ok, err := gopt.getOptionFromAliases(optElement.Option)
				if err != nil {
					return nil, err
				}
				if ok {
					Debug.Printf("Parse found optPair %s\n", optName)
					gopt.passArgsToParent()
					opt := gopt.Option(optName)
					handler := opt.Handler
					Debug.Printf("handler found: name %s, arguments %v, index %d, list %s, args %v\n", optName, optElement.Args, gopt.args.index(), optElement.Option, gopt.args.remaining())
					// TODO: Currently we handle at most 1 arg, but if we were to split on comma there could be multiple.
					var argument string
					if len(optElement.Args) > 0 {
						argument = optElement.Args[0]
					}
					err := handler(optName, argument, usedAlias)
					if err != nil {
						Debug.Printf("handler return: value %v, return %v, %v", opt.Value(), nil, err)
						return nil, err
					}
				} else {
					Debug.Printf("optPair not found for '%s'\n", optElement.Option)
					switch gopt.unknownMode {
					case Pass:
						if gopt.requireOrder {
							remaining = append(remaining, gopt.args.remaining()...)
							Debug.Printf("Stop on unknown options %s\n", arg)
							Debug.Printf("return %v, %v", remaining, nil)
							return remaining, nil
						}
						remaining = append(remaining, arg)
					case Warn:
						// TODO: This WARNING can't be changed into another language. Hardcoded.
						fmt.Fprintf(gopt.Writer, "WARNING: "+text.MessageOnUnknown+"\n", optElement.Option)
						remaining = append(remaining, arg)
					default:
						err := fmt.Errorf(text.MessageOnUnknown, optElement.Option)
						Debug.Printf("return %v, %v", nil, err)
						return nil, err
					}
				}
			}
		} else {
			if gopt.requireOrder {
				remaining = append(remaining, gopt.args.remaining()...)
				Debug.Printf("Stop on non option: %s\n", arg)
				Debug.Printf("return %v, %v", remaining, nil)
				return remaining, nil
			}
			remaining = append(remaining, arg)
		}
	}
	// After parsing all options, verify that all required options where called.
	for _, option := range gopt.obj {
		err := option.CheckRequired()
		if err != nil {
			Debug.Printf("return %v, %v", nil, err)
			return nil, err
		}
	}
	Debug.Printf("return %v, %v", remaining, nil)
	return remaining, nil
}

// InterruptContext - Creates a top level context that listens to os.Interrupt, syscall.SIGHUP and syscall.SIGTERM and calls the CancelFunc if the signals are triggered.
// When the listener finishes its work, it sends a message to the done channel.
//
// Use:
//     func main() { ...
//     ctx, cancel, done := getoptions.InterruptContext()
//     defer func() { cancel(); <-done }()
//
// NOTE: InterruptContext is a method to reuse gopt.Writer
func (gopt *GetOpt) InterruptContext() (ctx context.Context, cancel context.CancelFunc, done chan struct{}) {
	done = make(chan struct{}, 1)
	ctx, cancel = context.WithCancel(context.Background())
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGHUP, syscall.SIGTERM)
	go func() {
		defer func() {
			signal.Stop(signals)
			cancel()
			done <- struct{}{}
		}()
		select {
		case <-signals:
			fmt.Fprintf(gopt.Writer, "\n%s\n", text.MessageOnInterrupt)
		case <-ctx.Done():
		}
	}()
	return ctx, cancel, done
}
