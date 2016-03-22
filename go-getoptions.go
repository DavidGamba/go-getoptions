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
		opt := getoptions.GetOptions()

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

		if opt.Called["i"] {
			// ... do something with i
		}

		// Use subcommands by operating on the remaining items
		opt2 := getoptions.GetOptions()
		// ...
		remaining2, err := opt.Parse(remaining)


Features

* Support for `--long` options.

* Support for short (`-s`) options with flexible behaviour:

 - Normal (default)
 - Bundling
 - SingleDash

* Supports command line options with '='.
For example: You can use `--string=mystring` and `--string mystring`.

* opt.Called indicates if the parameter was passed on the command line.


Panic

The library will panic during compile time if it finds that the programmer
defined the same alias twice.
*/
package getoptions

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"regexp"
	"strconv"
	"strings"
)

// Debug Logger instance set to `ioutil.Discard` by default.
// Enable debug logging by setting: `Debug.SetOutput(os.Stderr)`
var Debug = log.New(ioutil.Discard, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)

// Option - map that contains the parsed arguments.
//
// Type assertions are required when using the elements of the map in cases
// where the compiler can't determine the type by context.
// For example: `opt.Option["flag"].(bool)`.
type Option map[string]interface{}

// GetOpt - main struct with Option map result and global configuration settings.
//
// * Mode: Operation mode for short options: normal (default), bundling, singleDash.
type GetOpt struct {
	Option           // Map with resulting variables
	Mode      string // Operation mode for short options: normal, bundling, singleDash
	Called    map[string]bool
	Writer    io.Writer
	config    map[string]string
	obj       map[string]option
	args      []string
	argsIndex int
}

type option struct {
	name    string
	aliases []string
	def     interface{} // default value
	handler func(optName string,
		argument string,
		usedAlias string) error // method used to handle the option
	// Pointer receivers:
	pBool   *bool   // receiver for bool pointer
	pInt    *int    // receiver for int pointer
	pString *string // receiver for string pointer
}

// GetOptions returns an empty object of type GetOpt.
// This is the starting point when using go-getoptions.
// For example:
//
//   opt := getoptions.GetOptions()
func GetOptions() *GetOpt {
	opt := &GetOpt{
		Option: Option{},
		Mode:   "normal",
		Called: make(map[string]bool),
		obj:    make(map[string]option),
	}
	return opt
}

// User facing errors

// ErrorMissingArgument holds the text for missing argument error.
// It has a string placeholder '%s' for the name of the option missing the argument.
var ErrorMissingArgument = "Missing argument for option '%s'!"

// ErrorArgumentWithDash holds the text for missing argument error in cases where the next argument looks like an option (starts with '-').
// It has a string placeholder '%s' for the name of the option missing the argument.
var ErrorArgumentWithDash = "Missing argument for option '%s'!\n" +
	"If passing arguments that start with '-' use --option=-argument"

// failIfDefined will *panic* if an option is defined twice.
// This is not an error because the programmer has to fix this!
func (opt *GetOpt) failIfDefined(name string, aliases []string) {
	Debug.Printf("checking option %s", name)
	if _, ok := opt.Option[name]; ok {
		panic(fmt.Sprintf("Option '%s' is already defined", name))
	}
	for _, a := range aliases {
		Debug.Printf("checking alias %s", a)
		if _, ok := opt.Option[a]; ok {
			panic(fmt.Sprintf("Alias '%s' is already defined as an option", a))
		}
		if optName, ok := opt.getOptionFromAliases(a); ok {
			if _, ok := opt.Option[optName]; ok {
				panic(fmt.Sprintf("Alias '%s' is already defined for option '%s'", a, optName))
			}
		}
	}
}

// Bool - define a `bool` option and its aliases.
// It returnns a `*bool` pointing to the variable holding the result.
// Additionally, the result will be available through the `Option` map.
// If the option is found, the result will be the opposite of the provided default.
func (opt *GetOpt) Bool(name string, def bool, aliases ...string) *bool {
	var b bool
	b = def
	opt.failIfDefined(name, aliases)
	opt.Option[name] = def
	aliases = append(aliases, name)
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
	opt.Called[optName] = true
	opt.Option[optName] = !opt.obj[optName].def.(bool)
	*opt.obj[optName].pBool = !opt.obj[optName].def.(bool)
	return nil
}

// NBool - define a *Negatable* `bool` option and its aliases.
// The result will be available through the `Option` map.
//
// NBool automatically makes aliases with the prefix 'no' and 'no-' to the given name and aliases.
// If the option is found, when the argument is prefixed by 'no' (or by 'no-'), for example '--no-nflag', the value is set to the provided default.
// Otherwise, with a regular call, for example '--nflag', it is set to the opposite of the default.
func (opt *GetOpt) NBool(name string, def bool, aliases ...string) *bool {
	var b bool
	b = def
	opt.failIfDefined(name, aliases)
	opt.Option[name] = def
	aliases = append(aliases, name)
	aliases = append(aliases, "no"+name)
	aliases = append(aliases, "no-"+name)
	for _, a := range aliases {
		aliases = append(aliases, "no"+a)
		aliases = append(aliases, "no-"+a)
	}
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
	opt.Called[optName] = true
	if strings.HasPrefix(usedAlias, "no-") {
		opt.Option[optName] = opt.obj[optName].def.(bool)
		*opt.obj[optName].pBool = opt.obj[optName].def.(bool)
	} else {
		opt.Option[optName] = !opt.obj[optName].def.(bool)
		*opt.obj[optName].pBool = !opt.obj[optName].def.(bool)
	}
	return nil
}

// String - define a `string` option and its aliases.
// The result will be available through the `Option` map.
// If not called, the return value will be that of the given default `def`.
func (opt *GetOpt) String(name, def string, aliases ...string) *string {
	var s string
	opt.failIfDefined(name, aliases)
	s = def
	opt.Option[name] = s
	aliases = append(aliases, name)
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
	opt.Called[optName] = true
	if argument != "" {
		opt.Option[optName] = argument
		*opt.obj[optName].pString = argument
		Debug.Printf("handleOption Option: %v\n", opt.Option)
		return nil
	}
	opt.argsIndex++
	Debug.Printf("len: %d, %d", len(opt.args), opt.argsIndex)
	if len(opt.args) < opt.argsIndex+1 {
		return fmt.Errorf(ErrorMissingArgument, optName)
	}
	// Check if next arg is option
	if optList, _ := isOption(opt.args[opt.argsIndex], opt.Mode); len(optList) > 0 {
		return fmt.Errorf(ErrorArgumentWithDash, optName)
	}
	opt.Option[optName] = opt.args[opt.argsIndex]
	*opt.obj[optName].pString = opt.args[opt.argsIndex]
	return nil
}

// StringOptional - define a `string` option and its aliases.
// The result will be available through the `Option` map.
//
// StringOptional will set the string to the provided default value when no value is given.
// For example, when called with `--strOpt value`, the value is `value`.
// when called with `--strOpt` the value is the given default.
func (opt *GetOpt) StringOptional(name string, def string, aliases ...string) *string {
	var s string
	s = def
	opt.failIfDefined(name, aliases)
	opt.Option[name] = s
	aliases = append(aliases, name)
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
	opt.Called[optName] = true
	if argument != "" {
		opt.Option[optName] = argument
		*opt.obj[optName].pString = argument
		Debug.Printf("handleOption Option: %v\n", opt.Option)
		return nil
	}
	opt.argsIndex++
	if len(opt.args) < opt.argsIndex+1 {
		opt.Option[optName] = opt.obj[optName].def
		*opt.obj[optName].pString = opt.obj[optName].def.(string)
		return nil
	}
	// Check if next arg is option
	if optList, _ := isOption(opt.args[opt.argsIndex], opt.Mode); len(optList) > 0 {
		opt.Option[optName] = opt.obj[optName].def
		*opt.obj[optName].pString = opt.obj[optName].def.(string)
		return nil
	}
	opt.Option[optName] = opt.args[opt.argsIndex]
	*opt.obj[optName].pString = opt.args[opt.argsIndex]
	return nil
}

// Int - define an `int` option and its aliases.
// The result will be available through the `Option` map.
func (opt *GetOpt) Int(name string, def int, aliases ...string) *int {
	var i int
	opt.failIfDefined(name, aliases)
	i = def
	opt.Option[name] = def
	aliases = append(aliases, name)
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
	opt.Called[optName] = true
	if argument != "" {
		iArg, err := strconv.Atoi(argument)
		if err != nil {
			return fmt.Errorf("Can't convert string to int: '%s'", argument)
		}
		opt.Option[optName] = iArg
		*opt.obj[optName].pInt = iArg
		Debug.Printf("handleOption Option: %v\n", opt.Option)
		return nil
	}
	opt.argsIndex++
	if len(opt.args) < opt.argsIndex+1 {
		return fmt.Errorf(ErrorMissingArgument, optName)
	}
	// Check if next arg is option
	if optList, _ := isOption(opt.args[opt.argsIndex], opt.Mode); len(optList) > 0 {
		return fmt.Errorf(ErrorArgumentWithDash, optName)
	}
	iArg, err := strconv.Atoi(opt.args[opt.argsIndex])
	if err != nil {
		return fmt.Errorf("Can't convert string to int: %q", err)
	}
	opt.Option[optName] = iArg
	*opt.obj[optName].pInt = iArg
	return nil
}

// IntOptional - define a `int` option and its aliases.
// The result will be available through the `Option` map.
//
// IntOptional will set the int to the provided default value when no value is given.
// For example, when called with `--intOpt 123`, the value is `123`.
// when called with `--intOpt` the value is the given default.
func (opt *GetOpt) IntOptional(name string, def int, aliases ...string) *int {
	var i int
	opt.failIfDefined(name, aliases)
	i = def
	opt.Option[name] = i
	aliases = append(aliases, name)
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
	opt.Called[optName] = true
	if argument != "" {
		iArg, err := strconv.Atoi(argument)
		if err != nil {
			return fmt.Errorf("Can't convert string to int: '%s'", argument)
		}
		opt.Option[optName] = iArg
		*opt.obj[optName].pInt = iArg
		Debug.Printf("handleOption Option: %v\n", opt.Option)
		return nil
	}
	opt.argsIndex++
	if len(opt.args) < opt.argsIndex+1 {
		opt.Option[optName] = opt.obj[optName].def
		*opt.obj[optName].pInt = opt.obj[optName].def.(int)
		return nil
	}
	// Check if next arg is option
	if optList, _ := isOption(opt.args[opt.argsIndex], opt.Mode); len(optList) > 0 {
		opt.Option[optName] = opt.obj[optName].def
		*opt.obj[optName].pInt = opt.obj[optName].def.(int)
		return nil
	}
	iArg, err := strconv.Atoi(opt.args[opt.argsIndex])
	if err != nil {
		return fmt.Errorf("Can't convert string to int: %q", err)
	}
	opt.Option[optName] = iArg
	*opt.obj[optName].pInt = iArg
	return nil
}

// StringSlice - define a `[]string` option and its aliases.
// The result will be available through the `Option` map.
//
// StringSlice will accept multiple calls to the same option and append them
// to the `[]string`.
// For example, when called with `--strRpt 1 --strRpt 2`, the value is `[]string{"1", "2"}`.
func (opt *GetOpt) StringSlice(name string, aliases ...string) *[]string {
	opt.failIfDefined(name, aliases)
	s := []string{}
	opt.Option[name] = s
	aliases = append(aliases, name)
	opt.obj[name] = option{
		name:    name,
		aliases: aliases,
		handler: opt.handleStringRepeat,
	}
	return &s
}

func (opt *GetOpt) handleStringRepeat(optName string, argument string, usedAlias string) error {
	opt.Called[optName] = true
	if _, ok := opt.Option[optName]; !ok {
		opt.Option[optName] = []string{}
	}
	if argument != "" {
		opt.Option[optName] = append(opt.Option[optName].([]string), argument)
		Debug.Printf("handleOption Option: %v\n", opt.Option)
		return nil
	}
	opt.argsIndex++
	Debug.Printf("len: %d, %d", len(opt.args), opt.argsIndex)
	if len(opt.args) < opt.argsIndex+1 {
		return fmt.Errorf(ErrorMissingArgument, optName)
	}
	opt.Option[optName] = append(opt.Option[optName].([]string), opt.args[opt.argsIndex])
	return nil
}

// StringMap - define a `map[string]string` option and its aliases.
// The result will be available through the `Option` map.
//
// StringMap will accept multiple calls of `key=value` type to the same option
// and add them to the `map[string]string` result.
// For example, when called with `--strMap k=v --strMap k2=v2`, the value is
// `map[string]string{"k":"v", "k2": "v2"}`.
func (opt *GetOpt) StringMap(name string, aliases ...string) *map[string]string {
	opt.failIfDefined(name, aliases)
	s := make(map[string]string)
	opt.Option[name] = s
	aliases = append(aliases, name)
	opt.obj[name] = option{
		name:    name,
		aliases: aliases,
		handler: opt.handleStringMap,
	}
	return &s
}

func (opt *GetOpt) handleStringMap(optName string, argument string, usedAlias string) error {
	opt.Called[optName] = true
	if _, ok := opt.Option[optName]; !ok {
		opt.Option[optName] = make(map[string]string)
	}
	if argument != "" {
		keyValue := strings.Split(argument, "=")
		if len(keyValue) < 2 {
			return fmt.Errorf("Argument for option '%s' should be of type 'key=value'!", optName)
		}
		opt.Option[optName].(map[string]string)[keyValue[0]] = keyValue[1]
		Debug.Printf("handleOption Option: %v\n", opt.Option)
		return nil
	}
	opt.argsIndex++
	Debug.Printf("len: %d, %d", len(opt.args), opt.argsIndex)
	if len(opt.args) < opt.argsIndex+1 {
		return fmt.Errorf(ErrorMissingArgument, optName)
	}
	keyValue := strings.Split(opt.args[opt.argsIndex], "=")
	opt.Option[optName].(map[string]string)[keyValue[0]] = keyValue[1]
	return nil
}

// func (opt *GetOpt) StringMulti(name string, def []string, min int, max int, aliases ...string) {}
// func (opt *GetOpt) StringMapMulti(name string, def map[string]string, min int, max int, aliases ...string) {}
// func (opt *GetOpt) Increment(name string, def int, aliases ...string) {}
// func (opt *GetOpt) Procedure(name string, lambda_func int, aliases ...string) {}

// Stringer - print a nice looking representation of the resulting `Option` map.
func (opt *GetOpt) Stringer() string {
	return fmt.Sprintf("%v", opt.Option)
}

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
	// Attempt to match intial chars of option
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
The behaviour changes depending on the mode: normal, bundling or SingleDash.
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
		if optList, argument := isOption(arg, opt.Mode); len(optList) > 0 {
			Debug.Printf("Parse opt_list: %v, argument: %v\n", optList, argument)
			// Check for termination: '--'
			if optList[0] == "--" {
				Debug.Printf("Parse -- found\n")
				remaining = append(remaining, args[opt.argsIndex+1:]...)
				Debug.Println(opt.Option)
				Debug.Printf("return %v, %v", remaining, nil)
				return remaining, nil
			}
			Debug.Printf("Parse continue\n")
			// TODO: Handle bundling options. Only index 0 is handled.
			if optName, ok := opt.getOptionFromAliases(optList[0]); ok {
				Debug.Printf("Parse found opt_list\n")
				handler := opt.obj[optName].handler
				Debug.Printf("handler found: %s, %s, %d, %s\n", optName, argument, opt.argsIndex, optList[0])
				err := handler(optName, argument, optList[0])
				if err != nil {
					Debug.Println(opt.Option)
					Debug.Printf("return %v, %v", nil, err)
					return nil, err
				}
			} else {
				Debug.Printf("opt_list not found for '%s'\n", optList[0])
				Debug.Println(opt.Option)
				err := fmt.Errorf("Unknown option '%s'", optList[0])
				Debug.Printf("return %v, %v", nil, err)
				return nil, err
			}
		} else {
			remaining = append(remaining, arg)
		}
	}
	Debug.Println(opt.Option)
	Debug.Printf("return %v, %v", remaining, nil)
	return remaining, nil
}
