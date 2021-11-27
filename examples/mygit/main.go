package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	gitlog "github.com/DavidGamba/go-getoptions/examples/mygit/log"
	gitshow "github.com/DavidGamba/go-getoptions/examples/mygit/show"
	gitslow "github.com/DavidGamba/go-getoptions/examples/mygit/slow"
	"github.com/DavidGamba/go-getoptions/go-getoptions"
)

var Logger = log.New(ioutil.Discard, "", log.LstdFlags)

func main() {
	os.Exit(program(os.Args))
}

func program(args []string) int {
	opt := getoptions.New()
	opt.Bool("help", false, opt.Alias("?"))
	opt.Bool("debug", false, opt.GetEnv("DEBUG"))
	opt.String("profile", "default", opt.ValidValues("default", "dev", "staging", "prod"))
	opt.SetUnknownMode(getoptions.Pass)
	gitlog.NewCommand(opt)
	gitshow.NewCommand(opt)
	gitslow.NewCommand(opt)
	opt.HelpCommand("help", "")
	remaining, err := opt.Parse(args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
		return 1
	}
	if opt.Called("debug") {
		Logger.SetOutput(os.Stderr)
		gitlog.Logger.SetOutput(os.Stderr)
		gitshow.Logger.SetOutput(os.Stderr)
		gitslow.Logger.SetOutput(os.Stderr)
	}
	if opt.Called("profile") {
		Logger.Printf("profile: %s\n", opt.Value("profile"))
	}
	Logger.Printf("Remaning cli args: %v", remaining)

	ctx, cancel, done := getoptions.InterruptContext()
	defer func() { cancel(); <-done }()

	err = opt.Dispatch(ctx, remaining)
	if err != nil {
		if errors.Is(err, getoptions.ErrorHelpCalled) {
			return 1
		}
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
		return 1
	}
	return 0
}
