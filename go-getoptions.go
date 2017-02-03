// This file is part of go-getoptions.
//
// Copyright (C) 2015  David Gamba Rios
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

		import "github.com/davidgamba/go-getoptions" // As getoptions

		// Declare the GetOptions object
		opt := getoptions.New()

		// Use methods that return pointers
		bp := opt.Bool("bp", false)
		sp := opt.String("sp", "")
		ip := opt.Int("ip", 0)

		// Use methods by passing pointers
		var b bool
		var s string
		var i int
		opt.BoolVar(&b, "b", true, "alias", "alias2") // Aliases can be defined
		opt.StringVar(&s, "s", "")
		opt.IntVar(&i, "i", 456)

		// Parse cmdline arguments or any provided []string
		remaining, err := opt.Parse(os.Args[1:])

		if *bp {
			// ... do something
		}

		if opt.Called("i") {
			// ... do something with i
		}

		// Use subcommands by operating on the remaining items
		// Requires `opt.SetRequireOrder()` before the initial `opt.Parse` call.
		opt2 := getoptions.New()
		// ...
		remaining2, err := opt.Parse(remaining)


Features

* Allow passing options and non-options in any order.

* Support for `--long` options.

* Support for short (`-s`) options with flexible behaviour (see https://github.com/DavidGamba/go-getoptions#operation_modes for details):

 - Normal (default)
 - Bundling
 - SingleDash

* Boolean, String, Int and Float64 type options.

* Multiple aliases for the same option. e.g. `help`, `man`.

* Negatable Boolean options.
For example: `--verbose`, `--no-verbose` or `--noverbose`.

* Options with Array arguments.
The same option can be used multiple times with different arguments.
The list of arguments will be saved into an Array like structure inside the program.

* Options with array arguments and multiple entries.

* Options with Key Value arguments.
This allows the same option to be used multiple times with arguments of key value type.
For example: `rpmbuild --define name=myrpm --define version=123`

* Supports passing `--` to stop parsing arguments (everything after will be left in the `remaining []string`).

* Supports subcommands (stop parsing arguments when non option is passed).

* Supports command line options with '='.
For example: You can use `--string=mystring` and `--string mystring`.

* Allows passing arguments to options that start with dash `-` when passed after equal.
For example: `--string=--hello` and `--int=-123`.

* Options with optional arguments.
If the default argument is not passed the default is set.

* Allows abbreviations when the provided option is not ambiguous.

* Called method indicates if the option was passed on the command line.

* Errors exposed as public variables to allow overriding them for internationalization.

* Multiple ways of managing unknown options:
  - Fail on unknown (default).
  - Warn on unknown.
  - Pass through, allows for subcommands and can be combined with Require Order.

* Require order: Allows for subcommands. Stop parsing arguments when the first non-option is found.
When mixed with Pass through, it also stops parsing arguments when the first unmatched option is found.

* Support for the lonesome dash "-".
To indicate, for example, when to read input from STDIO.

* Incremental options.
Allows the same option to be called multiple times to increment a counter.

* Supports case sensitive options.
For example, you can use `v` to define `verbose` and `V` to define `Version`.

Panic

The library will panic if it finds that the programmer defined the same alias twice.
*/
package getoptions

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// Debug Logger instance set to `ioutil.Discard` by default.
// Enable debug logging by setting: `Debug.SetOutput(os.Stderr)`
var Debug = log.New(ioutil.Discard, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)

// GetOpt - main object
type GetOpt struct {
	value        map[string]interface{} // Map with resulting variables
	mode         string                 // Operation mode for short options: normal, bundling, singleDash
	unknownMode  string                 // Unknown option mode
	requireOrder bool                   // Stop parsing on non option
	Writer       io.Writer              // io.Writer locations to write warnings to. Defaults to os.Stderr.
	obj          map[string]*option
	args         []string
	argsIndex    int
}

type option struct {
	name    string
	aliases []string
	def     interface{} // default value
	called  bool        // Indicates if the option was passed on the command line.
	handler func(optName string,
		argument string,
		usedAlias string) error // method used to handle the option
	// Pointer receivers:
	pBool    *bool             // receiver for bool pointer
	pInt     *int              // receiver for int pointer
	pString  *string           // receiver for string pointer
	pFloat64 *float64          // receiver for float64 pointer
	pStringS *[]string         // receiver for string slice pointer
	stringM  map[string]string // receiver for string map pointer
	min      int               // minimum args when using multi
	max      int               // maximum args when using multi
}

func (opt *option) setCalled() {
	opt.called = true
}

// New returns an empty object of type GetOpt.
// This is the starting point when using go-getoptions.
// For example:
//
//   opt := getoptions.New()
func New() *GetOpt {
	gopt := &GetOpt{
		value:  make(map[string]interface{}),
		obj:    make(map[string]*option),
		Writer: os.Stderr,
	}
	return gopt
}

// User facing errors

// ErrorMissingArgument holds the text for missing argument error.
// It has a string placeholder '%s' for the name of the option missing the argument.
var ErrorMissingArgument = "Missing argument for option '%s'!"

// ErrorArgumentIsNotKeyValue holds the text for Map type options where the argument is not of key=value type.
// It has a string placeholder '%s' for the name of the option missing the argument.
var ErrorArgumentIsNotKeyValue = "Argument error for option '%s': Should be of type 'key=value'!"

// ErrorArgumentWithDash holds the text for missing argument error in cases where the next argument looks like an option (starts with '-').
// It has a string placeholder '%s' for the name of the option missing the argument.
var ErrorArgumentWithDash = "Missing argument for option '%s'!\n" +
	"If passing arguments that start with '-' use --option=-argument"

// ErrorConvertToInt holds the text for Int Coversion argument error.
// It has two string placeholders ('%s'). The first one for the name of the option with the wrong argument and the second one for the argument that could not be converted.
var ErrorConvertToInt = "Argument error for option '%s': Can't convert string to int: '%s'"

// ErrorConvertToFloat64 holds the text for Float64 Coversion argument error.
// It has two string placeholders ('%s'). The first one for the name of the option with the wrong argument and the second one for the argument that could not be converted.
var ErrorConvertToFloat64 = "Argument error for option '%s': Can't convert string to float64: '%s'"

// MessageOnUnknown holds the text for the unknown option message.
// It has a string placeholder '%s' for the name of the option missing the argument.
var MessageOnUnknown = "Unknown option '%s'"

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
func (gopt *GetOpt) Called(name string) bool {
	return gopt.obj[name].called
}

// Option - Returns the value of the given option.
//
// Type assertions are required in cases where the compiler can't determine the type by context.
// For example: `opt.Option("flag").(bool)`.
func (gopt *GetOpt) Option(name string) interface{} {
	return gopt.value[name]
}

// option - Returns the *option for name.
func (gopt *GetOpt) option(name string) *option {
	return gopt.obj[name]
}

// option - Sets the *option for name.
func (gopt *GetOpt) setOption(name string, opt *option) {
	gopt.obj[name] = opt
}

// SetMode - Sets the Operation Mode.
// The operation mode only affects options starting with a single dash '-'.
// The available operation modes are: normal, bundling or singleDash
//
// The following table shows the different operation modes given the string "-opt=arg"
//
//     .Operation Modes for string "-opt=arg"
//     |===
//     |normal           |bundling       |singleDash
//
//     |option: "opt"    |option: o      |option: o
//     | argument: "arg" | argument: nil | argument: pt=arg
//     |                 |option: p      |
//     |                 | argument: nil |
//     |                 |option: t      |
//     |                 | argument: arg |
//
//     |===
func (gopt *GetOpt) SetMode(mode string) {
	gopt.mode = mode
}

// SetUnknownMode - Determines how to behave when encountering an unknown option.
//
// - 'fail' (default) will make 'Parse' return an error with the unknown option information.
//
// - 'warn' will make 'Parse' print a user warning indicating there was an unknown option.
// The unknown option will be left in the remaining array.
//
// - 'pass' will make 'Parse' ignore any unknown options and they will be passed onto the 'remaining' slice.
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

// Bool - define a `bool` option and its aliases.
// It returnns a `*bool` pointing to the variable holding the result.
// If the option is found, the result will be the opposite of the provided default.
func (gopt *GetOpt) Bool(name string, def bool, aliases ...string) *bool {
	var b bool
	b = def
	gopt.value[name] = def
	aliases = append(aliases, name)
	gopt.failIfDefined(aliases)
	gopt.setOption(name, &option{
		name:    name,
		aliases: aliases,
		pBool:   &b,
		def:     def,
		handler: gopt.handleBool})
	return &b
}

// BoolVar - define a `bool` option and its aliases.
// The result will be available through the variable marked by the given pointer.
// If the option is found, the result will be the opposite of the provided default.
func (gopt *GetOpt) BoolVar(p *bool, name string, def bool, aliases ...string) {
	gopt.Bool(name, def, aliases...)
	*p = def
	opt := gopt.option(name)
	opt.pBool = p
}

func (gopt *GetOpt) handleBool(name string, argument string, usedAlias string) error {
	Debug.Println("handleBool")
	opt := gopt.option(name)
	opt.setCalled()
	var b bool
	b = !opt.def.(bool)
	gopt.value[name] = b
	*opt.pBool = b
	return nil
}

// NBool - define a *Negatable* `bool` option and its aliases.
//
// NBool automatically makes aliases with the prefix 'no' and 'no-' to the given name and aliases.
// If the option is found, when the argument is prefixed by 'no' (or by 'no-'), for example '--no-nflag', the value is set to the provided default.
// Otherwise, with a regular call, for example '--nflag', it is set to the opposite of the default.
func (gopt *GetOpt) NBool(name string, def bool, aliases ...string) *bool {
	var b bool
	b = def
	gopt.value[name] = def
	aliases = append(aliases, name)
	for _, a := range aliases {
		aliases = append(aliases, "no"+a)
		aliases = append(aliases, "no-"+a)
	}
	gopt.failIfDefined(aliases)
	gopt.setOption(name, &option{
		name:    name,
		aliases: aliases,
		pBool:   &b,
		def:     def,
		handler: gopt.handleNBool})
	return &b
}

// NBoolVar - define a *Negatable* `bool` option and its aliases.
// The result will be available through the variable marked by the given pointer.
//
// NBoolVar automatically makes aliases with the prefix 'no' and 'no-' to the given name and aliases.
// If the option is found, when the argument is prefixed by 'no' (or by 'no-'), for example '--no-nflag', the value is set to the provided default.
// Otherwise, with a regular call, for example '--nflag', it is set to the opposite of the default.
func (gopt *GetOpt) NBoolVar(p *bool, name string, def bool, aliases ...string) {
	gopt.NBool(name, def, aliases...)
	*p = def
	opt := gopt.option(name)
	opt.pBool = p
}

func (gopt *GetOpt) handleNBool(name string, argument string, usedAlias string) error {
	Debug.Println("handleNBool")
	opt := gopt.option(name)
	opt.setCalled()
	var b bool
	b = opt.def.(bool)
	if strings.HasPrefix(usedAlias, "no-") {
		gopt.value[name] = b
		*opt.pBool = b
	} else {
		gopt.value[name] = !b
		*opt.pBool = !b
	}
	return nil
}

// String - define a `string` option and its aliases.
// If not called, the return value will be that of the given default `def`.
func (gopt *GetOpt) String(name, def string, aliases ...string) *string {
	var s string
	s = def
	gopt.value[name] = s
	aliases = append(aliases, name)
	gopt.failIfDefined(aliases)
	gopt.setOption(name, &option{
		name:    name,
		aliases: aliases,
		pString: &s,
		handler: gopt.handleString,
	})
	return &s
}

// StringVar - define a `string` option and its aliases.
// The result will be available through the variable marked by the given pointer.
// If not called, the return value will be that of the given default `def`.
func (gopt *GetOpt) StringVar(p *string, name, def string, aliases ...string) {
	gopt.String(name, def, aliases...)
	*p = def
	opt := gopt.option(name)
	opt.pString = p
}

func (gopt *GetOpt) handleString(name string, argument string, usedAlias string) error {
	Debug.Printf("handleString opt.args: %v(%d)\n", gopt.args, len(gopt.args))
	opt := gopt.option(name)
	opt.setCalled()
	if argument != "" {
		gopt.value[name] = argument
		*opt.pString = argument
		Debug.Printf("handleOption Option: %v\n", gopt.value)
		return nil
	}
	gopt.argsIndex++
	Debug.Printf("len: %d, %d", len(gopt.args), gopt.argsIndex)
	if len(gopt.args) < gopt.argsIndex+1 {
		return fmt.Errorf(ErrorMissingArgument, name)
	}
	// Check if next arg is option
	if optList, _ := isOption(gopt.args[gopt.argsIndex], gopt.mode); len(optList) > 0 {
		return fmt.Errorf(ErrorArgumentWithDash, name)
	}
	gopt.value[name] = gopt.args[gopt.argsIndex]
	*opt.pString = gopt.args[gopt.argsIndex]
	return nil
}

// StringOptional - define a `string` option and its aliases.
//
// StringOptional will set the string to the provided default value when no value is given.
// For example, when called with `--strOpt value`, the value is `value`.
// when called with `--strOpt` the value is the given default.
func (gopt *GetOpt) StringOptional(name string, def string, aliases ...string) *string {
	var s string
	s = def
	gopt.value[name] = s
	aliases = append(aliases, name)
	gopt.failIfDefined(aliases)
	gopt.setOption(name, &option{
		name:    name,
		aliases: aliases,
		def:     def,
		pString: &s,
		handler: gopt.handleStringOptional,
	})
	return &s
}

// StringVarOptional - define a `string` option and its aliases.
// The result will be available through the variable marked by the given pointer.
// StringVarOptional will set the string to the provided default value when no value is given.
// For example, when called with `--strOpt value`, the value is `value`.
// when called with `--strOpt` the value is the given default.
func (gopt *GetOpt) StringVarOptional(p *string, name, def string, aliases ...string) {
	gopt.StringOptional(name, def, aliases...)
	*p = def
	opt := gopt.option(name)
	opt.pString = p
}

func (gopt *GetOpt) handleStringOptional(name string, argument string, usedAlias string) error {
	opt := gopt.option(name)
	opt.setCalled()
	if argument != "" {
		gopt.value[name] = argument
		*opt.pString = argument
		Debug.Printf("handleOption Option: %v\n", gopt.value)
		return nil
	}
	gopt.argsIndex++
	if len(gopt.args) < gopt.argsIndex+1 {
		gopt.value[name] = opt.def
		*opt.pString = opt.def.(string)
		return nil
	}
	// Check if next arg is option
	if optList, _ := isOption(gopt.args[gopt.argsIndex], gopt.mode); len(optList) > 0 {
		gopt.value[name] = opt.def
		*opt.pString = opt.def.(string)
		return nil
	}
	gopt.value[name] = gopt.args[gopt.argsIndex]
	*opt.pString = gopt.args[gopt.argsIndex]
	return nil
}

// Int - define an `int` option and its aliases.
func (gopt *GetOpt) Int(name string, def int, aliases ...string) *int {
	var i int
	i = def
	gopt.value[name] = def
	aliases = append(aliases, name)
	gopt.failIfDefined(aliases)
	gopt.setOption(name, &option{
		name:    name,
		aliases: aliases,
		pInt:    &i,
		handler: gopt.handleInt,
	})
	return &i
}

// IntVar - define an `int` option and its aliases.
// The result will be available through the variable marked by the given pointer.
func (gopt *GetOpt) IntVar(p *int, name string, def int, aliases ...string) {
	gopt.Int(name, def, aliases...)
	*p = def
	opt := gopt.option(name)
	opt.pInt = p
}

func (gopt *GetOpt) handleInt(name string, argument string, usedAlias string) error {
	Debug.Println("handleInt")
	opt := gopt.option(name)
	opt.setCalled()
	if argument != "" {
		iArg, err := strconv.Atoi(argument)
		if err != nil {
			return fmt.Errorf(ErrorConvertToInt, name, argument)
		}
		gopt.value[name] = iArg
		*opt.pInt = iArg
		Debug.Printf("handleOption Option: %v\n", gopt.value)
		return nil
	}
	gopt.argsIndex++
	if len(gopt.args) < gopt.argsIndex+1 {
		return fmt.Errorf(ErrorMissingArgument, name)
	}
	// Check if next arg is option
	if optList, _ := isOption(gopt.args[gopt.argsIndex], gopt.mode); len(optList) > 0 {
		return fmt.Errorf(ErrorArgumentWithDash, name)
	}
	iArg, err := strconv.Atoi(gopt.args[gopt.argsIndex])
	if err != nil {
		return fmt.Errorf(ErrorConvertToInt, name, gopt.args[gopt.argsIndex])
	}
	gopt.value[name] = iArg
	*opt.pInt = iArg
	return nil
}

// IntOptional - define a `int` option and its aliases.
//
// IntOptional will set the int to the provided default value when no value is given.
// For example, when called with `--intOpt 123`, the value is `123`.
// when called with `--intOpt` the value is the given default.
func (gopt *GetOpt) IntOptional(name string, def int, aliases ...string) *int {
	var i int
	i = def
	gopt.value[name] = i
	aliases = append(aliases, name)
	gopt.failIfDefined(aliases)
	gopt.setOption(name, &option{
		name:    name,
		aliases: aliases,
		pInt:    &i,
		def:     def,
		handler: gopt.handleIntOptional,
	})
	return &i
}

// IntVarOptional - define a `int` option and its aliases.
// The result will be available through the variable marked by the given pointer.
// IntOptional will set the int to the provided default value when no value is given.
// For example, when called with `--intOpt 123`, the value is `123`.
// when called with `--intOpt` the value is the given default.
func (gopt *GetOpt) IntVarOptional(p *int, name string, def int, aliases ...string) {
	gopt.IntOptional(name, def, aliases...)
	*p = def
	opt := gopt.option(name)
	opt.pInt = p
}

func (gopt *GetOpt) handleIntOptional(name string, argument string, usedAlias string) error {
	opt := gopt.option(name)
	opt.setCalled()
	if argument != "" {
		iArg, err := strconv.Atoi(argument)
		if err != nil {
			return fmt.Errorf(ErrorConvertToInt, name, argument)
		}
		gopt.value[name] = iArg
		*opt.pInt = iArg
		Debug.Printf("handleOption Option: %v\n", gopt.value)
		return nil
	}
	gopt.argsIndex++
	if len(gopt.args) < gopt.argsIndex+1 {
		gopt.value[name] = opt.def
		*opt.pInt = opt.def.(int)
		return nil
	}
	// Check if next arg is option
	if optList, _ := isOption(gopt.args[gopt.argsIndex], gopt.mode); len(optList) > 0 {
		gopt.value[name] = opt.def
		*opt.pInt = opt.def.(int)
		return nil
	}
	iArg, err := strconv.Atoi(gopt.args[gopt.argsIndex])
	if err != nil {
		return fmt.Errorf(ErrorConvertToInt, name, gopt.args[gopt.argsIndex])
	}
	gopt.value[name] = iArg
	*opt.pInt = iArg
	return nil
}

// Float64 - define an `float64` option and its aliases.
func (gopt *GetOpt) Float64(name string, def float64, aliases ...string) *float64 {
	var i float64
	i = def
	gopt.value[name] = def
	aliases = append(aliases, name)
	gopt.failIfDefined(aliases)
	gopt.setOption(name, &option{
		name:     name,
		aliases:  aliases,
		pFloat64: &i,
		handler:  gopt.handleFloat64,
	})
	return &i
}

// Float64Var - define an `float64` option and its aliases.
// The result will be available through the variable marked by the given pointer.
func (gopt *GetOpt) Float64Var(p *float64, name string, def float64, aliases ...string) {
	gopt.Float64(name, def, aliases...)
	*p = def
	opt := gopt.option(name)
	opt.pFloat64 = p
}

func (gopt *GetOpt) handleFloat64(name string, argument string, usedAlias string) error {
	opt := gopt.option(name)
	opt.setCalled()
	if argument != "" {
		// TODO: Read the different errors when parsing float
		iArg, err := strconv.ParseFloat(argument, 64)
		if err != nil {
			return fmt.Errorf(ErrorConvertToFloat64, name, argument)
		}
		gopt.value[name] = iArg
		*opt.pFloat64 = iArg
		Debug.Printf("handleOption Option: %v\n", gopt.value)
		return nil
	}
	gopt.argsIndex++
	if len(gopt.args) < gopt.argsIndex+1 {
		return fmt.Errorf(ErrorMissingArgument, name)
	}
	// Check if next arg is option
	if optList, _ := isOption(gopt.args[gopt.argsIndex], gopt.mode); len(optList) > 0 {
		return fmt.Errorf(ErrorArgumentWithDash, name)
	}
	iArg, err := strconv.ParseFloat(gopt.args[gopt.argsIndex], 64)
	if err != nil {
		return fmt.Errorf(ErrorConvertToFloat64, name, gopt.args[gopt.argsIndex])
	}
	gopt.value[name] = iArg
	*opt.pFloat64 = iArg
	return nil
}

// StringSlice - define a `[]string` option and its aliases.
//
// StringSlice will accept multiple calls to the same option and append them
// to the `[]string`.
// For example, when called with `--strRpt 1 --strRpt 2`, the value is `[]string{"1", "2"}`.
func (gopt *GetOpt) StringSlice(name string, aliases ...string) *[]string {
	s := []string{}
	gopt.value[name] = s
	aliases = append(aliases, name)
	gopt.failIfDefined(aliases)
	gopt.setOption(name, &option{
		name:     name,
		aliases:  aliases,
		handler:  gopt.handleStringRepeat,
		pStringS: &s,
	})
	return &s
}

func (gopt *GetOpt) handleStringRepeat(name string, argument string, usedAlias string) error {
	opt := gopt.option(name)
	opt.setCalled()
	if argument != "" {
		gopt.value[name] = append(gopt.value[name].([]string), argument)
		*opt.pStringS = append(*opt.pStringS, argument)
		Debug.Printf("handleOption Option: %v\n", gopt.value)
		return nil
	}
	gopt.argsIndex++
	Debug.Printf("len: %d, %d", len(gopt.args), gopt.argsIndex)
	if len(gopt.args) < gopt.argsIndex+1 {
		return fmt.Errorf(ErrorMissingArgument, name)
	}
	// Check if next arg is option
	if optList, _ := isOption(gopt.args[gopt.argsIndex], gopt.mode); len(optList) > 0 {
		return fmt.Errorf(ErrorArgumentWithDash, name)
	}
	gopt.value[name] = append(gopt.value[name].([]string), gopt.args[gopt.argsIndex])
	*opt.pStringS = append(*opt.pStringS, gopt.args[gopt.argsIndex])
	return nil
}

// StringMap - define a `map[string]string` option and its aliases.
//
// StringMap will accept multiple calls of `key=value` type to the same option
// and add them to the `map[string]string` result.
// For example, when called with `--strMap k=v --strMap k2=v2`, the value is
// `map[string]string{"k":"v", "k2": "v2"}`.
func (gopt *GetOpt) StringMap(name string, aliases ...string) map[string]string {
	s := make(map[string]string)
	gopt.value[name] = s
	aliases = append(aliases, name)
	gopt.failIfDefined(aliases)
	gopt.setOption(name, &option{
		name:    name,
		aliases: aliases,
		handler: gopt.handleStringMap,
		stringM: s,
	})
	return s
}

func (gopt *GetOpt) handleStringMap(name string, argument string, usedAlias string) error {
	opt := gopt.option(name)
	opt.setCalled()
	if argument != "" {
		keyValue := strings.Split(argument, "=")
		if len(keyValue) < 2 {
			return fmt.Errorf(ErrorArgumentIsNotKeyValue, name)
		}
		gopt.value[name].(map[string]string)[keyValue[0]] = keyValue[1]
		Debug.Printf("handleOption Option: %v\n", gopt.value)
		return nil
	}
	gopt.argsIndex++
	Debug.Printf("len: %d, %d", len(gopt.args), gopt.argsIndex)
	if len(gopt.args) < gopt.argsIndex+1 {
		return fmt.Errorf(ErrorMissingArgument, name)
	}
	// Check if next arg is option
	if optList, _ := isOption(gopt.args[gopt.argsIndex], gopt.mode); len(optList) > 0 {
		return fmt.Errorf(ErrorArgumentWithDash, name)
	}
	keyValue := strings.Split(gopt.args[gopt.argsIndex], "=")
	if len(keyValue) < 2 {
		return fmt.Errorf(ErrorArgumentIsNotKeyValue, name)
	}
	gopt.value[name].(map[string]string)[keyValue[0]] = keyValue[1]
	Debug.Printf("handleOption Option: %v\n", gopt.value)
	return nil
}

// StringSliceMulti - define a `[]string` option and its aliases.
//
// StringSliceMulti will accept multiple calls to the same option and append them
// to the `[]string`.
// For example, when called with `--strRpt 1 --strRpt 2`, the value is `[]string{"1", "2"}`.
// Addtionally, StringMulti will allow to define a min and max amount of
// arguments to be passed at once.
// For example, when min is 1 and max is 3 and called with `--strRpt 1 2 3`,
// the value is `[]string{"1", "2", "3"}`.
// It could also be called with `--strRpt 1 --strRpt 2 --strRpt 3` for the same result.
// When min is bigger than 1, it is required to pass the amount of arguments defined by min at once.
// For example: with `min = 2`, you at least require `--strRpt 1 2 --strRpt 3`
func (gopt *GetOpt) StringSliceMulti(name string, min, max int, aliases ...string) *[]string {
	s := []string{}
	gopt.value[name] = s
	aliases = append(aliases, name)
	gopt.failIfDefined(aliases)
	gopt.setOption(name, &option{
		name:     name,
		aliases:  aliases,
		handler:  gopt.handleStringSliceMulti,
		pStringS: &s,
		min:      min,
		max:      max,
	})
	if min <= 0 {
		panic(fmt.Sprintf("%s min should be > 0", name))
	}
	if max <= 0 || max < min {
		panic(fmt.Sprintf("%s max should be > 0 and > min", name))
	}
	Debug.Printf("StringMulti return: %v\n", s)
	return &s
}

func (gopt *GetOpt) handleStringSliceMulti(name string, argument string, usedAlias string) error {
	Debug.Printf("handleStringSliceMulti optName: %s, argument %s, usedAlias %s, min %d, max %d\n",
		name, argument, usedAlias, gopt.obj[name].min, gopt.obj[name].max)
	opt := gopt.option(name)
	opt.setCalled()
	argCounter := 0

	if argument != "" {
		gopt.value[name] = append(gopt.value[name].([]string), argument)
		*opt.pStringS = append(*opt.pStringS, argument)
		argCounter++
		Debug.Printf("handleStringSliceMulti internal value: %v\n", gopt.value)
	}
	// Function to handle one arg at a time
	next := func() error {
		Debug.Printf("total arguments: %d, index: %d, counter %d", len(gopt.args), gopt.argsIndex, argCounter)
		if len(gopt.args) <= gopt.argsIndex+1 {
			if argCounter <= opt.min {
				Debug.Printf("ErrorMissingArgument\n")
				return fmt.Errorf(ErrorMissingArgument, name)
			}
			Debug.Printf("return no more arguments\n")
			return fmt.Errorf("NoMoreArguments")
		}
		// Check if next arg is option
		if optList, _ := isOption(gopt.args[gopt.argsIndex+1], gopt.mode); len(optList) > 0 {
			Debug.Printf("Next arg is option: %s\n", gopt.args[gopt.argsIndex+1])
			Debug.Printf("ErrorArgumentWithDash\n")
			return fmt.Errorf(ErrorArgumentWithDash, name)
		}
		gopt.argsIndex++
		gopt.value[name] = append(gopt.value[name].([]string), gopt.args[gopt.argsIndex])
		*opt.pStringS = append(*opt.pStringS, gopt.args[gopt.argsIndex])
		return nil
	}

	// Go through the required and optional iterations
	for argCounter < opt.max {
		argCounter++
		err := next()
		Debug.Printf("counter: %d, value: %v, err %v", argCounter, gopt.value[name], err)
		if err != nil {
			if err.Error() == fmt.Sprintf("NoMoreArguments") {
				Debug.Printf("return value: %v", gopt.value[name])
				return nil
			}
			// always fail if errors under min args
			// After min args, skip missing arg errors
			if argCounter <= opt.min ||
				(err.Error() != fmt.Sprintf(ErrorMissingArgument, name) &&
					err.Error() != fmt.Sprintf(ErrorArgumentWithDash, name)) {
				Debug.Printf("return value: %v, err: %v", gopt.value[name], err)
				return err
			}
			Debug.Printf("return value: %v", gopt.value[name])
			return nil
		}
	}
	Debug.Printf("return value: %v", gopt.value[name])
	return nil
}

// Increment - When called multiple times it increments its the int counter defined by this option.
func (gopt *GetOpt) Increment(name string, def int, aliases ...string) *int {
	var i int
	i = def
	gopt.value[name] = def
	aliases = append(aliases, name)
	gopt.failIfDefined(aliases)
	gopt.setOption(name, &option{
		name:    name,
		aliases: aliases,
		pInt:    &i,
		handler: gopt.handleIncrement,
	})
	return &i
}

// IncrementVar - When called multiple times it increments the provided int.
func (gopt *GetOpt) IncrementVar(p *int, name string, def int, aliases ...string) {
	gopt.Increment(name, def, aliases...)
	*p = def
	opt := gopt.option(name)
	opt.pInt = p
}

func (gopt *GetOpt) handleIncrement(name string, argument string, usedAlias string) error {
	Debug.Println("handleIncrement")
	opt := gopt.option(name)
	opt.setCalled()
	gopt.value[name] = gopt.value[name].(int) + 1
	*opt.pInt = gopt.value[name].(int)
	return nil
}

// func (opt *GetOpt) StringMulti(name string, def []string, min int, max int, aliases ...string) {}
// func (opt *GetOpt) StringMapMulti(name string, def map[string]string, min int, max int, aliases ...string) {}
// func (opt *GetOpt) Procedure(name string, lambda_func int, aliases ...string) {}

// Stringer - print a nice looking representation of the resulting `Option` map.
func (gopt *GetOpt) Stringer() string {
	s := "{\n"
	for name, value := range gopt.value {
		s += fmt.Sprintf("\"%s\":", name)
		switch v := value.(type) {
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
func (gopt *GetOpt) getOptionFromAliases(alias string) (optName string, found bool) {
	found = false
	for name, option := range gopt.obj {
		for _, v := range option.aliases {
			Debug.Printf("Trying to match '%s' against '%s' alias for '%s'\n", alias, v, name)
			if v == alias {
				found = true
				optName = name
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
	Debug.Printf("return: %s, %v\n", optName, found)
	return optName, found
}

var isOptionRegex = regexp.MustCompile(`^(--?)([^=]+)(.*?)$`)
var isOptionRegexEquals = regexp.MustCompile(`^=`)

/*
func isOption - Check if the given string is an option (starts with - or --).
Return the option(s) without the starting dash and an argument if the string contained one.
The behaviour changes depending on the mode: normal, bundling or singleDash.
Also, handle the single dash '-' and double dash '--' especial options.
*/
func isOption(s string, mode string) (options []string, argument string) {
	// Handle especial cases
	if s == "--" {
		return []string{"--"}, ""
	} else if s == "-" {
		return []string{"-"}, ""
	}

	match := isOptionRegex.FindStringSubmatch(s)
	if len(match) > 0 {
		// check long option
		if match[1] == "--" {
			options = []string{match[2]}
			argument = isOptionRegexEquals.ReplaceAllString(match[3], "")
			return
		}
		switch mode {
		case "bundling":
			options = strings.Split(match[2], "")
			argument = isOptionRegexEquals.ReplaceAllString(match[3], "")
		case "singleDash":
			options = []string{strings.Split(match[2], "")[0]}
			argument = strings.Join(strings.Split(match[2], "")[1:], "") + match[3]
		default:
			options = []string{match[2]}
			argument = isOptionRegexEquals.ReplaceAllString(match[3], "")
		}
		return
	}
	return []string{}, ""
}

// Parse - Call the parse method when done describing
func (gopt *GetOpt) Parse(args []string) ([]string, error) {
	gopt.args = args
	Debug.Printf("Parse args: %v(%d)\n", args, len(args))
	var remaining []string
	// opt.argsIndex is the index in the opt.args slice.
	// Option handlers will have to know about it, to ask for the next element.
	for gopt.argsIndex = 0; gopt.argsIndex < len(args); gopt.argsIndex++ {
		arg := args[gopt.argsIndex]
		Debug.Printf("Parse input arg: %s\n", arg)
		if optList, argument := isOption(arg, gopt.mode); len(optList) > 0 {
			Debug.Printf("Parse opt_list: %v, argument: %v\n", optList, argument)
			// Check for termination: '--'
			if optList[0] == "--" {
				Debug.Printf("Parse -- found\n")
				remaining = append(remaining, args[gopt.argsIndex+1:]...)
				Debug.Println(gopt.value)
				Debug.Printf("return %v, %v", remaining, nil)
				return remaining, nil
			}
			Debug.Printf("Parse continue\n")
			for _, optElement := range optList {
				Debug.Printf("Parse optElement: %s\n", optElement)
				if optName, ok := gopt.getOptionFromAliases(optElement); ok {
					Debug.Printf("Parse found opt_list\n")
					handler := gopt.obj[optName].handler
					Debug.Printf("handler found: name %s, argument %s, index %d, list %s\n", optName, argument, gopt.argsIndex, optList[0])
					err := handler(optName, argument, optElement)
					if err != nil {
						Debug.Printf("handler return: value %v, return %v, %v", gopt.value, nil, err)
						return nil, err
					}
				} else {
					Debug.Printf("opt_list not found for '%s'\n", optElement)
					Debug.Println(gopt.value)
					switch gopt.unknownMode {
					case "pass":
						if gopt.requireOrder {
							remaining = append(remaining, args[gopt.argsIndex:]...)
							Debug.Printf("Stop on unknown options %s\n", arg)
							Debug.Println(gopt.value)
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
				remaining = append(remaining, args[gopt.argsIndex:]...)
				Debug.Printf("Stop on non option: %s\n", arg)
				Debug.Println(gopt.value)
				Debug.Printf("return %v, %v", remaining, nil)
				return remaining, nil
			}
			remaining = append(remaining, arg)
		}
	}
	Debug.Println(gopt.value)
	Debug.Printf("return %v, %v", remaining, nil)
	return remaining, nil
}
