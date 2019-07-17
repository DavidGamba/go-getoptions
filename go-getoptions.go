// This file is part of go-getoptions.
//
// Copyright (C) 2015-2019  David Gamba Rios
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package getoptions

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

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

// GetOpt - main object.
type GetOpt struct {
	// Help fields
	name        string
	description string

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

// New returns an empty object of type GetOpt.
// This is the starting point when using go-getoptions.
// For example:
//
//   opt := getoptions.New()
func New() *GetOpt {
	root := completion.NewNode("root", completion.Root, nil)
	root.AddChild(completion.NewNode("options", completion.OptionsNode, nil))
	gopt := &GetOpt{
		obj:        make(map[string]*option.Option),
		commands:   make(map[string]*GetOpt),
		Writer:     os.Stderr,
		completion: root,
	}
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

// Self - Set a custom name and description that will show in the automated help.
// If name is an empty string, it will only use the description and use the name as the executable name.
func (gopt *GetOpt) Self(name string, description string) *GetOpt {
	gopt.name = name
	gopt.description = description
	return gopt
}

// TODO: Consider extracting, gopt.obj can be passed as an arg.

// failIfDefined will *panic* if an option is defined twice.
// This is not an error because the programmer has to fix this!
func (gopt *GetOpt) failIfDefined(aliases []string) {
	for _, a := range aliases {
		for _, option := range gopt.obj {
			for _, v := range option.Aliases {
				if v == a {
					panic(fmt.Sprintf("Option/Alias '%s' is already defined", a))
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

// SetOption - Sets a given *option.Option
func (gopt *GetOpt) SetOption(opts ...*option.Option) *GetOpt {
	node := gopt.completion.GetChildByName("options")
	for _, opt := range opts {
		gopt.obj[opt.Name] = opt
		// TODO: Add aliases
		node.Entries = append(node.Entries, opt.Name)
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
func (gopt *GetOpt) SetMode(mode Mode) {
	gopt.mode = mode
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
func (gopt *GetOpt) SetUnknownMode(mode UnknownMode) {
	gopt.unknownMode = mode
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
func (gopt *GetOpt) SetRequireOrder() {
	gopt.requireOrder = true
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
func (gopt *GetOpt) SetMapKeysToLower() {
	gopt.mapKeysToLower = true
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
		opt.HelpArgName = name
	}
}

// Help - Default help string that is composed of the HelpSynopsis and HelpOptionList.
func (gopt *GetOpt) Help() string {
	help := ""
	if gopt.name != "" || gopt.description != "" {
		help += gopt.HelpName()
		help += "\n"
	}
	help += gopt.HelpSynopsis()
	help += "\n"
	commands := gopt.HelpCommandList()
	if commands != "" {
		help += commands
		help += "\n"
	}
	help += gopt.HelpOptionList()
	return help
}

func (gopt *GetOpt) HelpName() string {
	scriptName := filepath.Base(os.Args[0])
	return help.HelpName(scriptName, gopt.name, gopt.description)
}

// HelpSynopsis - Return a default synopsis.
func (gopt *GetOpt) HelpSynopsis() string {
	scriptName := filepath.Base(os.Args[0])
	options := []*option.Option{}
	commands := []string{}
	for _, option := range gopt.obj {
		options = append(options, option)
	}
	for _, command := range gopt.commands {
		commands = append(commands, command.name)
	}
	return help.HelpSynopsis(scriptName, gopt.name, options, commands)
}

// HelpCommandList - Return a default command list.
func (gopt *GetOpt) HelpCommandList() string {
	m := make(map[string]string)
	for _, command := range gopt.commands {
		m[command.name] = command.description
	}
	return help.HelpCommandList(m)
}

// HelpOptionList - Return a formatted list of options and their descriptions.
func (gopt *GetOpt) HelpOptionList() string {
	options := []*option.Option{}
	for _, option := range gopt.obj {
		options = append(options, option)
	}
	return help.HelpOptionList(options)
}

func (gopt *GetOpt) Command(options *GetOpt) {
	if options == nil {
		options = New()
	}
	// TODO: Add check to see if options.name is != "", panic otherwise.
	node := options.completion
	node.Kind = completion.StringNode
	node.Name = options.name
	gopt.completion.AddChild(node)
	gopt.commands[options.name] = options
}

// CustomCompletion - Add a custom completion list.
func (gopt *GetOpt) CustomCompletion(list []string) *GetOpt {
	gopt.completion.AddChild(completion.NewNode("custom", completion.CustomNode, list))
	return gopt
}

// Bool - define a `bool` option and its aliases.
// It returns a `*bool` pointing to the variable holding the result.
// If the option is found, the result will be the opposite of the provided default.
func (gopt *GetOpt) Bool(name string, def bool, fns ...ModifyFn) *bool {
	gopt.failIfDefined([]string{name})
	opt := option.New(name, option.BoolType)
	opt.DefaultStr = fmt.Sprintf("%t", def)
	opt.SetBoolPtr(&def)
	opt.Handler = gopt.handleBool
	for _, fn := range fns {
		fn(opt)
	}
	gopt.completionAppendAliases(opt.Aliases)
	gopt.SetOption(opt)
	return &def
}

// BoolVar - define a `bool` option and its aliases.
// The result will be available through the variable marked by the given pointer.
// If the option is found, the result will be the opposite of the provided default.
func (gopt *GetOpt) BoolVar(p *bool, name string, def bool, fns ...ModifyFn) {
	gopt.Bool(name, def, fns...)
	*p = def
	gopt.Option(name).SetBoolPtr(p)
}

func (gopt *GetOpt) handleBool(name string, argument string, usedAlias string) error {
	Debug.Println("handleBool")
	opt := gopt.Option(name)
	opt.SetCalled(usedAlias)
	return opt.Save()
}

// NBool - define a *Negatable* `bool` option and its aliases.
//
// NBool automatically makes aliases with the prefix 'no' and 'no-' to the given name and aliases.
// If the option is found, when the argument is prefixed by 'no' (or by 'no-'), for example '--no-nflag', the value is set to the provided default.
// Otherwise, with a regular call, for example '--nflag', it is set to the opposite of the default.
func (gopt *GetOpt) NBool(name string, def bool, fns ...ModifyFn) *bool {
	gopt.failIfDefined([]string{name})
	opt := option.New(name, option.BoolType)
	opt.DefaultStr = fmt.Sprintf("%t", def)
	opt.SetBoolPtr(&def)
	opt.Handler = gopt.handleNBool
	for _, fn := range fns {
		fn(opt)
	}
	var aliases []string
	for _, a := range opt.Aliases {
		aliases = append(aliases, "no"+a)
		aliases = append(aliases, "no-"+a)
	}
	gopt.failIfDefined(aliases)
	opt.SetAlias(aliases...)
	gopt.completionAppendAliases(opt.Aliases)
	gopt.SetOption(opt)
	return &def
}

// NBoolVar - define a *Negatable* `bool` option and its aliases.
// The result will be available through the variable marked by the given pointer.
//
// NBoolVar automatically makes aliases with the prefix 'no' and 'no-' to the given name and aliases.
// If the option is found, when the argument is prefixed by 'no' (or by 'no-'), for example '--no-nflag', the value is set to the provided default.
// Otherwise, with a regular call, for example '--nflag', it is set to the opposite of the default.
func (gopt *GetOpt) NBoolVar(p *bool, name string, def bool, fns ...ModifyFn) {
	gopt.NBool(name, def, fns...)
	*p = def
	gopt.Option(name).SetBoolPtr(p)
}

func (gopt *GetOpt) handleNBool(name string, argument string, usedAlias string) error {
	Debug.Println("handleNBool")
	opt := gopt.Option(name)
	opt.SetCalled(usedAlias)
	if !strings.HasPrefix(usedAlias, "no-") {
		return opt.Save()
	}
	return nil
}

func (gopt *GetOpt) handleSingleOption(name string, argument string, usedAlias string) error {
	opt := gopt.Option(name)
	opt.SetCalled(usedAlias)
	if argument != "" {
		return opt.Save(argument)
	}
	if !gopt.args.existsNext() {
		if opt.IsOptional {
			return nil
		}
		return fmt.Errorf(text.ErrorMissingArgument, usedAlias)
	}
	// Check if next arg is option
	if optList, _ := isOption(gopt.args.peekNextValue(), gopt.mode); len(optList) > 0 {
		if opt.IsOptional {
			return nil
		}
		return fmt.Errorf(text.ErrorArgumentWithDash, usedAlias)
	}
	gopt.args.next()
	return opt.Save(gopt.args.value())
}

// String - define a `string` option and its aliases.
// If not called, the return value will be that of the given default `def`.
func (gopt *GetOpt) String(name, def string, fns ...ModifyFn) *string {
	gopt.failIfDefined([]string{name})
	opt := option.New(name, option.StringType)
	opt.DefaultStr = fmt.Sprintf(`"%s"`, def)
	opt.SetStringPtr(&def)
	opt.Handler = gopt.handleSingleOption
	opt.HelpArgName = "string"
	for _, fn := range fns {
		fn(opt)
	}
	gopt.completionAppendAliases(opt.Aliases)
	gopt.SetOption(opt)
	return &def
}

// StringVar - define a `string` option and its aliases.
// The result will be available through the variable marked by the given pointer.
// If not called, the return value will be that of the given default `def`.
func (gopt *GetOpt) StringVar(p *string, name, def string, fns ...ModifyFn) {
	gopt.String(name, def, fns...)
	*p = def
	gopt.Option(name).SetStringPtr(p)
}

// StringOptional - define a `string` option and its aliases.
//
// StringOptional will set the string to the provided default value when no value is given.
// For example, when called with `--strOpt value`, the value is `value`.
// when called with `--strOpt` the value is the given default.
func (gopt *GetOpt) StringOptional(name string, def string, fns ...ModifyFn) *string {
	gopt.failIfDefined([]string{name})
	opt := option.New(name, option.StringType)
	opt.DefaultStr = fmt.Sprintf(`"%s"`, def)
	opt.SetStringPtr(&def)
	opt.IsOptional = true
	opt.Handler = gopt.handleSingleOption
	opt.HelpArgName = "string"
	for _, fn := range fns {
		fn(opt)
	}
	gopt.completionAppendAliases(opt.Aliases)
	gopt.SetOption(opt)
	return &def
}

// StringVarOptional - define a `string` option and its aliases.
// The result will be available through the variable marked by the given pointer.
//
// StringVarOptional will set the string to the provided default value when no value is given.
// For example, when called with `--strOpt value`, the value is `value`.
// when called with `--strOpt` the value is the given default.
func (gopt *GetOpt) StringVarOptional(p *string, name, def string, fns ...ModifyFn) {
	gopt.StringOptional(name, def, fns...)
	*p = def
	gopt.Option(name).SetStringPtr(p)
}

// Int - define an `int` option and its aliases.
func (gopt *GetOpt) Int(name string, def int, fns ...ModifyFn) *int {
	gopt.failIfDefined([]string{name})
	opt := option.New(name, option.IntType)
	opt.DefaultStr = fmt.Sprintf("%d", def)
	opt.SetIntPtr(&def)
	opt.Handler = gopt.handleSingleOption
	opt.HelpArgName = "int"
	for _, fn := range fns {
		fn(opt)
	}
	gopt.completionAppendAliases(opt.Aliases)
	gopt.SetOption(opt)
	return &def
}

// IntVar - define an `int` option and its aliases.
// The result will be available through the variable marked by the given pointer.
func (gopt *GetOpt) IntVar(p *int, name string, def int, fns ...ModifyFn) {
	gopt.Int(name, def, fns...)
	*p = def
	gopt.Option(name).SetIntPtr(p)
}

// IntOptional - define a `int` option and its aliases.
//
// IntOptional will set the int to the provided default value when no value is given.
// For example, when called with `--intOpt 123`, the value is `123`.
// when called with `--intOpt` the value is the given default.
func (gopt *GetOpt) IntOptional(name string, def int, fns ...ModifyFn) *int {
	gopt.failIfDefined([]string{name})
	opt := option.New(name, option.IntType)
	opt.DefaultStr = fmt.Sprintf("%d", def)
	opt.SetIntPtr(&def)
	opt.IsOptional = true
	opt.Handler = gopt.handleSingleOption
	opt.HelpArgName = "int"
	for _, fn := range fns {
		fn(opt)
	}
	gopt.completionAppendAliases(opt.Aliases)
	gopt.SetOption(opt)
	return &def
}

// IntVarOptional - define a `int` option and its aliases.
// The result will be available through the variable marked by the given pointer.
//
// IntOptional will set the int to the provided default value when no value is given.
// For example, when called with `--intOpt 123`, the value is `123`.
// when called with `--intOpt` the value is the given default.
func (gopt *GetOpt) IntVarOptional(p *int, name string, def int, fns ...ModifyFn) {
	gopt.IntOptional(name, def, fns...)
	*p = def
	gopt.Option(name).SetIntPtr(p)
}

// Float64 - define an `float64` option and its aliases.
func (gopt *GetOpt) Float64(name string, def float64, fns ...ModifyFn) *float64 {
	gopt.failIfDefined([]string{name})
	opt := option.New(name, option.Float64Type)
	opt.DefaultStr = fmt.Sprintf("%f", def)
	opt.SetFloat64Ptr(&def)
	opt.Handler = gopt.handleSingleOption
	opt.HelpArgName = "float64"
	for _, fn := range fns {
		fn(opt)
	}
	gopt.completionAppendAliases(opt.Aliases)
	gopt.SetOption(opt)
	return &def
}

// Float64Var - define an `float64` option and its aliases.
// The result will be available through the variable marked by the given pointer.
func (gopt *GetOpt) Float64Var(p *float64, name string, def float64, fns ...ModifyFn) {
	gopt.Float64(name, def, fns...)
	*p = def
	gopt.Option(name).SetFloat64Ptr(p)
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
	gopt.failIfDefined([]string{name})
	opt := option.New(name, option.StringRepeatType)
	opt.DefaultStr = "[]"
	opt.SetStringSlicePtr(&s)
	opt.Handler = gopt.handleSliceMultiOption
	opt.MinArgs = min
	opt.MaxArgs = max
	opt.HelpArgName = "string"
	if min <= 0 {
		panic(fmt.Sprintf("%s min should be > 0", name))
	}
	if max <= 0 || max < min {
		panic(fmt.Sprintf("%s max should be > 0 and > min", name))
	}
	for _, fn := range fns {
		fn(opt)
	}
	Debug.Printf("StringMulti return: %v\n", s)
	gopt.completionAppendAliases(opt.Aliases)
	gopt.SetOption(opt)
	return &s
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
	gopt.StringSlice(name, min, max, fns...)
	gopt.Option(name).SetStringSlicePtr(p)
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
	gopt.failIfDefined([]string{name})
	opt := option.New(name, option.IntRepeatType)
	opt.DefaultStr = "[]"
	opt.SetIntSlicePtr(&s)
	opt.Handler = gopt.handleSliceMultiOption
	opt.MinArgs = min
	opt.MaxArgs = max
	opt.HelpArgName = "int"
	if min <= 0 {
		panic(fmt.Sprintf("%s min should be > 0", name))
	}
	if max <= 0 || max < min {
		panic(fmt.Sprintf("%s max should be > 0 and > min", name))
	}
	for _, fn := range fns {
		fn(opt)
	}
	Debug.Printf("IntMulti return: %v\n", s)
	gopt.completionAppendAliases(opt.Aliases)
	gopt.SetOption(opt)
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
	gopt.IntSlice(name, min, max, fns...)
	gopt.Option(name).SetIntSlicePtr(p)
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
	s := make(map[string]string)
	gopt.failIfDefined([]string{name})
	opt := option.New(name, option.StringMapType)
	opt.DefaultStr = "{}"
	opt.SetStringMapPtr(&s)
	opt.Handler = gopt.handleSliceMultiOption
	opt.MinArgs = min
	opt.MaxArgs = max
	opt.HelpArgName = "key=value"
	if min <= 0 {
		panic(fmt.Sprintf("%s min should be > 0", name))
	}
	if max <= 0 || max < min {
		panic(fmt.Sprintf("%s max should be > 0 and > min", name))
	}
	for _, fn := range fns {
		fn(opt)
	}
	Debug.Printf("StringMulti return: %v\n", s)
	gopt.completionAppendAliases(opt.Aliases)
	gopt.SetOption(opt)
	return s
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
	if *m == nil {
		*m = make(map[string]string)
	}
	gopt.StringMap(name, min, max, fns...)
	gopt.Option(name).SetStringMapPtr(m)
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
		if optList, _ := isOption(gopt.args.peekNextValue(), gopt.mode); len(optList) > 0 {
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
			if err.Error() == fmt.Sprintf("NoMoreArguments") {
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

// Increment - When called multiple times it increments the int counter defined by this option.
func (gopt *GetOpt) Increment(name string, def int, fns ...ModifyFn) *int {
	gopt.failIfDefined([]string{name})
	opt := option.New(name, option.IntType)
	opt.DefaultStr = fmt.Sprintf("%d", def)
	opt.SetIntPtr(&def)
	opt.Handler = gopt.handleIncrement
	for _, fn := range fns {
		fn(opt)
	}
	gopt.completionAppendAliases(opt.Aliases)
	gopt.SetOption(opt)
	return &def
}

// IncrementVar - When called multiple times it increments the provided int.
func (gopt *GetOpt) IncrementVar(p *int, name string, def int, fns ...ModifyFn) {
	gopt.Increment(name, def, fns...)
	*p = def
	gopt.Option(name).SetIntPtr(p)
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
func (gopt *GetOpt) getOptionFromAliases(alias string) (optName, usedAlias string, found bool) {
	found = false
	for name, option := range gopt.obj {
		for _, v := range option.Aliases {
			Debug.Printf("Trying to match '%s' against '%s' alias for '%s'\n", alias, v, name)
			if v == alias {
				found = true
				optName = name
				usedAlias = v
				break
			}
		}
	}
	// Attempt to match initial chars of option
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
		if len(matches) == 1 {
			found = true
			optName = matches[0]
		}
	}
	Debug.Printf("getOptionFromAliases return: %s, %s, %v\n", optName, usedAlias, found)
	return optName, usedAlias, found
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
	// https://stackoverflow.com/a/33396628
	if compLine != "" {
		fmt.Println(strings.Join(gopt.completion.CompLineComplete(compLine), "\n"))
		os.Exit(1)
	}
	al := newArgList(args)
	gopt.args = al
	Debug.Printf("Parse args: %v(%d)\n", args, len(args))
	var remaining []string
	// opt.argsIndex is the index in the opt.args slice.
	// Option handlers will have to know about it, to ask for the next element.
	for gopt.args.next() {
		arg := gopt.args.value()
		Debug.Printf("Parse input arg: %s\n", arg)
		if optList, argument := isOption(arg, gopt.mode); len(optList) > 0 {
			Debug.Printf("Parse opt_list: %v, argument: %v\n", optList, argument)
			// Check for termination: '--'
			if optList[0] == "--" {
				Debug.Printf("Parse -- found\n")
				// move index to next position (to not include '--') and return remaining.
				gopt.args.next()
				remaining = append(remaining, gopt.args.remaining()...)
				Debug.Printf("return %v, %v", remaining, nil)
				return remaining, nil
			}
			Debug.Printf("Parse continue\n")
			for _, optElement := range optList {
				Debug.Printf("Parse optElement: %s\n", optElement)
				if optName, usedAlias, ok := gopt.getOptionFromAliases(optElement); ok {
					Debug.Printf("Parse found opt_list\n")
					opt := gopt.Option(optName)
					handler := opt.Handler
					Debug.Printf("handler found: name %s, argument %s, index %d, list %s\n", optName, argument, gopt.args.index(), optList[0])
					err := handler(optName, argument, usedAlias)
					if err != nil {
						Debug.Printf("handler return: value %v, return %v, %v", opt.Value(), nil, err)
						return nil, err
					}
				} else {
					Debug.Printf("opt_list not found for '%s'\n", optElement)
					switch gopt.unknownMode {
					case Pass:
						if gopt.requireOrder {
							remaining = append(remaining, gopt.args.remaining()...)
							Debug.Printf("Stop on unknown options %s\n", arg)
							Debug.Printf("return %v, %v", remaining, nil)
							return remaining, nil
						}
						remaining = append(remaining, arg)
						break
					case Warn:
						fmt.Fprintf(gopt.Writer, text.MessageOnUnknown, optElement)
						remaining = append(remaining, arg)
						break
					default:
						err := fmt.Errorf(text.MessageOnUnknown, optElement)
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
