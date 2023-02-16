// This file is part of go-getoptions.
//
// Copyright (C) 2015-2023  David Gamba Rios
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// These examples demonstrate more intricate uses of the go-getoptions package.
package getoptions_test

import (
	"fmt"
	"os"

	"github.com/DavidGamba/go-getoptions" // As getoptions
)

func ExampleGetOpt_Alias() {
	opt := getoptions.New()
	opt.Bool("help", false, opt.Alias("?"))
	_, _ = opt.Parse([]string{"-?"})

	if opt.Called("help") {
		fmt.Println("help called")
	}

	if opt.CalledAs("help") == "?" {
		fmt.Println("help called as ?")
	}

	// Output:
	// help called
	// help called as ?
}

func ExampleGetOpt_ArgName() {
	opt := getoptions.New()
	opt.Bool("help", false, opt.Alias("?"))
	opt.String("host-default", "")
	opt.String("host-custom", "", opt.ArgName("hostname"))
	_, _ = opt.Parse([]string{"-?"})

	if opt.Called("help") {
		fmt.Println(opt.Help())
	}
	// Output:
	// SYNOPSIS:
	//     go-getoptions.test [--help|-?] [--host-custom <hostname>]
	//                        [--host-default <string>] [<args>]
	//
	// OPTIONS:
	//     --help|-?                   (default: false)
	//
	//     --host-custom <hostname>    (default: "")
	//
	//     --host-default <string>     (default: "")
}

func ExampleGetOpt_Bool() {
	opt := getoptions.New()
	opt.Bool("help", false, opt.Alias("?"))
	_, _ = opt.Parse([]string{"-?"})

	if opt.Called("help") {
		fmt.Println(opt.Help())
	}

	// Output:
	// SYNOPSIS:
	//     go-getoptions.test [--help|-?] [<args>]
	//
	// OPTIONS:
	//     --help|-?    (default: false)
}

func ExampleGetOpt_BoolVar() {
	var help bool
	opt := getoptions.New()
	opt.BoolVar(&help, "help", false, opt.Alias("?"))
	_, _ = opt.Parse([]string{"-?"})

	if help {
		fmt.Println(opt.Help())
	}

	// Output:
	// SYNOPSIS:
	//     go-getoptions.test [--help|-?] [<args>]
	//
	// OPTIONS:
	//     --help|-?    (default: false)
}

func ExampleGetOpt_Called() {
	opt := getoptions.New()
	opt.Bool("help", false, opt.Alias("?"))
	_, _ = opt.Parse([]string{"-?"})

	if opt.Called("help") {
		fmt.Println("help called")
	}

	if opt.CalledAs("help") == "?" {
		fmt.Println("help called as ?")
	}

	// Output:
	// help called
	// help called as ?
}

func ExampleGetOpt_CalledAs() {
	opt := getoptions.New()
	opt.Bool("help", false, opt.Alias("?"))
	_, _ = opt.Parse([]string{"-?"})

	if opt.Called("help") {
		fmt.Println("help called")
	}

	if opt.CalledAs("help") == "?" {
		fmt.Println("help called as ?")
	}

	// Output:
	// help called
	// help called as ?
}

func ExampleGetOpt_Description() {
	opt := getoptions.New()
	opt.HelpSynopsisArgs("[<commands>]")
	opt.Bool("help", false, opt.Alias("?"), opt.Description("Show help."))
	opt.String("hostname", "golang.org", opt.ArgName("host|IP"), opt.Description("Hostname to use."))
	opt.String("user", "", opt.ArgName("user_id"), opt.Required(), opt.Description("User to login as."))
	_, _ = opt.Parse([]string{"-?"})

	if opt.Called("help") {
		fmt.Println(opt.Help())
	}
	// Output:
	// SYNOPSIS:
	//     go-getoptions.test --user <user_id> [--help|-?] [--hostname <host|IP>]
	//                        [<commands>]
	//
	// REQUIRED PARAMETERS:
	//     --user <user_id>        User to login as.
	//
	// OPTIONS:
	//     --help|-?               Show help. (default: false)
	//
	//     --hostname <host|IP>    Hostname to use. (default: "golang.org")
}

func ExampleGetOpt_GetEnv() {
	os.Setenv("_AWS_PROFILE", "production")

	var profile string
	opt := getoptions.New()
	opt.StringVar(&profile, "profile", "default", opt.GetEnv("_AWS_PROFILE"))
	_, _ = opt.Parse([]string{})

	fmt.Println(profile)
	os.Unsetenv("_AWS_PROFILE")

	// Output:
	// production
}
