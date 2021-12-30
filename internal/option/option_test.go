// This file is part of go-getoptions.
//
// Copyright (C) 2015-2021  David Gamba Rios
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package option

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/DavidGamba/go-getoptions/text"
)

func TestOption(t *testing.T) {
	tests := []struct {
		name   string
		option *Option
		input  []string
		output interface{}
		err    error
	}{
		{"empty", func() *Option {
			b := false
			return New("help", BoolType, &b)
		}(), []string{}, true, nil},
		{"empty", func() *Option {
			b := true
			return New("help", BoolType, &b)
		}(), []string{}, false, nil},
		{"bool", func() *Option {
			b := false
			return New("help", BoolType, &b)
		}(), []string{""}, true, nil},
		{"bool", func() *Option {
			b := true
			return New("help", BoolType, &b)
		}(), []string{""}, false, nil},
		{"bool setbool", func() *Option {
			b := true
			return New("help", BoolType, &b).SetBool(b)
		}(), []string{""}, false, nil},
		{"bool setbool", func() *Option {
			b := false
			return New("help", BoolType, &b).SetBool(b)
		}(), []string{""}, true, nil},
		{"bool env", func() *Option {
			b := false
			return New("help", BoolType, &b)
		}(), []string{"true"}, true, nil},
		{"bool env", func() *Option {
			b := false
			return New("help", BoolType, &b)
		}(), []string{"false"}, false, nil},
		{"bool env", func() *Option {
			b := true
			return New("help", BoolType, &b)
		}(), []string{"true"}, true, nil},
		{"bool env", func() *Option {
			b := true
			return New("help", BoolType, &b)
		}(), []string{"false"}, false, nil},

		{"string", func() *Option {
			s := ""
			return New("help", StringType, &s)
		}(), []string{""}, "", nil},
		{"string", func() *Option {
			s := ""
			return New("help", StringType, &s)
		}(), []string{"hola"}, "hola", nil},
		{"string", func() *Option {
			s := ""
			return New("help", StringType, &s).SetString("xxx")
		}(), []string{""}, "", nil},
		{"string", func() *Option {
			s := ""
			return New("help", StringType, &s).SetString("xxx")
		}(), []string{"hola"}, "hola", nil},

		{"string optional", func() *Option {
			s := ""
			return New("help", StringOptionalType, &s).SetString("xxx")
		}(), []string{"hola"}, "hola", nil},
		{"string optional", func() *Option {
			s := ""
			return New("help", StringOptionalType, &s).SetString("xxx")
		}(), []string{}, "xxx", nil},

		{"int", func() *Option {
			i := 0
			return New("help", IntType, &i)
		}(), []string{"123"}, 123, nil},
		{"int", func() *Option {
			i := 0
			return New("help", IntType, &i).SetInt(456)
		}(), []string{"123"}, 123, nil},
		{"int error", func() *Option {
			i := 0
			return New("help", IntType, &i)
		}(), []string{"123x"}, 0,
			fmt.Errorf(text.ErrorConvertToInt, "", "123x")},
		{"int error alias", func() *Option {
			i := 0
			return New("help", IntType, &i).SetCalled("int")
		}(), []string{"123x"}, 0,
			fmt.Errorf(text.ErrorConvertToInt, "int", "123x")},

		{"int optinal", func() *Option {
			i := 0
			return New("help", IntOptionalType, &i)
		}(), []string{"123"}, 123, nil},
		{"int optinal", func() *Option {
			i := 0
			return New("help", IntOptionalType, &i)
		}(), []string{}, 0, nil},
		{"int optinal", func() *Option {
			i := 0
			return New("help", IntOptionalType, &i).SetInt(456)
		}(), []string{"123"}, 123, nil},
		{"int optinal", func() *Option {
			i := 0
			return New("help", IntOptionalType, &i).SetInt(456)
		}(), []string{}, 456, nil},
		{"int optinal error", func() *Option {
			i := 0
			return New("help", IntOptionalType, &i)
		}(), []string{"123x"}, 0,
			fmt.Errorf(text.ErrorConvertToInt, "", "123x")},
		{"int optinal error alias", func() *Option {
			i := 0
			return New("help", IntOptionalType, &i).SetCalled("int")
		}(), []string{"123x"}, 0,
			fmt.Errorf(text.ErrorConvertToInt, "int", "123x")},

		{"float64", func() *Option {
			f := 0.0
			return New("help", Float64Type, &f)
		}(), []string{"123.123"}, 123.123, nil},
		{"float64 error", func() *Option {
			f := 0.0
			return New("help", Float64Type, &f)
		}(), []string{"123x"}, 0.0,
			fmt.Errorf(text.ErrorConvertToFloat64, "", "123x")},
		{"float64 error alias", func() *Option {
			f := 0.0
			return New("help", Float64Type, &f).SetCalled("float")
		}(), []string{"123x"}, 0.0,
			fmt.Errorf(text.ErrorConvertToFloat64, "float", "123x")},

		{"float64 optional", func() *Option {
			f := 0.0
			return New("help", Float64OptionalType, &f)
		}(), []string{"123.123"}, 123.123, nil},
		{"float64 optional", func() *Option {
			f := 0.0
			return New("help", Float64OptionalType, &f)
		}(), []string{}, 0.0, nil},
		{"float64 optional error", func() *Option {
			f := 0.0
			return New("help", Float64OptionalType, &f)
		}(), []string{"123x"}, 0.0,
			fmt.Errorf(text.ErrorConvertToFloat64, "", "123x")},
		{"float64 optional error alias", func() *Option {
			f := 0.0
			return New("help", Float64OptionalType, &f).SetCalled("float")
		}(), []string{"123x"}, 0.0,
			fmt.Errorf(text.ErrorConvertToFloat64, "float", "123x")},

		{"string slice", func() *Option {
			ss := []string{}
			return New("help", StringRepeatType, &ss)
		}(), []string{"hola", "mundo"}, []string{"hola", "mundo"}, nil},

		{"int slice", func() *Option {
			ii := []int{}
			return New("help", IntRepeatType, &ii)
		}(), []string{"123", "456"}, []int{123, 456}, nil},
		{"int slice error", func() *Option {
			ii := []int{}
			return New("help", IntRepeatType, &ii)
		}(), []string{"x"}, []int{},
			fmt.Errorf(text.ErrorConvertToInt, "", "x")},

		{"float64 slice", func() *Option {
			ii := []float64{}
			return New("help", Float64RepeatType, &ii)
		}(), []string{"123.456", "456.789"}, []float64{123.456, 456.789}, nil},
		{"float64 slice error", func() *Option {
			ii := []float64{}
			return New("help", Float64RepeatType, &ii)
		}(), []string{"x"}, []float64{},
			fmt.Errorf(text.ErrorConvertToFloat64, "", "x")},

		{"int slice range", func() *Option {
			ii := []int{}
			return New("help", IntRepeatType, &ii)
		}(), []string{"1..5"}, []int{1, 2, 3, 4, 5}, nil},
		{"int slice range error", func() *Option {
			ii := []int{}
			return New("help", IntRepeatType, &ii)
		}(), []string{"x..5"}, []int{},
			fmt.Errorf(text.ErrorConvertToInt, "", "x..5")},
		{"int slice range error", func() *Option {
			ii := []int{}
			return New("help", IntRepeatType, &ii)
		}(), []string{"1..x"}, []int{},
			fmt.Errorf(text.ErrorConvertToInt, "", "1..x")},
		{"int slice range error", func() *Option {
			ii := []int{}
			return New("help", IntRepeatType, &ii)
		}(), []string{"5..1"}, []int{},
			fmt.Errorf(text.ErrorConvertToInt, "", "5..1")},

		{"map", func() *Option {
			m := make(map[string]string)
			return New("help", StringMapType, &m)
		}(), []string{"hola=mundo"}, map[string]string{"hola": "mundo"}, nil},
		{"map", func() *Option {
			m := make(map[string]string)
			opt := New("help", StringMapType, &m)
			opt.MapKeysToLower = true
			return opt
		}(), []string{"Hola=Mundo"}, map[string]string{"hola": "Mundo"}, nil},
		// TODO: Currently map is only handling one argument at a time so the test below fails.
		//	It seems like the caller is handling this properly so I don't really know if this is needed here.
		// {"map", func() *Option {
		// 	m := make(map[string]string)
		// 	return New("help", StringMapType, &m)
		// }(), []string{"hola=mundo", "hello=world"}, map[string]string{"hola": "mundo", "hello": "world"}, nil},
		{"map error", func() *Option {
			m := make(map[string]string)
			return New("help", StringMapType, &m)
		}(), []string{"hola"}, map[string]string{},
			fmt.Errorf(text.ErrorArgumentIsNotKeyValue, "")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.option.Save(tt.input...)
			if err == nil && tt.err != nil {
				t.Errorf("got = '%#v', want '%#v'", err, tt.err)
			}
			if err != nil && tt.err == nil {
				t.Errorf("got = '%#v', want '%#v'", err, tt.err)
			}
			if err != nil && tt.err != nil && err.Error() != tt.err.Error() {
				t.Errorf("got = '%#v', want '%#v'", err, tt.err)
			}
			got := tt.option.Value()
			if !reflect.DeepEqual(got, tt.output) {
				t.Errorf("got = '%#v', want '%#v'", got, tt.output)
			}
		})
	}
}

func TestRequired(t *testing.T) {
	tests := []struct {
		name        string
		option      *Option
		input       []string
		output      interface{}
		err         error
		errRequired error
	}{
		{"bool", func() *Option {
			b := false
			return New("help", BoolType, &b)
		}(), []string{""}, true, nil, nil},
		{"bool", func() *Option {
			b := false
			return New("help", BoolType, &b).SetRequired("")
		}(), []string{""}, true, nil, ErrorMissingRequiredOption},
		{"bool", func() *Option {
			b := false
			return New("help", BoolType, &b).SetRequired("err")
		}(), []string{""}, true, nil, ErrorMissingRequiredOption},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.option.Save(tt.input...)
			if err == nil && tt.err != nil {
				t.Errorf("got = '%#v', want '%#v'", err, tt.err)
			}
			if err != nil && tt.err == nil {
				t.Errorf("got = '%#v', want '%#v'", err, tt.err)
			}
			if err != nil && tt.err != nil && err.Error() != tt.err.Error() {
				t.Errorf("got = '%#v', want '%#v'", err, tt.err)
			}
			got := tt.option.Value()
			if !reflect.DeepEqual(got, tt.output) {
				t.Errorf("got = '%#v', want '%#v'", got, tt.output)
			}
			err = tt.option.CheckRequired()
			if err == nil && tt.errRequired != nil {
				t.Errorf("got = '%#v', want '%#v'", err, tt.errRequired)
			}
			if err != nil && tt.errRequired == nil {
				t.Errorf("got = '%#v', want '%#v'", err, tt.errRequired)
			}
			if err != nil && tt.errRequired != nil && !errors.Is(err, tt.errRequired) {
				t.Errorf("got = '%#v', want '%#v'", err, tt.errRequired)
			}
		})
	}
}

func TestOther(t *testing.T) {
	i := 0
	opt := New("help", IntType, &i).SetAlias("?", "h").SetDescription("int help").SetHelpArgName("myint").SetDefaultStr("5").SetEnvVar("ENV_VAR")
	got := opt.Aliases
	expected := []string{"help", "?", "h"}
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("got = '%#v', want '%#v'", got, expected)
	}
	if opt.Int() != 0 {
		t.Errorf("got = '%#v', want '%#v'", opt.Int(), 0)
	}
	i = 3
	if opt.Int() != 3 {
		t.Errorf("got = '%#v', want '%#v'", opt.Int(), 0)
	}
	if opt.Description != "int help" {
		t.Errorf("got = '%#v', want '%#v'", opt.Description, "int help")
	}
	if opt.HelpArgName != "myint" {
		t.Errorf("got = '%#v', want '%#v'", opt.HelpArgName, "myint")
	}
	if opt.DefaultStr != "5" {
		t.Errorf("got = '%#v', want '%#v'", opt.DefaultStr, "5")
	}
	if opt.EnvVar != "ENV_VAR" {
		t.Errorf("got = '%#v', want '%#v'", opt.EnvVar, "ENV_VAR")
	}

	b := true
	list := []*Option{New("b", BoolType, &b), New("a", BoolType, &b), New("c", BoolType, &b)}
	expectedList := []*Option{New("a", BoolType, &b), New("b", BoolType, &b), New("c", BoolType, &b)}
	Sort(list)
	if !reflect.DeepEqual(list, expectedList) {
		t.Errorf("got = '%#v', want '%#v'", list, expectedList)
	}

	ii := []int{}
	opt = New("help", IntRepeatType, &ii)
	opt.MaxArgs = 2
	opt.Synopsis()
	if opt.HelpSynopsis != "--help <int>..." {
		t.Errorf("got = '%#v', want '%#v'", opt.HelpSynopsis, "--help <int>...")
	}
}
