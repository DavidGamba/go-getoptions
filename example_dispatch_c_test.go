package getoptions_test

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/DavidGamba/go-getoptions"
)

var dispatchCommandHelpLogger = log.New(ioutil.Discard, "DEBUG: ", log.LstdFlags)

func dispatchCommandHelpListRun(ctx context.Context, opt *getoptions.GetOpt, args []string) error {
	return nil
}

func ExampleGetOpt_Dispatch_cCommandHelp() {
	opt := getoptions.New()
	opt.Bool("help", false, opt.Alias("?"))
	opt.Bool("debug", false)
	opt.SetRequireOrder()
	opt.SetUnknownMode(getoptions.Pass)
	list := opt.NewCommand("list", "list stuff")
	list.SetCommandFn(dispatchCommandHelpListRun)
	list.Bool("list-opt", false)
	opt.HelpCommand("")
	remaining, err := opt.Parse([]string{"help", "list"}) // <- argv set to call command help
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
		os.Exit(1)
	}

	getoptions.Writer = os.Stdout // Print help to stdout instead of stderr for test purpose

	err = opt.Dispatch(context.Background(), "help", remaining)
	if err != nil {
		if errors.Is(err, getoptions.ErrorHelpCalled) {
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
		os.Exit(1)
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
