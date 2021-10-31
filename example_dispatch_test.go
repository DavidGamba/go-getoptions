package getoptions_test

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/DavidGamba/go-getoptions"
)

func ExampleGetOpt_Dispatch() {
	runFn := func(ctx context.Context, opt *getoptions.GetOpt, args []string) error {
		fmt.Println("a, b, c, d")
		return nil
	}

	opt := getoptions.New()
	opt.NewCommand("list", "list stuff").SetCommandFn(runFn)
	opt.HelpCommand("help", opt.Alias("?"))
	remaining, err := opt.Parse([]string{"list"}) // <- argv set to call command
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
		os.Exit(1)
	}

	getoptions.Writer = os.Stdout // Print help to stdout instead of stderr for test purpose

	err = opt.Dispatch(context.Background(), remaining)
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
