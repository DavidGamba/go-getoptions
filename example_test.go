// This file is part of go-getoptions.
//
// Copyright (C) 2015  David Gamba Rios
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// These examples demonstrate more intricate uses of the go-getoptions package.
package getoptions_test

import (
	"fmt"
	"github.com/davidgamba/go-getoptions" // As getoptions
)

func Example() {
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

	// Normally you would run Parse on `os.Args[1:]`:
	// remaining, err := opt.Parse(os.Args[1:])
	remaining, err := opt.Parse([]string{
		"--bp",
		"word1",            // Non options can be mixed with options at any place
		"--sp", "strValue", // Values can be separated by space
		"--ip=123", // Or they can be passed after an equal `=` sign
		"word2",
		"--alias", // You can use any alias to call an option
		"--s=string",
		"word3",
		"--", "--not-parsed", // -- indicates end of parsing
	})

	if err != nil {
		fmt.Printf("ERROR: %s\n", err)
	}

	// The remaining slice holds non-options and anything after --
	fmt.Printf("remaining: %v\n", remaining)

	if *bp {
		fmt.Println("*bp is true")
	}

	// In the case of Bool and BoolVar, the value when called is the negated default
	if !b {
		fmt.Println("b is false")
	}

	// Called Map tells you if an option was actually called or not
	if opt.Called["ip"] {
		fmt.Printf("*ip is %d\n", *ip)
	}

	fmt.Printf("*sp is %s\n", *sp)
	fmt.Printf("s is %s\n", s)

	// When the option is not called, it will have the provided default
	if !opt.Called["i"] {
		fmt.Printf("i is %d\n", i)
	}

	// Output:
	// remaining: [word1 word2 word3 --not-parsed]
	// *bp is true
	// b is false
	// *ip is 123
	// *sp is strValue
	// s is string
	// i is 456

}
