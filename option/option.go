// This file is part of go-getoptions.
//
// Copyright (C) 2015-2019  David Gamba Rios
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package option - internal option struct and methods.
package option

import (
	"fmt"
	"io/ioutil"
	"log"
	"sort"
	"strconv"
	"strings"

	"github.com/DavidGamba/go-getoptions/text"
)

// Debug Logger instance set to `ioutil.Discard` by default.
// Enable debug logging by setting: `Debug.SetOutput(os.Stderr)`.
var Debug = log.New(ioutil.Discard, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)

// HandlerType - Signature for the function that handles saving to the option.
type HandlerType func(optName string, argument string, usedAlias string) error

// OptionType - Indicates the type of option.
type OptionType int

// Option Types
const (
	BoolType OptionType = iota
	StringType
	IntType
	Float64Type
	StringRepeatType
	IntRepeatType
	StringMapType
)

// Option - main object
type Option struct {
	Name           string
	Aliases        []string
	Called         bool        // Indicates if the option was passed on the command line
	UsedAlias      string      // Alias used when the option was called
	Handler        HandlerType // method used to handle the option
	IsOptional     bool        // Indicates if an option has an optional argument
	MapKeysToLower bool        // Indicates if the option of map type has it keys set ToLower
	OptType        OptionType  // Option Type
	MinArgs        int         // minimum args when using multi
	MaxArgs        int         // maximum args when using multi

	IsRequired    bool   // Indicates if the option is required
	IsRequiredErr string // Error message for the required option

	// Help
	DefaultStr  string // String representation of default value
	Description string // Optional description used for help
	HelpArgName string // Optional arg name used for help

	// Pointer receivers:
	value    interface{}       // Value without type safety
	pBool    *bool             // receiver for bool pointer
	pString  *string           // receiver for string pointer
	pInt     *int              // receiver for int pointer
	pFloat64 *float64          // receiver for float64 pointer
	pStringS *[]string         // receiver for string slice pointer
	pIntS    *[]int            // receiver for int slice pointer
	stringM  map[string]string // receiver for string map pointer
}

// New - Returns a new option object
func New(name string, optType OptionType) *Option {
	return &Option{
		Name:    name,
		OptType: optType,
		Aliases: []string{name},
	}
}

// SetAlias - Adds aliases to an option.
func (opt *Option) SetAlias(alias ...string) *Option {
	opt.Aliases = append(opt.Aliases, alias...)
	return opt
}

// SetRequired - Marks an option as required.
func (opt *Option) SetRequired(msg string) {
	opt.IsRequired = true
	opt.IsRequiredErr = msg
}

func (opt *Option) SetHandler(h HandlerType) {
	opt.Handler = h
}

// SetCalled - Convenience to avoid `opt.Called = true` verbosity.
func (opt *Option) SetCalled(usedAlias string) *Option {
	opt.Called = true
	opt.UsedAlias = usedAlias
	return opt
}

// SetIsOptional - Convenience to avoid `opt.IsOptional = true` verbosity.
func (opt *Option) SetIsOptional() {
	opt.IsOptional = true
}

// Value - Get untyped option value
func (opt *Option) Value() interface{} {
	return opt.value
}

func (opt *Option) SetDefaultStr(str string) *Option {
	opt.DefaultStr = str
	return opt
}

func (opt *Option) SetBool(b bool) {
	opt.value = b
	*opt.pBool = b
}

func (opt *Option) GetBool() bool {
	return *opt.pBool
}

func (opt *Option) SetBoolPtr(b *bool) *Option {
	opt.value = *b
	opt.pBool = b
	return opt
}

func (opt *Option) SetStringPtr(s *string) *Option {
	opt.value = *s
	opt.pString = s
	return opt
}

func (opt *Option) SetInt(i int) *Option {
	opt.value = i
	*opt.pInt = i
	return opt
}

func (opt *Option) GetInt() int {
	return *opt.pInt
}

func (opt *Option) SetIntPtr(i *int) *Option {
	opt.value = *i
	opt.pInt = i
	return opt
}

func (opt *Option) SetFloat64(f float64) {
	opt.value = f
	*opt.pFloat64 = f
}

func (opt *Option) SetFloat64Ptr(f *float64) *Option {
	opt.value = *f
	opt.pFloat64 = f
	return opt
}

func (opt *Option) SetStringSlicePtr(s *[]string) *Option {
	opt.value = *s
	opt.pStringS = s
	return opt
}

func (opt *Option) SetIntSlicePtr(s *[]int) *Option {
	opt.value = *s
	opt.pIntS = s
	return opt
}

func (opt *Option) SetStringMap(m map[string]string) *Option {
	opt.value = m
	opt.stringM = m
	return opt
}

func (opt *Option) SetKeyValueToStringMap(k, v string) {
	if opt.MapKeysToLower {
		opt.stringM[strings.ToLower(k)] = v
	} else {
		opt.stringM[k] = v
	}
	opt.value = opt.stringM
}

// SetMin - Convenience function for `opt.MinArgs = min`
func (opt *Option) SetMin(min int) {
	opt.MinArgs = min
}

// SetMax - Convenience function for `opt.MaxArgs = max`
func (opt *Option) SetMax(max int) {
	opt.MaxArgs = max
}

// Save - Saves the data provided into the option
func (opt *Option) Save(a ...string) error {
	Debug.Printf("optType: %d\n", opt.OptType)
	switch opt.OptType {
	case StringType:
		opt.value = a[0]
		*opt.pString = a[0]
		return nil
	case IntType:
		i, err := strconv.Atoi(a[0])
		if err != nil {
			return fmt.Errorf(text.ErrorConvertToInt, opt.UsedAlias, a[0])
		}
		opt.value = i
		*opt.pInt = i
		return nil
	case Float64Type:
		// TODO: Read the different errors when parsing float
		i, err := strconv.ParseFloat(a[0], 64)
		if err != nil {
			return fmt.Errorf(text.ErrorConvertToFloat64, opt.UsedAlias, a[0])
		}
		opt.SetFloat64(i)
		return nil
	case StringRepeatType:
		*opt.pStringS = append(*opt.pStringS, a...)
		opt.value = *opt.pStringS
		return nil
	case IntRepeatType:
		var is []int
		for _, e := range a {
			if strings.Contains(e, "..") {
				Debug.Printf("e: %s\n", e)
				n := strings.SplitN(e, "..", 2)
				Debug.Printf("n: %v\n", n)
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
						is = append(is, j)
					}
				} else {
					// TODO: Create new error description for this error.
					return fmt.Errorf(text.ErrorConvertToInt, opt.UsedAlias, e)
				}
			} else {
				i, err := strconv.Atoi(e)
				if err != nil {
					return fmt.Errorf(text.ErrorConvertToInt, opt.UsedAlias, e)
				}
				is = append(is, i)
			}
		}
		*opt.pIntS = append(*opt.pIntS, is...)
		opt.value = *opt.pIntS
		return nil
	case StringMapType:
		keyValue := strings.Split(a[0], "=")
		if len(keyValue) < 2 {
			return fmt.Errorf(text.ErrorArgumentIsNotKeyValue, opt.UsedAlias)
		}
		opt.SetKeyValueToStringMap(keyValue[0], keyValue[1])
		return nil
	default: // BoolType
		opt.SetBool(!opt.GetBool())
		return nil
	}
}

// Sort Interface
func Sort(list []*Option) {
	sort.Slice(list, func(i, j int) bool {
		return list[i].Name < list[j].Name
	})
}
