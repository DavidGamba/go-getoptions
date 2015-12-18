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

		opt := getoptions.GetOptions()
		opt.Flag("flag")
		opt.Int("int")
		opt.String("string")
		remaining, error := opt.Parse(os.Args[1:])

Features

* Support for `--long` options.

* Support for short (`-s`) options with flexible behaviour:

 - Normal (default)
 - Bundling
 - SingleDash

* Supports command line options with '='.
For example: You can use `--string=mystring` and `--string mystring`.


Common Issues

The `opt.Option` map will have a value of `nil` when the argument is not passed
as a `Parse` parameter. The map needs to be checked to ensure it is not `nil`
before attempting a type assertion.

Wrong:

    if opt.Option["flag"].(bool) {

Correct:

    if opt.Option["flag"] != nil && opt.Option["flag"].(bool) {

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

// Options - map that contains the parsed arguments.
//
// Type assertions are required when using the elements of the map in cases
// where the compiler can't determine the type by context.
// For example: `opt.Options["flag"].(bool)`.
//
// The default value for un-called options is `nil` within this map.
// Make sure to check for `nil` before trying the type assertion.
// For example:
//
//   if opt.Option["flag"] != nil && opt.Option["flag"].(bool) {
type Options map[string]interface{}

// GetOpt - main struct with Option map result and global configuration settings.
//
// * Mode: Operation mode for short options: normal (default), bundling, singleDash.
type GetOpt struct {
	Option    Options // Map with resulting variables
	Mode      string  // Operation mode for short options: normal, bundling, singleDash
	Writer    io.Writer
	config    map[string]string
	obj       map[string]option
	args      []string
	argsIndex int
}

type option struct {
	name    string
	optType string // String indicating the type of option
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
		Option: Options{},
		Mode:   "normal",
		obj:    make(map[string]option),
	}
	return opt
}

// failIfDefined will *panic* if an option is defined twice.
// This is not an error because the programmer has to fix this!
func (opt *GetOpt) failIfDefined(name string) {
	// TODO: Add support for checking aliases
	if _, ok := opt.Option[name]; ok {
		panic(fmt.Sprintf("Option '%s' is already defined", name))
	}
}

// Flag - define a `bool` option and its aliases.
// The result will be available through the `Option` map.
func (opt *GetOpt) Flag(name string, aliases ...string) {
	opt.failIfDefined(name)
	aliases = append(aliases, name)
	opt.obj[name] = option{name: name,
		optType: "flag",
		aliases: aliases,
		handler: opt.handleFlag}
}

// FlagVar - define a `bool` option and its aliases.
// The result will be available through the variable marked by the given pointer.
func (opt *GetOpt) FlagVar(p *bool, name string, aliases ...string) {
	opt.Flag(name, aliases...)
	var tmp = opt.obj[name]
	tmp.optType = "varflag"
	tmp.pBool = p
	opt.obj[name] = tmp
}

func (opt *GetOpt) handleFlag(optName string, argument string, usedAlias string) error {
	Debug.Println("handlerFlag")
	opt.Option[optName] = true
	if opt.obj[optName].pBool != nil {
		*opt.obj[optName].pBool = true
	}
	return nil
}

// NFlag - define a *Negatable* `bool` option and its aliases.
// The result will be available through the `Option` map.
//
// NFlag automatically makes aliases with the prefix 'no' and 'no-' to the given name and aliases.
// When the argument is prefixed by 'no' (or by 'no-'), for example '--no-nflag', the value is set to false.
// Otherwise, with a regular call, for example '--nflag', it is set to true.
func (opt *GetOpt) NFlag(name string, aliases ...string) {
	opt.failIfDefined(name)
	aliases = append(aliases, name)
	aliases = append(aliases, "no"+name)
	aliases = append(aliases, "no-"+name)
	for _, a := range aliases {
		aliases = append(aliases, "no"+a)
		aliases = append(aliases, "no-"+a)
	}
	opt.obj[name] = option{name: name,
		aliases: aliases,
		optType: "nflag",
		handler: opt.handleNFlag}
}

// NFlagVar - define a *Negatable* `bool` option and its aliases.
// The result will be available through the variable marked by the given pointer.
//
// NFlagVar automatically makes aliases with the prefix 'no' and 'no-' to the given name and aliases.
// When the argument is prefixed by 'no' (or by 'no-'), for example '--no-nflag', the value is set to false.
// Otherwise, with a regular call, for example '--nflag', it is set to true.
func (opt *GetOpt) NFlagVar(p *bool, name string, aliases ...string) {
	opt.NFlag(name, aliases...)
	var tmp = opt.obj[name]
	tmp.optType = "varnflag"
	tmp.pBool = p
	opt.obj[name] = tmp
}

func (opt *GetOpt) handleNFlag(optName string, argument string, usedAlias string) error {
	Debug.Println("handleNFlag")
	if strings.HasPrefix(usedAlias, "no-") {
		opt.Option[optName] = false
		if opt.obj[optName].pBool != nil {
			*opt.obj[optName].pBool = false
		}
	} else {
		opt.Option[optName] = true
		if opt.obj[optName].pBool != nil {
			*opt.obj[optName].pBool = true
		}
	}
	return nil
}

// String - define a `string` option and its aliases.
// The result will be available through the `Option` map.
func (opt *GetOpt) String(name string, aliases ...string) {
	opt.failIfDefined(name)
	aliases = append(aliases, name)
	opt.obj[name] = option{
		name:    name,
		aliases: aliases,
		optType: "string",
		handler: opt.handleString,
	}
}

// StringVar - define a `string` option and its aliases.
// The result will be available through the variable marked by the given pointer.
func (opt *GetOpt) StringVar(p *string, name string, aliases ...string) {
	opt.String(name, aliases...)
	var tmp = opt.obj[name]
	tmp.optType = "stringVar"
	tmp.pString = p
	opt.obj[name] = tmp
}

func (opt *GetOpt) handleString(optName string, argument string, usedAlias string) error {
	Debug.Printf("handleString opt.args: %v(%d)\n", opt.args, len(opt.args))
	if argument != "" {
		opt.Option[optName] = argument
		if opt.obj[optName].pString != nil {
			*opt.obj[optName].pString = argument
		}
		Debug.Printf("handleOption option: %v, Option: %v\n", opt.obj[optName].optType, opt.Option)
		return nil
	}
	opt.argsIndex++
	Debug.Printf("len: %d, %d", len(opt.args), opt.argsIndex)
	if len(opt.args) < opt.argsIndex+1 {
		return fmt.Errorf("Missing argument for option '%s'!", optName)
	}
	opt.Option[optName] = opt.args[opt.argsIndex]
	if opt.obj[optName].pString != nil {
		*opt.obj[optName].pString = opt.args[opt.argsIndex]
	}
	return nil
}

// StringOptional - define a `string` option and its aliases.
// The result will be available through the `Option` map.
//
// StringOptional will set the string to the provided default value when no value is given.
// For example, when called with `--strOpt value`, the value is `value`.
// when called with `--strOpt` the value is the given default.
func (opt *GetOpt) StringOptional(name string, def string, aliases ...string) {
	opt.failIfDefined(name)
	aliases = append(aliases, name)
	opt.obj[name] = option{name: name,
		aliases: aliases,
		optType: "stringOptional",
		def:     def,
		handler: opt.handleStringOptional,
	}
}

func (opt *GetOpt) handleStringOptional(optName string, argument string, usedAlias string) error {
	if argument != "" {
		opt.Option[optName] = argument
		Debug.Printf("handleOption option: %v, Option: %v\n", opt.obj[optName].optType, opt.Option)
		return nil
	}
	if len(opt.args) < opt.argsIndex+2 {
		opt.Option[optName] = opt.obj[optName].def
	} else {
		// TODO: Check if next arg is option
	}
	return nil
}

// Int - define an `int` option and its aliases.
// The result will be available through the `Option` map.
func (opt *GetOpt) Int(name string, aliases ...string) {
	opt.failIfDefined(name)
	aliases = append(aliases, name)
	opt.obj[name] = option{name: name,
		aliases: aliases,
		optType: "int",
		handler: opt.handleInt,
	}
}

// IntVar - define an `int` option and its aliases.
// The result will be available through the variable marked by the given pointer.
func (opt *GetOpt) IntVar(p *int, name string, aliases ...string) {
	opt.Int(name, aliases...)
	var tmp = opt.obj[name]
	tmp.optType = "intVar"
	tmp.pInt = p
	opt.obj[name] = tmp
}

func (opt *GetOpt) handleInt(optName string, argument string, usedAlias string) error {
	if argument != "" {
		iArg, err := strconv.Atoi(argument)
		if err != nil {
			return fmt.Errorf("Can't convert string to int: %q", err)
		}
		opt.Option[optName] = iArg
		if opt.obj[optName].pInt != nil {
			*opt.obj[optName].pInt = iArg
		}
		Debug.Printf("handleOption option: %v, Option: %v\n", opt.obj[optName].optType, opt.Option)
		return nil
	}
	opt.argsIndex++
	if len(opt.args) < opt.argsIndex+1 {
		return fmt.Errorf("Missing argument for option '%s'!", optName)
	}
	iArg, err := strconv.Atoi(opt.args[opt.argsIndex])
	if err != nil {
		return fmt.Errorf("Can't convert string to int: %q", err)
	}
	opt.Option[optName] = iArg
	if opt.obj[optName].pInt != nil {
		*opt.obj[optName].pInt = iArg
	}
	return nil
}

// IntOptional - define a `int` option and its aliases.
// The result will be available through the `Option` map.
//
// IntOptional will set the int to the provided default value when no value is given.
// For example, when called with `--intOpt 123`, the value is `123`.
// when called with `--intOpt` the value is the given default.
func (opt *GetOpt) IntOptional(name string, def int, aliases ...string) {
	opt.failIfDefined(name)
	aliases = append(aliases, name)
	opt.obj[name] = option{name: name,
		aliases: aliases,
		optType: "intOptional",
		def:     def,
	}
	panic(fmt.Sprintf("Not implemented IntOptional"))
}

func (opt *GetOpt) handleIntOptional(optName string, argument string, usedAlias string) error {
	return nil
}

// TODO: Change name to StringSlice?

// StringRepeat - define a `[]string` option and its aliases.
// The result will be available through the `Option` map.
//
// StringRepeat will accept multiple calls to the same option and append them
// to the `[]string`.
// For example, when called with `--strRpt 1 --strRpt 2`, the value is `[]string{"1", "2"}`.
func (opt *GetOpt) StringRepeat(name string, aliases ...string) {
	opt.failIfDefined(name)
	aliases = append(aliases, name)
	opt.obj[name] = option{
		name:    name,
		aliases: aliases,
		optType: "stringRepeat",
		handler: opt.handleStringRepeat,
	}
}

func (opt *GetOpt) handleStringRepeat(optName string, argument string, usedAlias string) error {
	if _, ok := opt.Option[optName]; !ok {
		opt.Option[optName] = []string{}
	}
	if argument != "" {
		opt.Option[optName] = append(opt.Option[optName].([]string), argument)
		Debug.Printf("handleOption option: %v, Option: %v\n", opt.obj[optName].optType, opt.Option)
		return nil
	}
	opt.argsIndex++
	Debug.Printf("len: %d, %d", len(opt.args), opt.argsIndex)
	if len(opt.args) < opt.argsIndex+1 {
		return fmt.Errorf("Missing argument for option '%s'!", optName)
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
func (opt *GetOpt) StringMap(name string, aliases ...string) {
	opt.failIfDefined(name)
	aliases = append(aliases, name)
	opt.obj[name] = option{
		name:    name,
		aliases: aliases,
		optType: "stringMap",
		handler: opt.handleStringMap,
	}
}

func (opt *GetOpt) handleStringMap(optName string, argument string, usedAlias string) error {
	if _, ok := opt.Option[optName]; !ok {
		opt.Option[optName] = make(map[string]string)
	}
	if argument != "" {
		keyValue := strings.Split(argument, "=")
		if len(keyValue) < 2 {
			return fmt.Errorf("Argument for option '%s' should be of type 'key=value'!", optName)
		}
		opt.Option[optName].(map[string]string)[keyValue[0]] = keyValue[1]
		Debug.Printf("handleOption option: %v, Option: %v\n", opt.obj[optName].optType, opt.Option)
		return nil
	}
	opt.argsIndex++
	Debug.Printf("len: %d, %d", len(opt.args), opt.argsIndex)
	if len(opt.args) < opt.argsIndex+1 {
		return fmt.Errorf("Missing argument for option '%s'!", optName)
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
