package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/DavidGamba/go-getoptions"
	complexgreet "github.com/DavidGamba/go-getoptions/examples/complex/greet"
	complexlog "github.com/DavidGamba/go-getoptions/examples/complex/log"
	complexlswrapper "github.com/DavidGamba/go-getoptions/examples/complex/lswrapper"
	complexshow "github.com/DavidGamba/go-getoptions/examples/complex/show"
	complexslow "github.com/DavidGamba/go-getoptions/examples/complex/slow"
)

var Logger = log.New(ioutil.Discard, "", log.LstdFlags)

func main() {
	os.Exit(program(os.Args))
}

func program(args []string) int {
	// getoptions.Logger.SetOutput(os.Stderr)
	opt := getoptions.New()
	// opt.SetUnknownMode(getoptions.Pass)
	opt.Bool("debug", false, opt.GetEnv("DEBUG"))
	opt.Bool("flag", false)
	opt.Bool("fleg", false)
	opt.String("profile", "default", opt.ValidValues("default", "dev", "staging", "prod"))
	complexgreet.NewCommand(opt)
	complexlog.NewCommand(opt)
	complexlswrapper.NewCommand(opt)
	complexshow.NewCommand(opt)
	complexslow.NewCommand(opt)
	opt.HelpCommand("help", opt.Alias("?"))
	remaining, err := opt.Parse(args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
		return 1
	}
	if opt.Called("debug") {
		Logger.SetOutput(os.Stderr)
		complexgreet.Logger.SetOutput(os.Stderr)
		complexlog.Logger.SetOutput(os.Stderr)
		complexlswrapper.Logger.SetOutput(os.Stderr)
		complexshow.Logger.SetOutput(os.Stderr)
		complexslow.Logger.SetOutput(os.Stderr)
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
