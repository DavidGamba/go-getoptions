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
		// Requires `opt.SetUnknownMode("pass")` before the initial `opt.Parse` call.
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

* Options with Key Value arguments.
This allows the same option to be used multiple times with arguments of key value type.
For example: `rpmbuild --define name=myrpm --define version=123`

* Supports passing `--` to stop parsing arguments (everything after will be left in the `remaining []string`).

* Supports command line options with '='.
For example: You can use `--string=mystring` and `--string mystring`.

* Options with optional arguments.
If the default argument is not passed the default is set.

* Allows abbreviations when the provided option is not ambiguous.

* Called method indicates if the option was passed on the command line.

* Errors exposed as variables to allow overriding them for internationalization.

* Multiple ways of managing unknown options:
  - Fail on unknown (default).
  - Warn on unknown.
  - Pass through, allows for subcommands.

* Support for the lonesome dash "-".
To indicate, for example, when to read input from STDIO.

* Incremental options.
Allows the same option to be called multiple times to increment a counter.

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
	value       map[string]interface{} // Map with resulting variables
	mode        string                 // Operation mode for short options: normal, bundling, singleDash
	unknownMode string                 // Unknown option mode
	Writer      io.Writer              // io.Writer locations to write warnings to. Defaults to os.Stderr.
	obj         map[string]option
	args        []string
	argsIndex   int
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
}

// New returns an empty object of type GetOpt.
// This is the starting point when using go-getoptions.
// For example:
//
//   opt := getoptions.New()
func New() *GetOpt {
	opt := &GetOpt{
		value:  make(map[string]interface{}),
		obj:    make(map[string]option),
		Writer: os.Stderr,
	}
	return opt
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
func (opt *GetOpt) failIfDefined(aliases []string) {
	for _, a := range aliases {
		for _, option := range opt.obj {
			for _, v := range option.aliases {
				if v == a {
					panic(fmt.Sprintf("Option/Alias '%s' is already defined", a))
				}
			}
		}
	}
}

// Called - Indicates if the option was passed on the command line.
func (opt *GetOpt) Called(name string) bool {
	return opt.obj[name].called
}

// Option - Returns the value of the given option.
//
// Type assertions are required in cases where the compiler can't determine the type by context.
// For example: `opt.Option("flag").(bool)`.
func (opt *GetOpt) Option(name string) interface{} {
	return opt.value[name]
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
func (opt *GetOpt) SetMode(mode string) {
	opt.mode = mode
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
func (opt *GetOpt) SetUnknownMode(mode string) {
	opt.unknownMode = mode
}

// Bool - define a `bool` option and its aliases.
// It returnns a `*bool` pointing to the variable holding the result.
// If the option is found, the result will be the opposite of the provided default.
func (opt *GetOpt) Bool(name string, def bool, aliases ...string) *bool {
	var b bool
	b = def
	opt.value[name] = def
	aliases = append(aliases, name)
	opt.failIfDefined(aliases)
	opt.obj[name] = option{name: name,
		aliases: aliases,
		pBool:   &b,
		def:     def,
		handler: opt.handleBool}
	return &b
}

// BoolVar - define a `bool` option and its aliases.
// The result will be available through the variable marked by the given pointer.
// If the option is found, the result will be the opposite of the provided default.
func (opt *GetOpt) BoolVar(p *bool, name string, def bool, aliases ...string) {
	opt.Bool(name, def, aliases...)
	*p = def
	var tmp = opt.obj[name]
	tmp.pBool = p
	opt.obj[name] = tmp
}

func (opt *GetOpt) handleBool(optName string, argument string, usedAlias string) error {
	Debug.Println("handleBool")
	opt.value[optName] = !opt.obj[optName].def.(bool)
	*opt.obj[optName].pBool = !opt.obj[optName].def.(bool)
	var tmp = opt.obj[optName]
	tmp.called = true
	opt.obj[optName] = tmp
	return nil
}

// NBool - define a *Negatable* `bool` option and its aliases.
//
// NBool automatically makes aliases with the prefix 'no' and 'no-' to the given name and aliases.
// If the option is found, when the argument is prefixed by 'no' (or by 'no-'), for example '--no-nflag', the value is set to the provided default.
// Otherwise, with a regular call, for example '--nflag', it is set to the opposite of the default.
func (opt *GetOpt) NBool(name string, def bool, aliases ...string) *bool {
	var b bool
	b = def
	opt.value[name] = def
	aliases = append(aliases, name)
	for _, a := range aliases {
		aliases = append(aliases, "no"+a)
		aliases = append(aliases, "no-"+a)
	}
	opt.failIfDefined(aliases)
	opt.obj[name] = option{name: name,
		aliases: aliases,
		pBool:   &b,
		def:     def,
		handler: opt.handleNBool}
	return &b
}

// NBoolVar - define a *Negatable* `bool` option and its aliases.
// The result will be available through the variable marked by the given pointer.
//
// NBoolVar automatically makes aliases with the prefix 'no' and 'no-' to the given name and aliases.
// If the option is found, when the argument is prefixed by 'no' (or by 'no-'), for example '--no-nflag', the value is set to the provided default.
// Otherwise, with a regular call, for example '--nflag', it is set to the opposite of the default.
func (opt *GetOpt) NBoolVar(p *bool, name string, def bool, aliases ...string) {
	opt.NBool(name, def, aliases...)
	*p = def
	var tmp = opt.obj[name]
	tmp.pBool = p
	opt.obj[name] = tmp
}

func (opt *GetOpt) handleNBool(optName string, argument string, usedAlias string) error {
	Debug.Println("handleNBool")
	if strings.HasPrefix(usedAlias, "no-") {
		opt.value[optName] = opt.obj[optName].def.(bool)
		*opt.obj[optName].pBool = opt.obj[optName].def.(bool)
	} else {
		opt.value[optName] = !opt.obj[optName].def.(bool)
		*opt.obj[optName].pBool = !opt.obj[optName].def.(bool)
	}
	var tmp = opt.obj[optName]
	tmp.called = true
	opt.obj[optName] = tmp
	return nil
}

// String - define a `string` option and its aliases.
// If not called, the return value will be that of the given default `def`.
func (opt *GetOpt) String(name, def string, aliases ...string) *string {
	var s string
	s = def
	opt.value[name] = s
	aliases = append(aliases, name)
	opt.failIfDefined(aliases)
	opt.obj[name] = option{
		name:    name,
		aliases: aliases,
		pString: &s,
		handler: opt.handleString,
	}
	return &s
}

// StringVar - define a `string` option and its aliases.
// The result will be available through the variable marked by the given pointer.
// If not called, the return value will be that of the given default `def`.
func (opt *GetOpt) StringVar(p *string, name, def string, aliases ...string) {
	opt.String(name, def, aliases...)
	*p = def
	var tmp = opt.obj[name]
	tmp.pString = p
	opt.obj[name] = tmp
}

func (opt *GetOpt) handleString(optName string, argument string, usedAlias string) error {
	Debug.Printf("handleString opt.args: %v(%d)\n", opt.args, len(opt.args))
	var tmp = opt.obj[optName]
	tmp.called = true
	opt.obj[optName] = tmp
	if argument != "" {
		opt.value[optName] = argument
		*opt.obj[optName].pString = argument
		Debug.Printf("handleOption Option: %v\n", opt.value)
		return nil
	}
	opt.argsIndex++
	Debug.Printf("len: %d, %d", len(opt.args), opt.argsIndex)
	if len(opt.args) < opt.argsIndex+1 {
		return fmt.Errorf(ErrorMissingArgument, optName)
	}
	// Check if next arg is option
	if optList, _ := isOption(opt.args[opt.argsIndex], opt.mode); len(optList) > 0 {
		return fmt.Errorf(ErrorArgumentWithDash, optName)
	}
	opt.value[optName] = opt.args[opt.argsIndex]
	*opt.obj[optName].pString = opt.args[opt.argsIndex]
	return nil
}

// StringOptional - define a `string` option and its aliases.
//
// StringOptional will set the string to the provided default value when no value is given.
// For example, when called with `--strOpt value`, the value is `value`.
// when called with `--strOpt` the value is the given default.
func (opt *GetOpt) StringOptional(name string, def string, aliases ...string) *string {
	var s string
	s = def
	opt.value[name] = s
	aliases = append(aliases, name)
	opt.failIfDefined(aliases)
	opt.obj[name] = option{name: name,
		aliases: aliases,
		def:     def,
		pString: &s,
		handler: opt.handleStringOptional,
	}
	return &s
}

// StringVarOptional - define a `string` option and its aliases.
// The result will be available through the variable marked by the given pointer.
// StringVarOptional will set the string to the provided default value when no value is given.
// For example, when called with `--strOpt value`, the value is `value`.
// when called with `--strOpt` the value is the given default.
func (opt *GetOpt) StringVarOptional(p *string, name, def string, aliases ...string) {
	opt.StringOptional(name, def, aliases...)
	*p = def
	var tmp = opt.obj[name]
	tmp.pString = p
	opt.obj[name] = tmp
}

func (opt *GetOpt) handleStringOptional(optName string, argument string, usedAlias string) error {
	var tmp = opt.obj[optName]
	tmp.called = true
	opt.obj[optName] = tmp
	if argument != "" {
		opt.value[optName] = argument
		*opt.obj[optName].pString = argument
		Debug.Printf("handleOption Option: %v\n", opt.value)
		return nil
	}
	opt.argsIndex++
	if len(opt.args) < opt.argsIndex+1 {
		opt.value[optName] = opt.obj[optName].def
		*opt.obj[optName].pString = opt.obj[optName].def.(string)
		return nil
	}
	// Check if next arg is option
	if optList, _ := isOption(opt.args[opt.argsIndex], opt.mode); len(optList) > 0 {
		opt.value[optName] = opt.obj[optName].def
		*opt.obj[optName].pString = opt.obj[optName].def.(string)
		return nil
	}
	opt.value[optName] = opt.args[opt.argsIndex]
	*opt.obj[optName].pString = opt.args[opt.argsIndex]
	return nil
}

// Int - define an `int` option and its aliases.
func (opt *GetOpt) Int(name string, def int, aliases ...string) *int {
	var i int
	i = def
	opt.value[name] = def
	aliases = append(aliases, name)
	opt.failIfDefined(aliases)
	opt.obj[name] = option{name: name,
		aliases: aliases,
		pInt:    &i,
		handler: opt.handleInt,
	}
	return &i
}

// IntVar - define an `int` option and its aliases.
// The result will be available through the variable marked by the given pointer.
func (opt *GetOpt) IntVar(p *int, name string, def int, aliases ...string) {
	opt.Int(name, def, aliases...)
	*p = def
	var tmp = opt.obj[name]
	tmp.pInt = p
	opt.obj[name] = tmp
}

func (opt *GetOpt) handleInt(optName string, argument string, usedAlias string) error {
	Debug.Println("handleInt")
	var tmp = opt.obj[optName]
	tmp.called = true
	opt.obj[optName] = tmp
	if argument != "" {
		iArg, err := strconv.Atoi(argument)
		if err != nil {
			return fmt.Errorf(ErrorConvertToInt, optName, argument)
		}
		opt.value[optName] = iArg
		*opt.obj[optName].pInt = iArg
		Debug.Printf("handleOption Option: %v\n", opt.value)
		return nil
	}
	opt.argsIndex++
	if len(opt.args) < opt.argsIndex+1 {
		return fmt.Errorf(ErrorMissingArgument, optName)
	}
	// Check if next arg is option
	if optList, _ := isOption(opt.args[opt.argsIndex], opt.mode); len(optList) > 0 {
		return fmt.Errorf(ErrorArgumentWithDash, optName)
	}
	iArg, err := strconv.Atoi(opt.args[opt.argsIndex])
	if err != nil {
		return fmt.Errorf(ErrorConvertToInt, optName, opt.args[opt.argsIndex])
	}
	opt.value[optName] = iArg
	*opt.obj[optName].pInt = iArg
	return nil
}

// IntOptional - define a `int` option and its aliases.
//
// IntOptional will set the int to the provided default value when no value is given.
// For example, when called with `--intOpt 123`, the value is `123`.
// when called with `--intOpt` the value is the given default.
func (opt *GetOpt) IntOptional(name string, def int, aliases ...string) *int {
	var i int
	i = def
	opt.value[name] = i
	aliases = append(aliases, name)
	opt.failIfDefined(aliases)
	opt.obj[name] = option{name: name,
		aliases: aliases,
		pInt:    &i,
		def:     def,
		handler: opt.handleIntOptional,
	}
	return &i
}

// IntVarOptional - define a `int` option and its aliases.
// The result will be available through the variable marked by the given pointer.
// IntOptional will set the int to the provided default value when no value is given.
// For example, when called with `--intOpt 123`, the value is `123`.
// when called with `--intOpt` the value is the given default.
func (opt *GetOpt) IntVarOptional(p *int, name string, def int, aliases ...string) {
	opt.IntOptional(name, def, aliases...)
	*p = def
	var tmp = opt.obj[name]
	tmp.pInt = p
	opt.obj[name] = tmp
}

func (opt *GetOpt) handleIntOptional(optName string, argument string, usedAlias string) error {
	var tmp = opt.obj[optName]
	tmp.called = true
	opt.obj[optName] = tmp
	if argument != "" {
		iArg, err := strconv.Atoi(argument)
		if err != nil {
			return fmt.Errorf(ErrorConvertToInt, optName, argument)
		}
		opt.value[optName] = iArg
		*opt.obj[optName].pInt = iArg
		Debug.Printf("handleOption Option: %v\n", opt.value)
		return nil
	}
	opt.argsIndex++
	if len(opt.args) < opt.argsIndex+1 {
		opt.value[optName] = opt.obj[optName].def
		*opt.obj[optName].pInt = opt.obj[optName].def.(int)
		return nil
	}
	// Check if next arg is option
	if optList, _ := isOption(opt.args[opt.argsIndex], opt.mode); len(optList) > 0 {
		opt.value[optName] = opt.obj[optName].def
		*opt.obj[optName].pInt = opt.obj[optName].def.(int)
		return nil
	}
	iArg, err := strconv.Atoi(opt.args[opt.argsIndex])
	if err != nil {
		return fmt.Errorf(ErrorConvertToInt, optName, opt.args[opt.argsIndex])
	}
	opt.value[optName] = iArg
	*opt.obj[optName].pInt = iArg
	return nil
}

// Float64 - define an `float64` option and its aliases.
func (opt *GetOpt) Float64(name string, def float64, aliases ...string) *float64 {
	var i float64
	i = def
	opt.value[name] = def
	aliases = append(aliases, name)
	opt.failIfDefined(aliases)
	opt.obj[name] = option{name: name,
		aliases:  aliases,
		pFloat64: &i,
		handler:  opt.handleFloat64,
	}
	return &i
}

// Float64Var - define an `float64` option and its aliases.
// The result will be available through the variable marked by the given pointer.
func (opt *GetOpt) Float64Var(p *float64, name string, def float64, aliases ...string) {
	opt.Float64(name, def, aliases...)
	*p = def
	var tmp = opt.obj[name]
	tmp.pFloat64 = p
	opt.obj[name] = tmp
}

func (opt *GetOpt) handleFloat64(optName string, argument string, usedAlias string) error {
	var tmp = opt.obj[optName]
	tmp.called = true
	opt.obj[optName] = tmp
	if argument != "" {
		// TODO: Read the different errors when parsing float
		iArg, err := strconv.ParseFloat(argument, 64)
		if err != nil {
			return fmt.Errorf(ErrorConvertToFloat64, optName, argument)
		}
		opt.value[optName] = iArg
		*opt.obj[optName].pFloat64 = iArg
		Debug.Printf("handleOption Option: %v\n", opt.value)
		return nil
	}
	opt.argsIndex++
	if len(opt.args) < opt.argsIndex+1 {
		return fmt.Errorf(ErrorMissingArgument, optName)
	}
	// Check if next arg is option
	if optList, _ := isOption(opt.args[opt.argsIndex], opt.mode); len(optList) > 0 {
		return fmt.Errorf(ErrorArgumentWithDash, optName)
	}
	iArg, err := strconv.ParseFloat(opt.args[opt.argsIndex], 64)
	if err != nil {
		return fmt.Errorf(ErrorConvertToFloat64, optName, opt.args[opt.argsIndex])
	}
	opt.value[optName] = iArg
	*opt.obj[optName].pFloat64 = iArg
	return nil
}

// StringSlice - define a `[]string` option and its aliases.
//
// StringSlice will accept multiple calls to the same option and append them
// to the `[]string`.
// For example, when called with `--strRpt 1 --strRpt 2`, the value is `[]string{"1", "2"}`.
func (opt *GetOpt) StringSlice(name string, aliases ...string) *[]string {
	s := []string{}
	opt.value[name] = s
	aliases = append(aliases, name)
	opt.failIfDefined(aliases)
	opt.obj[name] = option{
		name:     name,
		aliases:  aliases,
		handler:  opt.handleStringRepeat,
		pStringS: &s,
	}
	return &s
}

func (opt *GetOpt) handleStringRepeat(optName string, argument string, usedAlias string) error {
	var tmp = opt.obj[optName]
	tmp.called = true
	opt.obj[optName] = tmp
	if argument != "" {
		opt.value[optName] = append(opt.value[optName].([]string), argument)
		*opt.obj[optName].pStringS = append(*opt.obj[optName].pStringS, argument)
		Debug.Printf("handleOption Option: %v\n", opt.value)
		return nil
	}
	opt.argsIndex++
	Debug.Printf("len: %d, %d", len(opt.args), opt.argsIndex)
	if len(opt.args) < opt.argsIndex+1 {
		return fmt.Errorf(ErrorMissingArgument, optName)
	}
	opt.value[optName] = append(opt.value[optName].([]string), opt.args[opt.argsIndex])
	*opt.obj[optName].pStringS = append(*opt.obj[optName].pStringS, opt.args[opt.argsIndex])
	return nil
}

// StringMap - define a `map[string]string` option and its aliases.
//
// StringMap will accept multiple calls of `key=value` type to the same option
// and add them to the `map[string]string` result.
// For example, when called with `--strMap k=v --strMap k2=v2`, the value is
// `map[string]string{"k":"v", "k2": "v2"}`.
func (opt *GetOpt) StringMap(name string, aliases ...string) map[string]string {
	s := make(map[string]string)
	opt.value[name] = s
	aliases = append(aliases, name)
	opt.failIfDefined(aliases)
	opt.obj[name] = option{
		name:    name,
		aliases: aliases,
		handler: opt.handleStringMap,
		stringM: s,
	}
	return s
}

func (opt *GetOpt) handleStringMap(optName string, argument string, usedAlias string) error {
	tmp := opt.obj[optName]
	tmp.called = true
	opt.obj[optName] = tmp
	if argument != "" {
		keyValue := strings.Split(argument, "=")
		if len(keyValue) < 2 {
			return fmt.Errorf(ErrorArgumentIsNotKeyValue, optName)
		}
		opt.value[optName].(map[string]string)[keyValue[0]] = keyValue[1]
		Debug.Printf("handleOption Option: %v\n", opt.value)
		return nil
	}
	opt.argsIndex++
	Debug.Printf("len: %d, %d", len(opt.args), opt.argsIndex)
	if len(opt.args) < opt.argsIndex+1 {
		return fmt.Errorf(ErrorMissingArgument, optName)
	}
	keyValue := strings.Split(opt.args[opt.argsIndex], "=")
	if len(keyValue) < 2 {
		return fmt.Errorf(ErrorArgumentIsNotKeyValue, optName)
	}
	opt.value[optName].(map[string]string)[keyValue[0]] = keyValue[1]
	Debug.Printf("handleOption Option: %v\n", opt.value)
	return nil
}

// Increment - When called multiple times it increments its the int counter defined by this option.
func (opt *GetOpt) Increment(name string, def int, aliases ...string) *int {
	var i int
	i = def
	opt.value[name] = def
	aliases = append(aliases, name)
	opt.failIfDefined(aliases)
	opt.obj[name] = option{name: name,
		aliases: aliases,
		pInt:    &i,
		handler: opt.handleIncrement,
	}
	return &i
}

// IncrementVar - When called multiple times it increments the provided int.
func (opt *GetOpt) IncrementVar(p *int, name string, def int, aliases ...string) {
	opt.Increment(name, def, aliases...)
	*p = def
	var tmp = opt.obj[name]
	tmp.pInt = p
	opt.obj[name] = tmp
}

func (opt *GetOpt) handleIncrement(optName string, argument string, usedAlias string) error {
	Debug.Println("handleIncrement")
	var tmp = opt.obj[optName]
	tmp.called = true
	opt.obj[optName] = tmp
	opt.value[optName] = opt.value[optName].(int) + 1
	*opt.obj[optName].pInt = opt.value[optName].(int)
	return nil
}

// func (opt *GetOpt) StringMulti(name string, def []string, min int, max int, aliases ...string) {}
// func (opt *GetOpt) StringMapMulti(name string, def map[string]string, min int, max int, aliases ...string) {}
// func (opt *GetOpt) Procedure(name string, lambda_func int, aliases ...string) {}

// Stringer - print a nice looking representation of the resulting `Option` map.
func (opt *GetOpt) Stringer() string {
	s := "{\n"
	for name, value := range opt.value {
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
func (opt *GetOpt) getOptionFromAliases(alias string) (optName string, found bool) {
	found = false
	for name, option := range opt.obj {
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
		for name, option := range opt.obj {
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
func (opt *GetOpt) Parse(args []string) ([]string, error) {
	opt.args = args
	Debug.Printf("Parse args: %v(%d)\n", args, len(args))
	var remaining []string
	// opt.argsIndex is the index in the opt.args slice.
	// Option handlers will have to know about it, to ask for the next element.
	for opt.argsIndex = 0; opt.argsIndex < len(args); opt.argsIndex++ {
		arg := args[opt.argsIndex]
		Debug.Printf("Parse input arg: %s\n", arg)
		if optList, argument := isOption(arg, opt.mode); len(optList) > 0 {
			Debug.Printf("Parse opt_list: %v, argument: %v\n", optList, argument)
			// Check for termination: '--'
			if optList[0] == "--" {
				Debug.Printf("Parse -- found\n")
				remaining = append(remaining, args[opt.argsIndex+1:]...)
				Debug.Println(opt.value)
				Debug.Printf("return %v, %v", remaining, nil)
				return remaining, nil
			}
			Debug.Printf("Parse continue\n")
			for _, optElement := range optList {
				if optName, ok := opt.getOptionFromAliases(optElement); ok {
					Debug.Printf("Parse found opt_list\n")
					handler := opt.obj[optName].handler
					Debug.Printf("handler found: %s, %s, %d, %s\n", optName, argument, opt.argsIndex, optList[0])
					err := handler(optName, argument, optElement)
					if err != nil {
						Debug.Println(opt.value)
						Debug.Printf("return %v, %v", nil, err)
						return nil, err
					}
				} else {
					Debug.Printf("opt_list not found for '%s'\n", optElement)
					Debug.Println(opt.value)
					switch opt.unknownMode {
					case "pass":
						remaining = append(remaining, arg)
						break
					case "warn":
						fmt.Fprintf(opt.Writer, MessageOnUnknown, optElement)
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
			remaining = append(remaining, arg)
		}
	}
	Debug.Println(opt.value)
	Debug.Printf("return %v, %v", remaining, nil)
	return remaining, nil
}
