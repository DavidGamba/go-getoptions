// This file is part of go-getoptions.
//
// Copyright (C) 2015-2023  David Gamba Rios
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package option - internal option struct and methods.
package option

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"sort"
	"strconv"
	"strings"

	"github.com/DavidGamba/go-getoptions/text"
)

// Logger instance set to `ioutil.Discard` by default.
// Enable debug logging by setting: `Logger.SetOutput(os.Stderr)`.
var Logger = log.New(ioutil.Discard, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)

var ErrorMissingRequiredOption = errors.New("")

// Handler - Signature for the function that handles saving to the option.
type Handler func(optName string, argument string, usedAlias string) error

// Type - Indicates the type of option.
type Type int

// Option Types
const (
	BoolType Type = iota
	IncrementType

	StringType
	IntType
	Float64Type

	StringOptionalType
	IntOptionalType
	Float64OptionalType

	StringRepeatType
	IntRepeatType
	Float64RepeatType

	StringMapType
)

// Option - main object
type Option struct {
	Name           string
	Aliases        []string
	EnvVar         string  // Env Var that sets the option value
	Called         bool    // Indicates if the option was passed on the command line
	UsedAlias      string  // Alias/Env var used when the option was called
	Handler        Handler // method used to handle the option
	IsOptional     bool    // Indicates if an option has an optional argument
	MapKeysToLower bool    // Indicates if the option of map type has it keys set ToLower
	OptType        Type    // Option Type
	MinArgs        int     // minimum args when using multi
	MaxArgs        int     // maximum args when using multi

	IsRequired    bool   // Indicates if the option is required
	IsRequiredErr string // Error message for the required option

	// SuggestedValues used for completions, suggestions don't necessarily limit
	// the values you are able to use
	SuggestedValues []string
	ValidValues     []string // ValidValues that can be passed to Save

	// Help
	DefaultStr   string // String representation of default value
	Description  string // Optional description used for help
	HelpArgName  string // Optional arg name used for help
	HelpSynopsis string // Help synopsis

	boolDefault bool // copy of bool default value

	// Pointer receivers:
	pBool     *bool              // receiver for bool pointer
	pString   *string            // receiver for string pointer
	pInt      *int               // receiver for int pointer
	pFloat64  *float64           // receiver for float64 pointer
	pStringS  *[]string          // receiver for string slice pointer
	pIntS     *[]int             // receiver for int slice pointer
	pFloat64S *[]float64         // receiver for float64 slice pointer
	pStringM  *map[string]string // receiver for string map pointer

	Unknown bool // Temporary marker used during parsing

	// Verbatim text used to generate the option
	// Used in cases where the option is unknown and we eventually have to send it to the remaining slice.
	Verbatim string
}

// New - Returns a new option object
func New(name string, optType Type, data interface{}) *Option {
	opt := &Option{
		Name:    name,
		OptType: optType,
		Aliases: []string{name},
	}
	switch optType {
	case StringType:
		opt.HelpArgName = "string"
		opt.pString = data.(*string)
		opt.DefaultStr = fmt.Sprintf("\"%s\"", *data.(*string))
		opt.MinArgs = 1
		opt.MaxArgs = 1
	case StringOptionalType:
		opt.HelpArgName = "string"
		opt.pString = data.(*string)
		opt.DefaultStr = *data.(*string)
		opt.MinArgs = 0
		opt.MaxArgs = 1
		opt.IsOptional = true
	case StringRepeatType:
		opt.HelpArgName = "string"
		opt.pStringS = data.(*[]string)
		opt.DefaultStr = "[]"
		opt.MinArgs = 1
		opt.MaxArgs = 1 // By default we only allow one argument at a time
	case IntType:
		opt.HelpArgName = "int"
		opt.pInt = data.(*int)
		opt.DefaultStr = fmt.Sprintf("%d", *data.(*int))
		opt.MinArgs = 1
		opt.MaxArgs = 1
	case IntOptionalType:
		opt.HelpArgName = "int"
		opt.pInt = data.(*int)
		opt.DefaultStr = fmt.Sprintf("%d", *data.(*int))
		opt.MinArgs = 0
		opt.MaxArgs = 1
		opt.IsOptional = true
	case IntRepeatType:
		opt.HelpArgName = "int"
		opt.pIntS = data.(*[]int)
		opt.DefaultStr = "[]"
		opt.MinArgs = 1
		opt.MaxArgs = 1 // By default we only allow one argument at a time
	case Float64Type:
		opt.HelpArgName = "float64"
		opt.pFloat64 = data.(*float64)
		opt.DefaultStr = fmt.Sprintf("%f", *data.(*float64))
		opt.MinArgs = 1
		opt.MaxArgs = 1
	case Float64OptionalType:
		opt.HelpArgName = "float64"
		opt.pFloat64 = data.(*float64)
		opt.DefaultStr = fmt.Sprintf("%f", *data.(*float64))
		opt.MinArgs = 0
		opt.MaxArgs = 1
		opt.IsOptional = true
	case Float64RepeatType:
		opt.HelpArgName = "float64"
		opt.pFloat64S = data.(*[]float64)
		opt.DefaultStr = "[]"
		opt.MinArgs = 1
		opt.MaxArgs = 1 // By default we only allow one argument at a time
	case StringMapType:
		opt.HelpArgName = "key=value"
		opt.pStringM = data.(*map[string]string)
		opt.DefaultStr = "{}"
		opt.MinArgs = 1
		opt.MaxArgs = 1 // By default we only allow one argument at a time
	case IncrementType:
		opt.pInt = data.(*int)
		opt.DefaultStr = fmt.Sprintf("%d", *data.(*int))
		opt.MinArgs = 0
		opt.MaxArgs = 0
	case BoolType:
		opt.pBool = data.(*bool)
		opt.boolDefault = *data.(*bool)
		opt.DefaultStr = fmt.Sprintf("%t", *data.(*bool))
		opt.MinArgs = 0
		opt.MaxArgs = 0
	}
	opt.Synopsis()
	return opt
}

// ValidateMinMaxArgs - validates that the min and max make sense.
//
// NOTE: This should only be called to validate Repeat types.
func (opt *Option) ValidateMinMaxArgs() error {
	if opt.MinArgs <= 0 {
		return fmt.Errorf("min should be > 0")
	}
	if opt.MaxArgs <= 0 || opt.MaxArgs < opt.MinArgs {
		return fmt.Errorf("max should be > 0 and > min")
	}
	return nil
}

func (opt *Option) Synopsis() {
	aliases := []string{}
	for _, e := range opt.Aliases {
		if len(e) > 1 {
			e = "--" + e
		} else {
			// Don't add extra dash for lonesome dash
			if e != "-" {
				e = "-" + e
			}
		}
		aliases = append(aliases, e)
	}
	opt.HelpSynopsis = strings.Join(aliases, "|")
	if opt.OptType != BoolType {
		opt.HelpSynopsis += fmt.Sprintf(" <%s>", opt.HelpArgName)
	}
	if opt.MaxArgs > 1 {
		opt.HelpSynopsis += "..."
	}
}

// Value - Get untyped option value
func (opt *Option) Value() interface{} {
	switch opt.OptType {
	case StringType, StringOptionalType:
		return *opt.pString
	case StringRepeatType:
		return *opt.pStringS
	case IncrementType, IntType, IntOptionalType:
		return *opt.pInt
	case IntRepeatType:
		return *opt.pIntS
	case Float64Type, Float64OptionalType:
		return *opt.pFloat64
	case Float64RepeatType:
		return *opt.pFloat64S
	case StringMapType:
		return *opt.pStringM
	default: // BoolType:
		return *opt.pBool
	}
}

// SetAlias - Adds aliases to an option.
func (opt *Option) SetAlias(alias ...string) *Option {
	opt.Aliases = append(opt.Aliases, alias...)
	opt.Synopsis()
	return opt
}

// SetDescription - Updates the Description.
func (opt *Option) SetDescription(s string) *Option {
	opt.Description = s
	return opt
}

// SetHelpArgName - Updates the HelpArgName.
func (opt *Option) SetHelpArgName(s string) *Option {
	opt.HelpArgName = s
	opt.Synopsis()
	return opt
}

// SetDefaultStr - Updates the DefaultStr.
func (opt *Option) SetDefaultStr(s string) *Option {
	opt.DefaultStr = s
	return opt
}

// SetRequired - Marks an option as required.
func (opt *Option) SetRequired(msg string) *Option {
	opt.IsRequired = true
	opt.IsRequiredErr = msg
	return opt
}

// SetEnvVar - Sets the name of the Env var that sets the option's value.
func (opt *Option) SetEnvVar(name string) *Option {
	opt.EnvVar = name
	return opt
}

// CheckRequired - Returns error if the option is required.
func (opt *Option) CheckRequired() error {
	if opt.IsRequired {
		if !opt.Called {
			if opt.IsRequiredErr != "" {
				return fmt.Errorf("%w%s", ErrorMissingRequiredOption, opt.IsRequiredErr)
			}
			return fmt.Errorf("%w"+text.ErrorMissingRequiredOption, ErrorMissingRequiredOption, opt.Name)
		}
	}
	return nil
}

// SetCalled - Marks the option as called and records the alias used to call it.
func (opt *Option) SetCalled(usedAlias string) *Option {
	opt.Called = true
	opt.UsedAlias = usedAlias
	return opt
}

// SetBool - Set the option's data.
func (opt *Option) SetBool(b bool) *Option {
	*opt.pBool = b
	return opt
}

func (opt *Option) SetBoolAsOppositeToDefault() *Option {
	*opt.pBool = !opt.boolDefault
	return opt
}

// SetString - Set the option's data.
func (opt *Option) SetString(s string) *Option {
	*opt.pString = s
	return opt
}

// SetInt - Set the option's data.
func (opt *Option) SetInt(i int) *Option {
	*opt.pInt = i
	return opt
}

// Int - Get the option's data.
// Exposed due to handle increment. Maybe there is a better way.
func (opt *Option) Int() int {
	return *opt.pInt
}

// SetFloat64 - Set the option's data.
func (opt *Option) SetFloat64(f float64) *Option {
	*opt.pFloat64 = f
	return opt
}

// SetStringSlice - Set the option's data.
func (opt *Option) SetStringSlice(s []string) *Option {
	*opt.pStringS = s
	return opt
}

// SetIntSlice - Set the option's data.
func (opt *Option) SetIntSlice(s []int) *Option {
	*opt.pIntS = s
	return opt
}

// SetFloat64Slice - Set the option's data.
func (opt *Option) SetFloat64Slice(s []float64) *Option {
	*opt.pFloat64S = s
	return opt
}

// SetKeyValueToStringMap - Set the option's data.
func (opt *Option) SetKeyValueToStringMap(k, v string) *Option {
	if opt.MapKeysToLower {
		(*opt.pStringM)[strings.ToLower(k)] = v
	} else {
		(*opt.pStringM)[k] = v
	}
	return opt
}

// stringSliceIndex - indicates if an element is found in the slice and what its index is
func stringSliceIndex(ss []string, e string) (int, bool) {
	for i, s := range ss {
		if s == e {
			return i, true
		}
	}
	return -1, false
}

// Save - Saves the data provided into the option
func (opt *Option) Save(a ...string) error {
	Logger.Printf("name: %s, optType: %d\n", opt.Name, opt.OptType)
	if len(a) < 1 {
		switch opt.OptType {
		case BoolType:
			opt.SetBoolAsOppositeToDefault()
		case IncrementType:
			opt.SetInt(opt.Int() + 1)
		}
		return nil
	}
	for _, e := range a {
		if len(opt.ValidValues) > 0 {
			_, ok := stringSliceIndex(opt.ValidValues, e)
			if !ok {
				// TODO: convert to text variable
				return fmt.Errorf("wrong value for option '%s', valid values are %q", opt.Name, opt.ValidValues)
			}
		}
	}

	switch opt.OptType {
	case StringType, StringOptionalType:
		opt.SetString(a[0])
		return nil
	case IntType, IntOptionalType:
		i, err := strconv.Atoi(a[0])
		if err != nil {
			// TODO: Create error type for use in tests with errors.Is
			return fmt.Errorf(text.ErrorConvertToInt, opt.UsedAlias, a[0])
		}
		opt.SetInt(i)
		return nil
	case Float64Type, Float64OptionalType:
		// TODO: Read the different errors when parsing float
		f, err := strconv.ParseFloat(a[0], 64)
		if err != nil {
			// TODO: Create error type for use in tests with errors.Is
			return fmt.Errorf(text.ErrorConvertToFloat64, opt.UsedAlias, a[0])
		}
		opt.SetFloat64(f)
		return nil
	case StringRepeatType:
		opt.SetStringSlice(append(*opt.pStringS, a...))
		return nil
	case IntRepeatType:
		var ii []int
		for _, e := range a {
			if strings.Contains(e, "..") {
				Logger.Printf("e: %s\n", e)
				n := strings.SplitN(e, "..", 2)
				Logger.Printf("n: %v\n", n)
				n1, n2 := n[0], n[1]
				in1, err := strconv.Atoi(n1)
				if err != nil {
					// TODO: Create new error description for this error.
					return fmt.Errorf(text.ErrorConvertToInt, opt.UsedAlias, e)
				}
				in2, err := strconv.Atoi(n2)
				if err != nil {
					// TODO: Create new error description for this error.
					return fmt.Errorf(text.ErrorConvertToInt, opt.UsedAlias, e)
				}
				if in1 < in2 {
					for j := in1; j <= in2; j++ {
						ii = append(ii, j)
					}
				} else {
					// TODO: Create new error description for this error.
					return fmt.Errorf(text.ErrorConvertToInt, opt.UsedAlias, e)
				}
			} else {
				i, err := strconv.Atoi(e)
				if err != nil {
					// TODO: Create error type for use in tests with errors.Is
					return fmt.Errorf(text.ErrorConvertToInt, opt.UsedAlias, e)
				}
				ii = append(ii, i)
			}
		}
		opt.SetIntSlice(append(*opt.pIntS, ii...))
		return nil
	case Float64RepeatType:
		var ff []float64
		for _, e := range a {
			f, err := strconv.ParseFloat(e, 64)
			if err != nil {
				// TODO: Create error type for use in tests with errors.Is
				return fmt.Errorf(text.ErrorConvertToFloat64, opt.UsedAlias, e)
			}
			ff = append(ff, f)
		}
		opt.SetFloat64Slice(append(*opt.pFloat64S, ff...))
		return nil
	case StringMapType:
		keyValue := strings.Split(a[0], "=")
		if len(keyValue) < 2 {
			// TODO: Create error type for use in tests with errors.Is
			return fmt.Errorf(text.ErrorArgumentIsNotKeyValue, opt.UsedAlias)
		}
		opt.SetKeyValueToStringMap(keyValue[0], keyValue[1])
		return nil
	case IncrementType:
		opt.SetInt(opt.Int() + 1)
		return nil
	default: // BoolType:
		if len(a) > 0 && a[0] == "true" {
			opt.SetBool(true)
		} else if len(a) > 0 && a[0] == "false" {
			opt.SetBool(false)
		} else {
			opt.SetBoolAsOppositeToDefault()
		}
		return nil
	}
}

// Sort Interface
func Sort(list []*Option) {
	sort.Slice(list, func(i, j int) bool {
		return list[i].Name < list[j].Name
	})
}
