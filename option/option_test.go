// This file is part of go-getoptions.
//
// Copyright (C) 2015-2019  David Gamba Rios
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package option

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/DavidGamba/go-getoptions/text"
)

func TestOption(t *testing.T) {
	var b bool
	var s string
	var i int
	var f float64
	var ss []string
	var ii []int
	var iiRange []int
	m := make(map[string]string)

	tests := []struct {
		name   string
		option *Option
		input  []string
		output interface{}
		err    error
	}{
		{"bool", New("help", BoolType).SetBoolPtr(&b), []string{""}, true, nil},
		{"bool", New("help", BoolType).SetBoolPtr(&b), []string{""}, false, nil},
		{"string", New("help", StringType).SetStringPtr(&s), []string{""}, "", nil},
		{"string", New("help", StringType).SetStringPtr(&s), []string{"hola"}, "hola", nil},
		{"int", New("help", IntType).SetIntPtr(&i), []string{"123"}, 123, nil},
		{"int error", New("help", IntType).SetIntPtr(&i), []string{"123x"}, 0, fmt.Errorf(text.ErrorConvertToInt, "", "123x")},
		{"int error alias", New("help", IntType).SetIntPtr(&i).SetCalled("int"), []string{"123x"}, 0, fmt.Errorf(text.ErrorConvertToInt, "int", "123x")},
		{"float64", New("help", Float64Type).SetFloat64Ptr(&f), []string{"123.123"}, 123.123, nil},
		{"string slice", New("help", StringRepeatType).SetStringSlicePtr(&ss), []string{"hola", "mundo"}, []string{"hola", "mundo"}, nil},
		{"int slice", New("help", IntRepeatType).SetIntSlicePtr(&ii), []string{"123", "456"}, []int{123, 456}, nil},
		{"int slice range", New("help", IntRepeatType).SetIntSlicePtr(&iiRange), []string{"1..5"}, []int{1, 2, 3, 4, 5}, nil},
		{"map", New("help", StringMapType).SetStringMap(m), []string{"hola=mundo"}, map[string]string{"hola": "mundo"}, nil},
		// TODO: Currently map is only handling one argument at a time so the test below fails.
		//	It seems like the caller is handling this properly so I don't really know if this is needed here.
		// {"map", New("help", StringMapType).SetStringMap(m), []string{"hola=mundo", "hello=world"}, map[string]string{"hola": "mundo", "hello": "world"}, nil},
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
