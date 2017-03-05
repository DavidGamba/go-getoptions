// This file is part of go-getoptions.
//
// Copyright (C) 2015-2017  David Gamba Rios
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// internal option struct and methods.

package getoptions

import (
	"fmt"
	"strconv"
	"strings"
)

type optionType int

const (
	stringType optionType = iota
	intType
	float64Type
	stringRepeatType
	intRepeatType
	stringMapType
)

type option struct {
	name          string
	aliases       []string
	value         interface{} // Value without type safety
	called        bool        // Indicates if the option was passed on the command line.
	handler       handlerType // method used to handle the option
	isOptionalOpt bool        // Indicates if an option has an optional argument
	optType       optionType  // Option Type
	// Pointer receivers:
	pBool    *bool             // receiver for bool pointer
	pString  *string           // receiver for string pointer
	pInt     *int              // receiver for int pointer
	pFloat64 *float64          // receiver for float64 pointer
	pStringS *[]string         // receiver for string slice pointer
	pIntS    *[]int            // receiver for int slice pointer
	stringM  map[string]string // receiver for string map pointer
	minArgs  int               // minimum args when using multi
	maxArgs  int               // maximum args when using multi
}

func newOption(name string, aliases []string) *option {
	return &option{
		name:    name,
		aliases: aliases,
	}
}

func (opt *option) setHandler(h handlerType) {
	opt.handler = h
}

func (opt *option) setCalled() {
	opt.called = true
}

func (opt *option) setIsOptional() {
	opt.isOptionalOpt = true
}

func (opt *option) isOptional() bool {
	return opt.isOptionalOpt
}

func (opt *option) setBool(b bool) {
	opt.value = b
	*opt.pBool = b
}

func (opt *option) getBool() bool {
	return *opt.pBool
}

func (opt *option) setBoolPtr(b *bool) {
	opt.value = *b
	opt.pBool = b
}

func (opt *option) setStringPtr(s *string) {
	opt.value = *s
	opt.pString = s
}

func (opt *option) setInt(i int) {
	opt.value = i
	*opt.pInt = i
}

func (opt *option) getInt() int {
	return *opt.pInt
}

func (opt *option) setIntPtr(i *int) {
	opt.value = *i
	opt.pInt = i
}

func (opt *option) setFloat64(f float64) {
	opt.value = f
	*opt.pFloat64 = f
}

func (opt *option) setFloat64Ptr(f *float64) {
	opt.value = *f
	opt.pFloat64 = f
}

func (opt *option) setStringSlicePtr(s *[]string) {
	opt.value = *s
	opt.pStringS = s
}

func (opt *option) setIntSlicePtr(s *[]int) {
	opt.value = *s
	opt.pIntS = s
}

func (opt *option) setStringMap(m map[string]string) {
	opt.value = m
	opt.stringM = m
}

func (opt *option) setKeyValueToStringMap(k, v string) {
	opt.stringM[k] = v
	opt.value = opt.stringM
}

func (opt *option) setMin(min int) {
	opt.minArgs = min
}

func (opt *option) min() int {
	return opt.minArgs
}

func (opt *option) setMax(max int) {
	opt.maxArgs = max
}

func (opt *option) max() int {
	return opt.maxArgs
}

func (opt *option) save(name string, a ...string) error {
	Debug.Printf("optType: %d\n", opt.optType)
	switch opt.optType {
	case intType:
		i, err := strconv.Atoi(a[0])
		if err != nil {
			return fmt.Errorf(ErrorConvertToInt, name, a[0])
		}
		opt.value = i
		*opt.pInt = i
		return nil
	case float64Type:
		// TODO: Read the different errors when parsing float
		i, err := strconv.ParseFloat(a[0], 64)
		if err != nil {
			return fmt.Errorf(ErrorConvertToFloat64, name, a[0])
		}
		opt.setFloat64(i)
		return nil
	case stringRepeatType:
		*opt.pStringS = append(*opt.pStringS, a...)
		opt.value = *opt.pStringS
		return nil
	case intRepeatType:
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
					return fmt.Errorf(ErrorConvertToInt, name, e)
				}
				in2, err := strconv.Atoi(n2)
				if err != nil {
					// TODO: Create new error description for this error.
					return fmt.Errorf(ErrorConvertToInt, name, e)
				}
				if in1 < in2 {
					for j := in1; j <= in2; j++ {
						is = append(is, j)
					}
				} else {
					// TODO: Create new error description for this error.
					return fmt.Errorf(ErrorConvertToInt, name, e)
				}
			} else {
				i, err := strconv.Atoi(e)
				if err != nil {
					return fmt.Errorf(ErrorConvertToInt, name, e)
				}
				is = append(is, i)
			}
		}
		*opt.pIntS = append(*opt.pIntS, is...)
		opt.value = *opt.pIntS
		return nil
	case stringMapType:
		keyValue := strings.Split(a[0], "=")
		if len(keyValue) < 2 {
			return fmt.Errorf(ErrorArgumentIsNotKeyValue, name)
		}
		opt.setKeyValueToStringMap(keyValue[0], keyValue[1])
		return nil
	default: // strType
		opt.value = a[0]
		*opt.pString = a[0]
		return nil
	}
}
