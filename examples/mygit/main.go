package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/DavidGamba/go-getoptions"
	gitlog "github.com/DavidGamba/go-getoptions/examples/mygit/log"
	gitshow "github.com/DavidGamba/go-getoptions/examples/mygit/show"
)

var logger = log.New(ioutil.Discard, "", log.LstdFlags)

func main() {
	opt := getoptions.New()
	opt.Bool("help", false, opt.Alias("?"))
	opt.Bool("debug", false)
	opt.SetRequireOrder()
	opt.SetUnknownMode(getoptions.Pass)
	opt.Command(gitlog.Options().SetOption(opt.Option("help"), opt.Option("debug")).SetCommandFn(gitlog.Log))
	opt.Command(gitshow.Options().SetOption(opt.Option("help"), opt.Option("debug")).SetCommandFn(gitshow.Show))
	opt.Command(opt.HelpCommand(""))
	remaining, err := opt.Parse(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
		os.Exit(1)
	}
	if opt.Called("debug") {
		logger.SetOutput(os.Stderr)
	}
	logger.Printf("Remaning cli args: %v", remaining)

	err = opt.Dispatch("help", remaining)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
		os.Exit(1)
	}
}
