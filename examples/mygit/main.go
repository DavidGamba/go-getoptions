package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/DavidGamba/go-getoptions"
	gitlog "github.com/DavidGamba/go-getoptions/examples/mygit/log"
	gitshow "github.com/DavidGamba/go-getoptions/examples/mygit/show"
	gitslow "github.com/DavidGamba/go-getoptions/examples/mygit/slow"
)

var logger = log.New(ioutil.Discard, "", log.LstdFlags)

func main() {
	opt := getoptions.New()
	opt.Bool("help", false, opt.Alias("?"))
	opt.Bool("debug", false)
	opt.SetRequireOrder()
	opt.SetUnknownMode(getoptions.Pass)
	gitlog.New(opt).SetOption(opt.Option("help"), opt.Option("debug")).SetCommandFn(gitlog.Run)
	gitshow.New(opt).SetOption(opt.Option("help"), opt.Option("debug")).SetCommandFn(gitshow.Run)
	gitslow.New(opt).SetOption(opt.Option("help"), opt.Option("debug")).SetCommandFn(gitslow.Run)
	opt.HelpCommand("")
	remaining, err := opt.Parse(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
		os.Exit(1)
	}
	if opt.Called("debug") {
		logger.SetOutput(os.Stderr)
	}
	logger.Printf("Remaning cli args: %v", remaining)

	exitCode := 0
	ctx, cancel, done := opt.InterruptContext()
	defer func() { cancel(); <-done; os.Exit(exitCode) }()

	err = opt.Dispatch(ctx, "help", remaining)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
		exitCode = 1
		return
	}
}
