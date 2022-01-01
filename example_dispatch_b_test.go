package getoptions_test

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/DavidGamba/go-getoptions"
)

func ExampleGetOpt_Dispatch_bHelp() {
	runFn := func(ctx context.Context, opt *getoptions.GetOpt, args []string) error {
		return nil
	}

	opt := getoptions.New()
	opt.Bool("debug", false)
	opt.NewCommand("list", "list stuff").SetCommandFn(runFn)
	opt.HelpCommand("help", "", opt.Alias("?"))
	remaining, err := opt.Parse([]string{"help"}) // <- argv set to call help
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
	// SYNOPSIS:
	//     go-getoptions.test [--debug] [--help|-?] <command> [<args>]
	//
	// COMMANDS:
	//     list    list stuff
	//
	// OPTIONS:
	//     --debug      (default: false)
	//
	//     --help|-?    (default: false)
	//
	// Use 'go-getoptions.test help <command>' for extra details.
}
