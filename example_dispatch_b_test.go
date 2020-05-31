package getoptions_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/DavidGamba/go-getoptions"
)

var dispatchHelpLogger = log.New(ioutil.Discard, "DEBUG: ", log.LstdFlags)

func dispatchHelpListRun(ctx context.Context, opt *getoptions.GetOpt, args []string) error {
	return nil
}

func ExampleGetOpt_Dispatch_bHelp() {
	opt := getoptions.New()
	opt.Bool("help", false, opt.Alias("?"))
	opt.Bool("debug", false)
	opt.SetRequireOrder()
	opt.SetUnknownMode(getoptions.Pass)
	list := opt.NewCommand("list", "list stuff")
	list.SetCommandFn(dispatchHelpListRun)
	list.Bool("list-opt", false)
	opt.HelpCommand("")
	remaining, err := opt.Parse([]string{"help"}) // <- argv set to call help
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
		os.Exit(1)
	}

	opt.Writer = os.Stdout // Print help to stdout instead of stderr for test purpose

	err = opt.Dispatch(context.Background(), "help", remaining)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
		os.Exit(1)
	}

	// Output:
	// SYNOPSIS:
	//     go-getoptions.test [--debug] [--help|-?] <command> [<args>]
	//
	// COMMANDS:
	//     help    Use 'go-getoptions.test help <command>' for extra details.
	//     list    list stuff
	//
	// OPTIONS:
	//     --debug      (default: false)
	//
	//     --help|-?    (default: false)
	//
	// Use 'go-getoptions.test help <command>' for extra details.
}
