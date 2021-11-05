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

var logger = log.New(ioutil.Discard, "", log.LstdFlags)

func main() {
	os.Exit(program())
}

func program() int {
	opt := getoptions.New()
	opt.Bool("help", false, opt.Alias("?"))
	opt.Bool("debug", false, opt.GetEnv("DEBUG"))
	opt.String("profile", "default")
	opt.SetUnknownMode(getoptions.Pass)
	gitlog.New(opt).SetCommandFn(gitlog.Run)
	gitshow.New(opt).SetCommandFn(gitshow.Run)
	gitslow.New(opt).SetCommandFn(gitslow.Run)
	opt.HelpCommand("")
	remaining, err := opt.Parse(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
		os.Exit(1)
	}
	if opt.Called("debug") {
		logger.SetOutput(os.Stderr)
	}
	if opt.Called("profile") {
		logger.Printf("profile: %s\n", opt.Value("profile"))
	}
	logger.Printf("Remaning cli args: %v", remaining)

	ctx, cancel, done := getoptions.InterruptContext()
	defer func() { cancel(); <-done }()

	err = opt.Dispatch(ctx, remaining)
	if err != nil {
		// if errors.Is(err, getoptions.ErrorHelpCalled) {
		// 	return 1
		// }
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
		return 1
	}
	return 0
}
