// This file is part of go-getoptions.
//
// Copyright (C) 2015-2019  David Gamba Rios
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

/*
Package getoptions - Go option parser inspired on the flexibility of Perl’s
GetOpt::Long.

It will operate on any given slice of strings and return the remaining (non
used) command line arguments. This allows to easily subcommand.


Usage

The following is a basic example:

	package main

	import (
		"fmt"
		"io/ioutil"
		"log"
		"os"

		"github.com/DavidGamba/go-getoptions"
	)

	var logger = log.New(os.Stderr, "DEBUG: ", log.LstdFlags)

	func main() {
		// Declare the variables you want your options to update
		var debug bool
		var greetCount int

		// Declare the GetOptions object
		opt := getoptions.New()

		// Options definition
		opt.Bool("help", false, opt.Alias("h", "?")) // Aliases can be defined
		opt.BoolVar(&debug, "debug", false)
		opt.IntVar(&greetCount, "greet", 0,
			opt.Required(), // Mark option as required
			opt.Description("Number of times to greet."), // Set the automated help description
			opt.ArgName("number"),                        // Change the help synopsis arg from <int> to <number>
		)
		greetings := opt.StringMap("list", 1, 99,
			opt.Description("Greeting list by language."),
			opt.ArgName("lang=msg"), // Change the help synopsis arg from <key=value> to <lang=msg>
		)

		// Parse cmdline arguments or any provided []string
		remaining, err := opt.Parse(os.Args[1:])

		// Handle help before handling user errors
		if opt.Called("help") {
			fmt.Fprintf(os.Stderr, opt.Help())
			os.Exit(1)
		}

		// Handle user errors
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: %s\n\n", err)
			fmt.Fprintf(os.Stderr, opt.HelpSynopsis())
			os.Exit(1)
		}
		if !debug {
			logger.SetOutput(ioutil.Discard)
		}
		logger.Printf("Remaining: %v\n", remaining)

		for i := 0; i < greetCount; i++ {
			fmt.Println("Hello World, from go-getoptions!")
		}
		if len(greetings) > 0 {
			fmt.Printf("Greeting List:\n")
			for k, v := range greetings {
				fmt.Printf("\t%s=%s\n", k, v)
			}
		}
	}

Features

• Allow passing options and non-options in any order.

• Support for `--long` options.

• Support for short (`-s`) options with flexible behaviour (see https://github.com/DavidGamba/go-getoptions#operation_modes for details):

 - Normal (default)
 - Bundling
 - SingleDash

• `Called()` method indicates if the option was passed on the command line.

• Multiple aliases for the same option. e.g. `help`, `man`.

• `CalledAs()` method indicates what alias was used to call the option on the command line.

• Simple synopsis and option list automated help.

• Boolean, String, Int and Float64 type options.

• Negatable Boolean options.
For example: `--verbose`, `--no-verbose` or `--noverbose`.

• Options with Array arguments.
The same option can be used multiple times with different arguments.
The list of arguments will be saved into an Array like structure inside the program.

• Options with array arguments and multiple entries.
For example: `color --rgb 10 20 30 --next-option`

• When using integer array options with multiple arguments, positive integer ranges are allowed.
For example: `1..3` to indicate `1 2 3`.

• Options with key value arguments and multiple entries.

• Options with Key Value arguments.
This allows the same option to be used multiple times with arguments of key value type.
For example: `rpmbuild --define name=myrpm --define version=123`.

• Supports passing `--` to stop parsing arguments (everything after will be left in the `remaining []string`).

• Supports command line options with '='.
For example: You can use `--string=mystring` and `--string mystring`.

• Allows passing arguments to options that start with dash `-` when passed after equal.
For example: `--string=--hello` and `--int=-123`.

• Options with optional arguments.
If the default argument is not passed the default is set.
For example: You can call `--int 123` which yields `123` or `--int` which yields the given default.

• Allows abbreviations when the provided option is not ambiguous.
For example: An option called `build` can be called with `--b`, `--bu`, `--bui`, `--buil` and `--build` as long as there is no ambiguity.
In the case of ambiguity, the shortest non ambiguous combination is required.

• Support for the lonesome dash "-".
To indicate, for example, when to read input from STDIO.

• Incremental options.
Allows the same option to be called multiple times to increment a counter.

• Supports case sensitive options.
For example, you can use `v` to define `verbose` and `V` to define `Version`.

• Support indicating if an option is required and allows overriding default error message.

• Errors exposed as public variables to allow overriding them for internationalization.

• Supports subcommands (stop parsing arguments when non option is passed).

• Multiple ways of managing unknown options:
  - Fail on unknown (default).
  - Warn on unknown.
  - Pass through, allows for subcommands and can be combined with Require Order.

• Require order: Allows for subcommands. Stop parsing arguments when the first non-option is found.
When mixed with Pass through, it also stops parsing arguments when the first unmatched option is found.

Panic

The library will panic if it finds that the programmer (not end user):

• Defined the same alias twice.

• Defined wrong min and max values for SliceMulti methods.
*/
package getoptions

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Debug Logger instance set to `ioutil.Discard` by default.
// Enable debug logging by setting: `Debug.SetOutput(os.Stderr)`.
var Debug = log.New(ioutil.Discard, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)

// GetOpt - main object.
type GetOpt struct {
	mode           string    // Operation mode for short options: normal, bundling, singleDash
	unknownMode    string    // Unknown option mode
	requireOrder   bool      // Stop parsing on non option
	mapKeysToLower bool      // Set Map keys lower case
	Writer         io.Writer // io.Writer locations to write warnings to. Defaults to os.Stderr.
	obj            map[string]*option
	args           *argList
}

// ModifyFn - Function signature for functions that modify an option.
type ModifyFn func(*option)

// handlerType - method used to handle the option.
type handlerType func(optName string, argument string, usedAlias string) error

// New returns an empty object of type GetOpt.
// This is the starting point when using go-getoptions.
// For example:
//
//   opt := getoptions.New()
func New() *GetOpt {
	gopt := &GetOpt{
		obj:    make(map[string]*option),
		Writer: os.Stderr,
	}
	return gopt
}

// failIfDefined will *panic* if an option is defined twice.
// This is not an error because the programmer has to fix this!
func (gopt *GetOpt) failIfDefined(aliases []string) {
	for _, a := range aliases {
		for _, option := range gopt.obj {
			for _, v := range option.aliases {
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
		return v.called
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
		return v.usedAlias
	}
	return ""
}

// Option - Returns the value of the given option.
//
// Type assertions are required in cases where the compiler can't determine the type by context.
// For example: `opt.Option("flag").(bool)`.
func (gopt *GetOpt) Option(name string) interface{} {
	opt := gopt.option(name)
	if opt != nil {
		return opt.value
	}
	return nil
}

// option - Returns the *option for name.
func (gopt *GetOpt) option(name string) *option {
	if value, ok := gopt.obj[name]; ok {
		return value
	}
	return nil
}

// option - Sets the *option for name.
func (gopt *GetOpt) setOption(name string, opt *option) {
	gopt.obj[name] = opt
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
func (gopt *GetOpt) SetMode(mode string) {
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
func (gopt *GetOpt) SetUnknownMode(mode string) {
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
	return func(opt *option) {
		opt.setAlias(alias...)
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
	return func(opt *option) {
		opt.setRequired(errTxt)
	}
}

// Description - Add a description to an option for use in automated help.
func (gopt *GetOpt) Description(msg string) ModifyFn {
	return func(opt *option) {
		opt.description = msg
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
	return func(opt *option) {
		opt.helpArgName = name
	}
}

// Help - Default help string that is composed of the HelpSynopsis and HelpOptionList.
func (gopt *GetOpt) Help() string {
	return gopt.HelpSynopsis() + "\n" + gopt.HelpOptionList()
}

// HelpSynopsis - Return a default synopsis.
func (gopt *GetOpt) HelpSynopsis() string {
	scriptName := filepath.Base(os.Args[0])
	optionNames := []string{}
	requiredNames := []string{}
	for _, option := range gopt.obj {
		if option.isRequired {
			requiredNames = append(requiredNames, option.name)
		} else {
			optionNames = append(optionNames, option.name)
		}
	}
	sort.Strings(optionNames)
	sort.Strings(requiredNames)
	optSynopsis := func(name string) string {
		txt := ""
		aliases := []string{}
		for _, alias := range gopt.option(name).aliases {
			if len(alias) > 1 {
				aliases = append(aliases, fmt.Sprintf("--%s", alias))
			} else {
				aliases = append(aliases, fmt.Sprintf("-%s", alias))
			}
		}
		aliasStr := strings.Join(aliases, "|")
		open := ""
		close := ""
		if !gopt.option(name).isRequired {
			open = "["
			close = "]"
		}
		argName := gopt.option(name).helpArgName
		switch gopt.option(name).optType {
		case boolType:
			txt += fmt.Sprintf("%s%s%s", open, aliasStr, close)
		case stringType, intType, float64Type:
			txt += fmt.Sprintf("%s%s <%s>%s", open, aliasStr, argName, close)
		case stringRepeatType, intRepeatType, stringMapType:
			if gopt.option(name).isRequired {
				open = "<"
				close = ">"
			}
			repeat := ""
			if gopt.option(name).maxArgs > 1 {
				repeat = "..."
			}
			txt += fmt.Sprintf("%s%s <%s>%s%s...", open, aliasStr, argName, repeat, close)
		}
		return txt
	}
	var out string
	line := scriptName
	for _, name := range append(requiredNames, optionNames...) {
		syn := optSynopsis(name)
		// fmt.Printf("%d - %d - %d | %s | %s\n", len(line), len(syn), len(line)+len(syn), syn, line)
		if len(line)+len(syn) > 80 {
			out += line + "\n"
			line = fmt.Sprintf("%s %s", strings.Repeat(" ", len(scriptName)), syn)
		} else {
			line += fmt.Sprintf(" %s", syn)
		}
	}
	out += line
	return fmt.Sprintf("%s:\n%s\n", HelpSynopsisHeader, out)
}

// HelpOptionList - Return a formatted list of options and their descriptions.
func (gopt *GetOpt) HelpOptionList() string {
	aliasListLength := 0
	optionNames := []string{}
	requiredNames := []string{}
	for _, option := range gopt.obj {
		l := len(option.aliases)
		for _, alias := range option.aliases {
			// --alias || -a
			l += len(alias) + 1
			if len(alias) > 1 {
				l++
			}
		}
		if l > aliasListLength {
			aliasListLength = l
		}
		if option.isRequired {
			requiredNames = append(requiredNames, option.name)
		} else {
			optionNames = append(optionNames, option.name)
		}
	}
	sort.Strings(optionNames)
	sort.Strings(requiredNames)
	helpString := func(nameList []string) string {
		txt := ""
		for _, name := range nameList {
			aliases := []string{}
			for _, alias := range gopt.option(name).aliases {
				if len(alias) > 1 {
					aliases = append(aliases, fmt.Sprintf("--%s", alias))
				} else {
					aliases = append(aliases, fmt.Sprintf("-%s", alias))
				}
			}
			aliasStr := strings.Join(aliases, "|")
			factor := aliasListLength + 16
			padding := strings.Repeat(" ", factor)
			pad := func(s string, factor int) string {
				return fmt.Sprintf("%-"+strconv.Itoa(factor)+"s", s)
			}
			argName := gopt.option(name).helpArgName
			switch gopt.option(name).optType {
			case boolType:
				txt += fmt.Sprintf("    %s", pad(aliasStr+"", factor))
			case stringType, intType, float64Type:
				txt += fmt.Sprintf("    %s", pad(aliasStr+" <"+argName+">", factor))
			case stringRepeatType, intRepeatType, stringMapType:
				txt += fmt.Sprintf("    %s", pad(aliasStr+" <"+argName+">...", factor))
			}
			if gopt.option(name).description != "" {
				description := strings.Replace(gopt.option(name).description, "\n", "\n    "+padding, -1)
				txt += fmt.Sprintf("%s ", description)
			}
			if !gopt.option(name).isRequired {
				txt += fmt.Sprintf("(default: %s)\n\n", gopt.option(name).defaultStr)
			} else {
				txt += "\n\n"
			}
		}
		return txt
	}
	out := ""
	if len(requiredNames) > 0 {
		out += fmt.Sprintf("%s:\n%s", HelpRequiredOptionsHeader, helpString(requiredNames))
	}
	if len(optionNames) > 0 {
		out += fmt.Sprintf("%s:\n%s", HelpOptionsHeader, helpString(optionNames))
	}
	return out
}

// Bool - define a `bool` option and its aliases.
// It returns a `*bool` pointing to the variable holding the result.
// If the option is found, the result will be the opposite of the provided default.
func (gopt *GetOpt) Bool(name string, def bool, fns ...ModifyFn) *bool {
	gopt.failIfDefined([]string{name})
	gopt.setOption(name, newOption(name, []string{name}))
	gopt.option(name).defaultStr = fmt.Sprintf("%t", def)
	gopt.option(name).setBoolPtr(&def)
	gopt.option(name).setHandler(gopt.handleBool)
	gopt.option(name).optType = boolType
	for _, fn := range fns {
		fn(gopt.option(name))
	}
	return &def
}

// BoolVar - define a `bool` option and its aliases.
// The result will be available through the variable marked by the given pointer.
// If the option is found, the result will be the opposite of the provided default.
func (gopt *GetOpt) BoolVar(p *bool, name string, def bool, fns ...ModifyFn) {
	gopt.Bool(name, def, fns...)
	*p = def
	gopt.option(name).setBoolPtr(p)
}

func (gopt *GetOpt) handleBool(name string, argument string, usedAlias string) error {
	Debug.Println("handleBool")
	opt := gopt.option(name)
	opt.setCalled()
	opt.usedAlias = usedAlias
	return opt.save(name)
}

// NBool - define a *Negatable* `bool` option and its aliases.
//
// NBool automatically makes aliases with the prefix 'no' and 'no-' to the given name and aliases.
// If the option is found, when the argument is prefixed by 'no' (or by 'no-'), for example '--no-nflag', the value is set to the provided default.
// Otherwise, with a regular call, for example '--nflag', it is set to the opposite of the default.
func (gopt *GetOpt) NBool(name string, def bool, fns ...ModifyFn) *bool {
	gopt.failIfDefined([]string{name})
	gopt.setOption(name, newOption(name, []string{name}))
	gopt.option(name).defaultStr = fmt.Sprintf("%t", def)
	gopt.option(name).setBoolPtr(&def)
	gopt.option(name).setHandler(gopt.handleNBool)
	gopt.option(name).optType = boolType
	for _, fn := range fns {
		fn(gopt.option(name))
	}
	var aliases []string
	for _, a := range gopt.option(name).aliases {
		aliases = append(aliases, "no"+a)
		aliases = append(aliases, "no-"+a)
	}
	gopt.failIfDefined(aliases)
	gopt.option(name).setAlias(aliases...)
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
	gopt.option(name).setBoolPtr(p)
}

func (gopt *GetOpt) handleNBool(name string, argument string, usedAlias string) error {
	Debug.Println("handleNBool")
	opt := gopt.option(name)
	opt.setCalled()
	opt.usedAlias = usedAlias
	if !strings.HasPrefix(usedAlias, "no-") {
		return opt.save(name)
	}
	return nil
}

func (gopt *GetOpt) handleSingleOption(name string, argument string, usedAlias string) error {
	opt := gopt.option(name)
	opt.setCalled()
	opt.usedAlias = usedAlias
	if argument != "" {
		return opt.save(name, argument)
	}
	if !gopt.args.existsNext() {
		if opt.isOptional() {
			return nil
		}
		return fmt.Errorf(ErrorMissingArgument, usedAlias)
	}
	// Check if next arg is option
	if optList, _ := isOption(gopt.args.peekNextValue(), gopt.mode); len(optList) > 0 {
		if opt.isOptional() {
			return nil
		}
		return fmt.Errorf(ErrorArgumentWithDash, usedAlias)
	}
	gopt.args.next()
	return opt.save(name, gopt.args.value())
}

// String - define a `string` option and its aliases.
// If not called, the return value will be that of the given default `def`.
func (gopt *GetOpt) String(name, def string, fns ...ModifyFn) *string {
	gopt.failIfDefined([]string{name})
	gopt.setOption(name, newOption(name, []string{name}))
	gopt.option(name).defaultStr = fmt.Sprintf(`"%s"`, def)
	gopt.option(name).setStringPtr(&def)
	gopt.option(name).setHandler(gopt.handleSingleOption)
	gopt.option(name).optType = stringType
	gopt.option(name).helpArgName = "string"
	for _, fn := range fns {
		fn(gopt.option(name))
	}
	return &def
}

// StringVar - define a `string` option and its aliases.
// The result will be available through the variable marked by the given pointer.
// If not called, the return value will be that of the given default `def`.
func (gopt *GetOpt) StringVar(p *string, name, def string, fns ...ModifyFn) {
	gopt.String(name, def, fns...)
	*p = def
	gopt.option(name).setStringPtr(p)
}

// StringOptional - define a `string` option and its aliases.
//
// StringOptional will set the string to the provided default value when no value is given.
// For example, when called with `--strOpt value`, the value is `value`.
// when called with `--strOpt` the value is the given default.
func (gopt *GetOpt) StringOptional(name string, def string, fns ...ModifyFn) *string {
	gopt.failIfDefined([]string{name})
	gopt.setOption(name, newOption(name, []string{name}))
	gopt.option(name).defaultStr = fmt.Sprintf(`"%s"`, def)
	gopt.option(name).setStringPtr(&def)
	gopt.option(name).setIsOptional()
	gopt.option(name).setHandler(gopt.handleSingleOption)
	gopt.option(name).optType = stringType
	gopt.option(name).helpArgName = "string"
	for _, fn := range fns {
		fn(gopt.option(name))
	}
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
	gopt.option(name).setStringPtr(p)
}

// Int - define an `int` option and its aliases.
func (gopt *GetOpt) Int(name string, def int, fns ...ModifyFn) *int {
	gopt.failIfDefined([]string{name})
	gopt.setOption(name, newOption(name, []string{name}))
	gopt.option(name).defaultStr = fmt.Sprintf("%d", def)
	gopt.option(name).setIntPtr(&def)
	gopt.option(name).setHandler(gopt.handleSingleOption)
	gopt.option(name).optType = intType
	gopt.option(name).helpArgName = "int"
	for _, fn := range fns {
		fn(gopt.option(name))
	}
	return &def
}

// IntVar - define an `int` option and its aliases.
// The result will be available through the variable marked by the given pointer.
func (gopt *GetOpt) IntVar(p *int, name string, def int, fns ...ModifyFn) {
	gopt.Int(name, def, fns...)
	*p = def
	gopt.option(name).setIntPtr(p)
}

// IntOptional - define a `int` option and its aliases.
//
// IntOptional will set the int to the provided default value when no value is given.
// For example, when called with `--intOpt 123`, the value is `123`.
// when called with `--intOpt` the value is the given default.
func (gopt *GetOpt) IntOptional(name string, def int, fns ...ModifyFn) *int {
	gopt.failIfDefined([]string{name})
	gopt.setOption(name, newOption(name, []string{name}))
	gopt.option(name).defaultStr = fmt.Sprintf("%d", def)
	gopt.option(name).setIntPtr(&def)
	gopt.option(name).setIsOptional()
	gopt.option(name).setHandler(gopt.handleSingleOption)
	gopt.option(name).optType = intType
	gopt.option(name).helpArgName = "int"
	for _, fn := range fns {
		fn(gopt.option(name))
	}
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
	gopt.option(name).setIntPtr(p)
}

// Float64 - define an `float64` option and its aliases.
func (gopt *GetOpt) Float64(name string, def float64, fns ...ModifyFn) *float64 {
	gopt.failIfDefined([]string{name})
	gopt.setOption(name, newOption(name, []string{name}))
	gopt.option(name).defaultStr = fmt.Sprintf("%f", def)
	gopt.option(name).setFloat64Ptr(&def)
	gopt.option(name).setHandler(gopt.handleSingleOption)
	gopt.option(name).optType = float64Type
	gopt.option(name).helpArgName = "float64"
	for _, fn := range fns {
		fn(gopt.option(name))
	}
	return &def
}

// Float64Var - define an `float64` option and its aliases.
// The result will be available through the variable marked by the given pointer.
func (gopt *GetOpt) Float64Var(p *float64, name string, def float64, fns ...ModifyFn) {
	gopt.Float64(name, def, fns...)
	*p = def
	gopt.option(name).setFloat64Ptr(p)
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
	gopt.setOption(name, newOption(name, []string{name}))
	gopt.option(name).defaultStr = "[]"
	gopt.option(name).setStringSlicePtr(&s)
	gopt.option(name).setHandler(gopt.handleSliceMultiOption)
	gopt.option(name).setMin(min)
	gopt.option(name).setMax(max)
	gopt.option(name).optType = stringRepeatType
	gopt.option(name).helpArgName = "string"
	if min <= 0 {
		panic(fmt.Sprintf("%s min should be > 0", name))
	}
	if max <= 0 || max < min {
		panic(fmt.Sprintf("%s max should be > 0 and > min", name))
	}
	for _, fn := range fns {
		fn(gopt.option(name))
	}
	Debug.Printf("StringMulti return: %v\n", s)
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
	gopt.option(name).setStringSlicePtr(p)
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
	gopt.setOption(name, newOption(name, []string{name}))
	gopt.option(name).defaultStr = "[]"
	gopt.option(name).setIntSlicePtr(&s)
	gopt.option(name).setHandler(gopt.handleSliceMultiOption)
	gopt.option(name).setMin(min)
	gopt.option(name).setMax(max)
	gopt.option(name).optType = intRepeatType
	gopt.option(name).helpArgName = "int"
	if min <= 0 {
		panic(fmt.Sprintf("%s min should be > 0", name))
	}
	if max <= 0 || max < min {
		panic(fmt.Sprintf("%s max should be > 0 and > min", name))
	}
	for _, fn := range fns {
		fn(gopt.option(name))
	}
	Debug.Printf("IntMulti return: %v\n", s)
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
	gopt.option(name).setIntSlicePtr(p)
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
	gopt.setOption(name, newOption(name, []string{name}))
	gopt.option(name).defaultStr = "{}"
	gopt.option(name).setStringMap(s)
	gopt.option(name).setHandler(gopt.handleSliceMultiOption)
	gopt.option(name).setMin(min)
	gopt.option(name).setMax(max)
	gopt.option(name).optType = stringMapType
	gopt.option(name).helpArgName = "key=value"
	if min <= 0 {
		panic(fmt.Sprintf("%s min should be > 0", name))
	}
	if max <= 0 || max < min {
		panic(fmt.Sprintf("%s max should be > 0 and > min", name))
	}
	for _, fn := range fns {
		fn(gopt.option(name))
	}
	Debug.Printf("StringMulti return: %v\n", s)
	return s
}

// NOTE: Options that can be called multiple times and thus modify the used
// alias, don't use usedAlias for their errors because the error is used to
// check the min, max args.
// TODO: Do a regex instead of matching the full error to enable usedAlias in errors.
func (gopt *GetOpt) handleSliceMultiOption(name string, argument string, usedAlias string) error {
	Debug.Printf("handleStringSlice\n")
	opt := gopt.option(name)
	opt.setCalled()
	opt.usedAlias = usedAlias
	opt.mapKeysToLower = gopt.mapKeysToLower
	argCounter := 0

	if argument != "" {
		argCounter++
		err := opt.save(name, argument)
		if err != nil {
			return err
		}
	}
	// Function to handle one arg at a time
	next := func(required bool) error {
		Debug.Printf("total arguments: %d, index: %d, counter %d", gopt.args.size(), gopt.args.index(), argCounter)
		if !gopt.args.existsNext() {
			if required {
				return fmt.Errorf(ErrorMissingArgument, name)
			}
			return fmt.Errorf("NoMoreArguments")
		}
		// Check if next arg is option
		if optList, _ := isOption(gopt.args.peekNextValue(), gopt.mode); len(optList) > 0 {
			Debug.Printf("Next arg is option: %s\n", gopt.args.peekNextValue())
			return fmt.Errorf(ErrorArgumentWithDash, name)
		}
		// Check if next arg is not key=value
		if opt.optType == stringMapType && !strings.Contains(gopt.args.peekNextValue(), "=") {
			if required {
				return fmt.Errorf(ErrorArgumentIsNotKeyValue, name)
			}
			return nil
		}
		if opt.optType == intRepeatType {
			_, err := strconv.Atoi(gopt.args.peekNextValue())
			if !required && err != nil {
				return nil
			}
		}
		gopt.args.next()
		return opt.save(name, gopt.args.value())
	}

	// Go through the required and optional iterations
	for argCounter < opt.max() {
		argCounter++
		err := next(argCounter <= opt.min())
		Debug.Printf("counter: %d, value: %v, err %v", argCounter, opt.value, err)
		if err != nil {
			if err.Error() == fmt.Sprintf("NoMoreArguments") {
				Debug.Printf("return value: %v", opt.value)
				return nil
			}
			// always fail if errors under min args
			// After min args, skip missing arg errors
			if argCounter <= opt.min() ||
				(err.Error() != fmt.Sprintf(ErrorMissingArgument, name) &&
					err.Error() != fmt.Sprintf(ErrorArgumentWithDash, name)) {
				Debug.Printf("return value: %v, err: %v", opt.value, err)
				return err
			}
			Debug.Printf("return value: %v", opt.value)
			return nil
		}
	}
	Debug.Printf("return value: %v", opt.value)
	return nil
}

// Increment - When called multiple times it increments the int counter defined by this option.
func (gopt *GetOpt) Increment(name string, def int, fns ...ModifyFn) *int {
	gopt.failIfDefined([]string{name})
	gopt.setOption(name, newOption(name, []string{name}))
	gopt.option(name).defaultStr = fmt.Sprintf("%d", def)
	gopt.option(name).setIntPtr(&def)
	gopt.option(name).setHandler(gopt.handleIncrement)
	for _, fn := range fns {
		fn(gopt.option(name))
	}
	return &def
}

// IncrementVar - When called multiple times it increments the provided int.
func (gopt *GetOpt) IncrementVar(p *int, name string, def int, fns ...ModifyFn) {
	gopt.Increment(name, def, fns...)
	*p = def
	gopt.option(name).setIntPtr(p)
}

func (gopt *GetOpt) handleIncrement(name string, argument string, usedAlias string) error {
	Debug.Println("handleIncrement")
	opt := gopt.option(name)
	opt.setCalled()
	opt.usedAlias = usedAlias
	opt.setInt(opt.getInt() + 1)
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
		switch v := opt.value.(type) {
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
		for _, v := range option.aliases {
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
			for _, v := range option.aliases {
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
					opt := gopt.option(optName)
					handler := opt.handler
					Debug.Printf("handler found: name %s, argument %s, index %d, list %s\n", optName, argument, gopt.args.index(), optList[0])
					err := handler(optName, argument, usedAlias)
					if err != nil {
						Debug.Printf("handler return: value %v, return %v, %v", opt.value, nil, err)
						return nil, err
					}
				} else {
					Debug.Printf("opt_list not found for '%s'\n", optElement)
					switch gopt.unknownMode {
					case "pass":
						if gopt.requireOrder {
							remaining = append(remaining, gopt.args.remaining()...)
							Debug.Printf("Stop on unknown options %s\n", arg)
							Debug.Printf("return %v, %v", remaining, nil)
							return remaining, nil
						}
						remaining = append(remaining, arg)
						break
					case "warn":
						fmt.Fprintf(gopt.Writer, MessageOnUnknown, optElement)
						remaining = append(remaining, arg)
						break
					default:
						err := fmt.Errorf(MessageOnUnknown, optElement)
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
		if option.isRequired {
			if !option.called {
				var err error
				if option.isRequiredErr != "" {
					err = fmt.Errorf(option.isRequiredErr)
				} else {
					err = fmt.Errorf(ErrorMissingRequiredOption, option.name)
				}
				Debug.Printf("return %v, %v", nil, err)
				return nil, err
			}
		}
	}
	Debug.Printf("return %v, %v", remaining, nil)
	return remaining, nil
}
