package getoptions_test

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/DavidGamba/go-getoptions"
)

func ExampleGetOpt_Dispatch_cCommandHelp() {
	runFn := func(ctx context.Context, opt *getoptions.GetOpt, args []string) error {
		return nil
	}

	opt := getoptions.New()
	opt.Bool("debug", false)
	list := opt.NewCommand("list", "list stuff").SetCommandFn(runFn)
	list.Bool("list-opt", false)
	opt.HelpCommand("help", opt.Alias("?"))
	remaining, err := opt.Parse([]string{"help", "list"}) // <- argv set to call command help
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
		os.Exit(1)
	}

	getoptions.Writer = os.Stdout // Print help to stdout instead of stderr for test purpose

	err = opt.Dispatch(context.Background(), remaining)
	if err != nil {
		if !errors.Is(err, getoptions.ErrorHelpCalled) {
			fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
			os.Exit(1)
		}
	}

	// Output:
	// NAME:
	//     go-getoptions.test list - list stuff
	//
	// SYNOPSIS:
	//     go-getoptions.test list [--debug] [--help|-?] [--list-opt] [<args>]
	//
	// OPTIONS:
	//     --debug       (default: false)
	//
	//     --help|-?     (default: false)
	//
	//     --list-opt    (default: false)
	//
}
