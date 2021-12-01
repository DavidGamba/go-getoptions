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

var dispatchLogger = log.New(ioutil.Discard, "DEBUG: ", log.LstdFlags)

func listRun(ctx context.Context, opt *getoptions.GetOpt, args []string) error {
	fmt.Println("a, b, c, d")
	return nil
}

func ExampleGetOpt_Dispatch() {
	opt := getoptions.New()
	opt.Bool("help", false, opt.Alias("?"))
	opt.Bool("debug", false)
	opt.SetRequireOrder()
	opt.SetUnknownMode(getoptions.Pass)
	list := opt.NewCommand("list", "list stuff").SetCommandFn(listRun)
	list.Bool("list-opt", false)
	opt.HelpCommand("")
	remaining, err := opt.Parse([]string{"list"}) // <- argv set to call command
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
	// a, b, c, d
}
