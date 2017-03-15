// This file is part of go-getoptions.
//
// Copyright (C) 2015-2017  David Gamba Rios
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

• Allow passing options and non-options in any order.

• Support for `--long` options.

• Support for short (`-s`) options with flexible behaviour (see https://github.com/DavidGamba/go-getoptions#operation_modes for details):

 - Normal (default)
 - Bundling
 - SingleDash

• Boolean, String, Int and Float64 type options.

• Multiple aliases for the same option. e.g. `help`, `man`.

• Negatable Boolean options.
For example: `--verbose`, `--no-verbose` or `--noverbose`.

• Options with Array arguments.
The same option can be used multiple times with different arguments.
The list of arguments will be saved into an Array like structure inside the program.

• Options with array arguments and multiple entries.

• When using integer array options with multiple arguments, positive integer ranges are allowed.
For example: `1..3` to indicate `1 2 3`.

• Options with key value arguments and multiple entries.

• Options with Key Value arguments.
This allows the same option to be used multiple times with arguments of key value type.
For example: `rpmbuild --define name=myrpm --define version=123`.

• Supports passing `--` to stop parsing arguments (everything after will be left in the `remaining []string`).

• Supports subcommands (stop parsing arguments when non option is passed).

• Supports command line options with '='.
For example: You can use `--string=mystring` and `--string mystring`.

• Allows passing arguments to options that start with dash `-` when passed after equal.
For example: `--string=--hello` and `--int=-123`.

• Options with optional arguments.
If the default argument is not passed the default is set.

• Allows abbreviations when the provided option is not ambiguous.

• Called method indicates if the option was passed on the command line.

• Errors exposed as public variables to allow overriding them for internationalization.

• Multiple ways of managing unknown options:
  - Fail on unknown (default).
  - Warn on unknown.
  - Pass through, allows for subcommands and can be combined with Require Order.

• Require order: Allows for subcommands. Stop parsing arguments when the first non-option is found.
When mixed with Pass through, it also stops parsing arguments when the first unmatched option is found.

• Support for the lonesome dash "-".
To indicate, for example, when to read input from STDIO.

• Incremental options.
Allows the same option to be called multiple times to increment a counter.

• Supports case sensitive options.
For example, you can use `v` to define `verbose` and `V` to define `Version`.

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
	"regexp"
	"strconv"
	"strings"
)

// Debug Logger instance set to `ioutil.Discard` by default.
// Enable debug logging by setting: `Debug.SetOutput(os.Stderr)`.
var Debug = log.New(ioutil.Discard, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)

// GetOpt - main object.
type GetOpt struct {
	mode         string    // Operation mode for short options: normal, bundling, singleDash
	unknownMode  string    // Unknown option mode
	requireOrder bool      // Stop parsing on non option
	Writer       io.Writer // io.Writer locations to write warnings to. Defaults to os.Stderr.
	obj          map[string]*option
	args         *argList
}

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

// Bool - define a `bool` option and its aliases.
// It returnns a `*bool` pointing to the variable holding the result.
// If the option is found, the result will be the opposite of the provided default.
func (gopt *GetOpt) Bool(name string, def bool, aliases ...string) *bool {
	aliases = append(aliases, name)
	gopt.failIfDefined(aliases)
	gopt.setOption(name, newOption(name, aliases))
	gopt.option(name).setBoolPtr(&def)
	gopt.option(name).setHandler(gopt.handleBool)
	return &def
}

// BoolVar - define a `bool` option and its aliases.
// The result will be available through the variable marked by the given pointer.
// If the option is found, the result will be the opposite of the provided default.
func (gopt *GetOpt) BoolVar(p *bool, name string, def bool, aliases ...string) {
	gopt.Bool(name, def, aliases...)
	*p = def
	gopt.option(name).setBoolPtr(p)
}

func (gopt *GetOpt) handleBool(name string, argument string, usedAlias string) error {
	Debug.Println("handleBool")
	opt := gopt.option(name)
	opt.setCalled()
	opt.setBool(!opt.getBool())
	return nil
}

// NBool - define a *Negatable* `bool` option and its aliases.
//
// NBool automatically makes aliases with the prefix 'no' and 'no-' to the given name and aliases.
// If the option is found, when the argument is prefixed by 'no' (or by 'no-'), for example '--no-nflag', the value is set to the provided default.
// Otherwise, with a regular call, for example '--nflag', it is set to the opposite of the default.
func (gopt *GetOpt) NBool(name string, def bool, aliases ...string) *bool {
	aliases = append(aliases, name)
	for _, a := range aliases {
		aliases = append(aliases, "no"+a)
		aliases = append(aliases, "no-"+a)
	}
	gopt.failIfDefined(aliases)
	gopt.setOption(name, newOption(name, aliases))
	gopt.option(name).setBoolPtr(&def)
	gopt.option(name).setHandler(gopt.handleNBool)
	return &def
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
	gopt.option(name).setBoolPtr(p)
}

func (gopt *GetOpt) handleNBool(name string, argument string, usedAlias string) error {
	Debug.Println("handleNBool")
	opt := gopt.option(name)
	opt.setCalled()
	if !strings.HasPrefix(usedAlias, "no-") {
		opt.setBool(!opt.getBool())
	}
	return nil
}

func (gopt *GetOpt) handleSingleOption(name string, argument string, usedAlias string) error {
	opt := gopt.option(name)
	opt.setCalled()
	if argument != "" {
		return opt.save(name, argument)
	}
	if !gopt.args.existsNext() {
		if opt.isOptional() {
			return nil
		}
		return fmt.Errorf(ErrorMissingArgument, name)
	}
	// Check if next arg is option
	if optList, _ := isOption(gopt.args.peekNextValue(), gopt.mode); len(optList) > 0 {
		if opt.isOptional() {
			return nil
		}
		return fmt.Errorf(ErrorArgumentWithDash, name)
	}
	gopt.args.next()
	return opt.save(name, gopt.args.value())
}

// String - define a `string` option and its aliases.
// If not called, the return value will be that of the given default `def`.
func (gopt *GetOpt) String(name, def string, aliases ...string) *string {
	aliases = append(aliases, name)
	gopt.failIfDefined(aliases)
	gopt.setOption(name, newOption(name, aliases))
	gopt.option(name).setStringPtr(&def)
	gopt.option(name).setHandler(gopt.handleSingleOption)
	return &def
}

// StringVar - define a `string` option and its aliases.
// The result will be available through the variable marked by the given pointer.
// If not called, the return value will be that of the given default `def`.
func (gopt *GetOpt) StringVar(p *string, name, def string, aliases ...string) {
	gopt.String(name, def, aliases...)
	*p = def
	gopt.option(name).setStringPtr(p)
}

// StringOptional - define a `string` option and its aliases.
//
// StringOptional will set the string to the provided default value when no value is given.
// For example, when called with `--strOpt value`, the value is `value`.
// when called with `--strOpt` the value is the given default.
func (gopt *GetOpt) StringOptional(name string, def string, aliases ...string) *string {
	aliases = append(aliases, name)
	gopt.failIfDefined(aliases)
	gopt.setOption(name, newOption(name, aliases))
	gopt.option(name).setStringPtr(&def)
	gopt.option(name).setIsOptional()
	gopt.option(name).setHandler(gopt.handleSingleOption)
	return &def
}

// StringVarOptional - define a `string` option and its aliases.
// The result will be available through the variable marked by the given pointer.
//
// StringVarOptional will set the string to the provided default value when no value is given.
// For example, when called with `--strOpt value`, the value is `value`.
// when called with `--strOpt` the value is the given default.
func (gopt *GetOpt) StringVarOptional(p *string, name, def string, aliases ...string) {
	gopt.StringOptional(name, def, aliases...)
	*p = def
	gopt.option(name).setStringPtr(p)
}

// Int - define an `int` option and its aliases.
func (gopt *GetOpt) Int(name string, def int, aliases ...string) *int {
	aliases = append(aliases, name)
	gopt.failIfDefined(aliases)
	gopt.setOption(name, newOption(name, aliases))
	gopt.option(name).setIntPtr(&def)
	gopt.option(name).setHandler(gopt.handleSingleOption)
	gopt.option(name).optType = intType
	return &def
}

// IntVar - define an `int` option and its aliases.
// The result will be available through the variable marked by the given pointer.
func (gopt *GetOpt) IntVar(p *int, name string, def int, aliases ...string) {
	gopt.Int(name, def, aliases...)
	*p = def
	gopt.option(name).setIntPtr(p)
}

// IntOptional - define a `int` option and its aliases.
//
// IntOptional will set the int to the provided default value when no value is given.
// For example, when called with `--intOpt 123`, the value is `123`.
// when called with `--intOpt` the value is the given default.
func (gopt *GetOpt) IntOptional(name string, def int, aliases ...string) *int {
	aliases = append(aliases, name)
	gopt.failIfDefined(aliases)
	gopt.setOption(name, newOption(name, aliases))
	gopt.option(name).setIntPtr(&def)
	gopt.option(name).setIsOptional()
	gopt.option(name).setHandler(gopt.handleSingleOption)
	gopt.option(name).optType = intType
	return &def
}

// IntVarOptional - define a `int` option and its aliases.
// The result will be available through the variable marked by the given pointer.
//
// IntOptional will set the int to the provided default value when no value is given.
// For example, when called with `--intOpt 123`, the value is `123`.
// when called with `--intOpt` the value is the given default.
func (gopt *GetOpt) IntVarOptional(p *int, name string, def int, aliases ...string) {
	gopt.IntOptional(name, def, aliases...)
	*p = def
	gopt.option(name).setIntPtr(p)
}

// Float64 - define an `float64` option and its aliases.
func (gopt *GetOpt) Float64(name string, def float64, aliases ...string) *float64 {
	aliases = append(aliases, name)
	gopt.failIfDefined(aliases)
	gopt.setOption(name, newOption(name, aliases))
	gopt.option(name).setFloat64Ptr(&def)
	gopt.option(name).setHandler(gopt.handleSingleOption)
	gopt.option(name).optType = float64Type
	return &def
}

// Float64Var - define an `float64` option and its aliases.
// The result will be available through the variable marked by the given pointer.
func (gopt *GetOpt) Float64Var(p *float64, name string, def float64, aliases ...string) {
	gopt.Float64(name, def, aliases...)
	*p = def
	gopt.option(name).setFloat64Ptr(p)
}

// StringSlice - define a `[]string` option and its aliases.
//
// StringSlice will accept multiple calls to the same option and append them
// to the `[]string`.
// For example, when called with `--strRpt 1 --strRpt 2`, the value is `[]string{"1", "2"}`.
func (gopt *GetOpt) StringSlice(name string, aliases ...string) *[]string {
	s := []string{}
	aliases = append(aliases, name)
	gopt.failIfDefined(aliases)
	gopt.setOption(name, newOption(name, aliases))
	gopt.option(name).setStringSlicePtr(&s)
	gopt.option(name).setHandler(gopt.handleSingleOption)
	gopt.option(name).optType = stringRepeatType
	return &s
}

// StringMap - define a `map[string]string` option and its aliases.
//
// StringMap will accept multiple calls of `key=value` type to the same option
// and add them to the `map[string]string` result.
// For example, when called with `--strMap k=v --strMap k2=v2`, the value is
// `map[string]string{"k":"v", "k2": "v2"}`.
func (gopt *GetOpt) StringMap(name string, aliases ...string) map[string]string {
	s := make(map[string]string)
	aliases = append(aliases, name)
	gopt.failIfDefined(aliases)
	gopt.setOption(name, newOption(name, aliases))
	gopt.option(name).setStringMap(s)
	gopt.option(name).setHandler(gopt.handleSingleOption)
	gopt.option(name).optType = stringMapType
	return s
}

// StringSliceMulti - define a `[]string` option and its aliases.
//
// StringSliceMulti will accept multiple calls to the same option and append them
// to the `[]string`.
// For example, when called with `--strRpt 1 --strRpt 2`, the value is `[]string{"1", "2"}`.
//
// Addtionally, StringSliceMulti will allow to define a min and max amount of
// arguments to be passed at once.
// For example, when min is 1 and max is 3 and called with `--strRpt 1 2 3`,
// the value is `[]string{"1", "2", "3"}`.
// It could also be called with `--strRpt 1 --strRpt 2 --strRpt 3` for the same result.
//
// When min is bigger than 1, it is required to pass the amount of arguments defined by min at once.
// For example: with `min = 2`, you at least require `--strRpt 1 2 --strRpt 3`
func (gopt *GetOpt) StringSliceMulti(name string, min, max int, aliases ...string) *[]string {
	s := []string{}
	aliases = append(aliases, name)
	gopt.failIfDefined(aliases)
	gopt.setOption(name, newOption(name, aliases))
	gopt.option(name).setStringSlicePtr(&s)
	gopt.option(name).setHandler(gopt.handleSliceMultiOption)
	gopt.option(name).setMin(min)
	gopt.option(name).setMax(max)
	gopt.option(name).optType = stringRepeatType
	if min <= 0 {
		panic(fmt.Sprintf("%s min should be > 0", name))
	}
	if max <= 0 || max < min {
		panic(fmt.Sprintf("%s max should be > 0 and > min", name))
	}
	Debug.Printf("StringMulti return: %v\n", s)
	return &s
}

// IntSliceMulti - define a `[]int` option and its aliases.
//
// IntSliceMulti will accept multiple calls to the same option and append them
// to the `[]int`.
// For example, when called with `--intRpt 1 --intRpt 2`, the value is `[]int{1, 2}`.
//
// Addtionally, IntSliceMulti will allow to define a min and max amount of
// arguments to be passed at once.
// For example, when min is 1 and max is 3 and called with `--strRpt 1 2 3`,
// the value is `[]int{1, 2, 3}`.
// It could also be called with `--strRpt 1 --strRpt 2 --strRpt 3` for the same result.
//
// When min is bigger than 1, it is required to pass the amount of arguments defined by min at once.
// For example: with `min = 2`, you at least require `--strRpt 1 2 --strRpt 3`
//
// Finally, possitive integer ranges are allowed.
// For example, Instead of writting: `csv --columns 1 2 3` or
// `csv --columns 1 --columns 2 --columns 3`
// The input could be: `csv --columns 1..3`.
func (gopt *GetOpt) IntSliceMulti(name string, min, max int, aliases ...string) *[]int {
	s := []int{}
	aliases = append(aliases, name)
	gopt.failIfDefined(aliases)
	gopt.setOption(name, newOption(name, aliases))
	gopt.option(name).setIntSlicePtr(&s)
	gopt.option(name).setHandler(gopt.handleSliceMultiOption)
	gopt.option(name).setMin(min)
	gopt.option(name).setMax(max)
	gopt.option(name).optType = intRepeatType
	if min <= 0 {
		panic(fmt.Sprintf("%s min should be > 0", name))
	}
	if max <= 0 || max < min {
		panic(fmt.Sprintf("%s max should be > 0 and > min", name))
	}
	Debug.Printf("IntMulti return: %v\n", s)
	return &s
}

// StringMapMulti - define a `map[string]string` option and its aliases.
//
// StringMapMulti will accept multiple calls of `key=value` type to the same option
// and add them to the `map[string]string` result.
// For example, when called with `--strMap k=v --strMap k2=v2`, the value is
// `map[string]string{"k":"v", "k2": "v2"}`.
//
// Addtionally, StringMapMulti will allow to define a min and max amount of
// arguments to be passed at once.
// For example, when min is 1 and max is 3 and called with `--strMap k=v k2=v2 k3=v3`,
// the value is `map[string]string{"k":"v", "k2": "v2", "k3": "v3"}`.
// It could also be called with `--strMap k=v --strMap k2=v2 --strMap k3=v3` for the same result.
//
// When min is bigger than 1, it is required to pass the amount of arguments defined by min at once.
// For example: with `min = 2`, you at least require `--strMap k=v k2=v2 --strMap k3=v3`
func (gopt *GetOpt) StringMapMulti(name string, min, max int, aliases ...string) map[string]string {
	s := make(map[string]string)
	aliases = append(aliases, name)
	gopt.failIfDefined(aliases)
	gopt.setOption(name, newOption(name, aliases))
	gopt.option(name).setStringMap(s)
	gopt.option(name).setHandler(gopt.handleSliceMultiOption)
	gopt.option(name).setMin(min)
	gopt.option(name).setMax(max)
	gopt.option(name).optType = stringMapType
	if min <= 0 {
		panic(fmt.Sprintf("%s min should be > 0", name))
	}
	if max <= 0 || max < min {
		panic(fmt.Sprintf("%s max should be > 0 and > min", name))
	}
	Debug.Printf("StringMulti return: %v\n", s)
	return s
}

func (gopt *GetOpt) handleSliceMultiOption(name string, argument string, usedAlias string) error {
	Debug.Printf("handleStringSliceMulti\n")
	opt := gopt.option(name)
	opt.setCalled()
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
func (gopt *GetOpt) Increment(name string, def int, aliases ...string) *int {
	aliases = append(aliases, name)
	gopt.failIfDefined(aliases)
	gopt.setOption(name, newOption(name, aliases))
	gopt.option(name).setIntPtr(&def)
	gopt.option(name).setHandler(gopt.handleIncrement)
	return &def
}

// IncrementVar - When called multiple times it increments the provided int.
func (gopt *GetOpt) IncrementVar(p *int, name string, def int, aliases ...string) {
	gopt.Increment(name, def, aliases...)
	*p = def
	gopt.option(name).setIntPtr(p)
}

func (gopt *GetOpt) handleIncrement(name string, argument string, usedAlias string) error {
	Debug.Println("handleIncrement")
	opt := gopt.option(name)
	opt.setCalled()
	opt.setInt(opt.getInt() + 1)
	return nil
}

// func (opt *GetOpt) StringMulti(name string, def []string, min int, max int, aliases ...string) {}
// func (opt *GetOpt) StringMapMulti(name string, def map[string]string, min int, max int, aliases ...string) {}
// func (opt *GetOpt) Procedure(name string, lambda_func int, aliases ...string) {}

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
				// move index to next possition (to not include '--') and return remaining.
				gopt.args.next()
				remaining = append(remaining, gopt.args.remaining()...)
				Debug.Printf("return %v, %v", remaining, nil)
				return remaining, nil
			}
			Debug.Printf("Parse continue\n")
			for _, optElement := range optList {
				Debug.Printf("Parse optElement: %s\n", optElement)
				if optName, ok := gopt.getOptionFromAliases(optElement); ok {
					Debug.Printf("Parse found opt_list\n")
					opt := gopt.option(optName)
					handler := opt.handler
					Debug.Printf("handler found: name %s, argument %s, index %d, list %s\n", optName, argument, gopt.args.index(), optList[0])
					err := handler(optName, argument, optElement)
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
	Debug.Printf("return %v, %v", remaining, nil)
	return remaining, nil
}
