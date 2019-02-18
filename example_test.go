// This file is part of go-getoptions.
//
// Copyright (C) 2015-2019  David Gamba Rios
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// These examples demonstrate more intricate uses of the go-getoptions package.
package getoptions_test

import (
	"fmt"

	"github.com/DavidGamba/go-getoptions" // As getoptions
)

func Example() {
	// Declare the GetOptions object
	opt := getoptions.New()

	// Declare the variables you want your options to update
	var flag bool
	var str string
	var i int
	var f float64
	opt.BoolVar(&flag, "flag", false, opt.Alias("f", "alias2")) // Aliases can be defined
	opt.StringVar(&str, "string", "")
	opt.IntVar(&i, "i", 456)
	opt.Float64Var(&f, "float", 0)

	// Parse cmdline arguments or any provided []string
	// Normally you would run Parse on `os.Args[1:]`:
	// remaining, err := opt.Parse(os.Args[1:])
	remaining, err := opt.Parse([]string{
		"non-option", // Non options can be mixed with options at any place
		"-f",
		"--string=mystring", // Arguments can be passed with equals
		"--float", "3.14",   // Or with space
		"non-option2",
		"--", "--not-parsed", // -- indicates end of parsing
	})

	if err != nil {
		fmt.Printf("ERROR: %s\n", err)
	}

	// The remaining slice holds non-options and anything after --
	fmt.Printf("remaining: %v\n", remaining)

	if flag {
		fmt.Println("flag is true")
	}

	// Called method tells you if an option was actually called or not
	if opt.Called("string") {
		fmt.Printf("srt is %s\n", str)
	}

	// When the option is not called, it will have the provided default
	if !opt.Called("i") {
		fmt.Printf("i is %d\n", i)
	}

	fmt.Printf("f is %.2f", f)

	// Output:
	// remaining: [non-option non-option2 --not-parsed]
	// flag is true
	// srt is mystring
	// i is 456
	// f is 3.14

}
