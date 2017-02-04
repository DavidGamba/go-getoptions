// This file is part of go-getoptions.
//
// Copyright (C) 2015-2017  David Gamba Rios
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// internal option struct and methods.

package getoptions

type option struct {
	name    string
	aliases []string
	value   interface{} // Value without type safety
	called  bool        // Indicates if the option was passed on the command line.
	handler handlerType // method used to handle the option
	// Pointer receivers:
	pBool    *bool             // receiver for bool pointer
	pString  *string           // receiver for string pointer
	pInt     *int              // receiver for int pointer
	pFloat64 *float64          // receiver for float64 pointer
	pStringS *[]string         // receiver for string slice pointer
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

func (opt *option) setString(s string) {
	opt.value = s
	*opt.pString = s
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

func (opt *option) appendStringSlice(s ...string) {
	*opt.pStringS = append(*opt.pStringS, s...)
	opt.value = *opt.pStringS
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
