package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/DavidGamba/go-getoptions"
)

var Logger = log.New(os.Stderr, "", log.LstdFlags)

func main() {
	os.Exit(program(os.Args))
}

func program(args []string) int {
	ctx, cancel, done := getoptions.InterruptContext()
	defer func() { cancel(); <-done }()

	opt := getoptions.New()
	opt.Self("myscript", "Simple demo script")
	opt.Bool("debug", false, opt.GetEnv("DEBUG"))
	opt.Int("greet", 0, opt.Required(), opt.Description("Number of times to greet."))
	opt.StringMap("list", 1, 99, opt.Description("Greeting list by language."))
	opt.Bool("quiet", false, opt.GetEnv("QUIET"))
	opt.HelpSynopsisArg("<name>", "Name to greet.")
	opt.SetCommandFn(Run)
	opt.HelpCommand("help", opt.Alias("?"))
	remaining, err := opt.Parse(args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
		return 1
	}
	if opt.Called("quiet") {
		Logger.SetOutput(io.Discard)
	}

	err = opt.Dispatch(ctx, remaining)
	if err != nil {
		if errors.Is(err, getoptions.ErrorHelpCalled) {
			return 1
		}
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
		if errors.Is(err, getoptions.ErrorParsing) {
			fmt.Fprintf(os.Stderr, "\n"+opt.Help())
		}
		return 1
	}
	return 0
}

func Run(ctx context.Context, opt *getoptions.GetOpt, args []string) error {
	// Get arguments and options
	name, _, err := opt.GetRequiredArg(args)
	if err != nil {
		return err
	}
	greetCount := opt.Value("greet").(int)
	list := opt.Value("list").(map[string]string)

	Logger.Printf("Running: %v", args)

	// Use the int variable
	for i := 0; i < greetCount; i++ {
		fmt.Printf("Hello %s, from go-getoptions!\n", name)
	}

	// Use the map[string]string variable
	if len(list) > 0 {
		fmt.Printf("Greeting List:\n")
		for k, v := range list {
			fmt.Printf("\t%s=%s\n", k, v)
		}
	}

	return nil
}
