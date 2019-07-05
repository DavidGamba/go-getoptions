package log

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/DavidGamba/go-getoptions"
)

var logger = log.New(ioutil.Discard, "log ", log.LstdFlags)

func Options() *getoptions.GetOpt {
	opt := getoptions.New()
	opt.Self("log", "Show commit logs")
	opt.Bool("help", false, opt.Alias("?"))
	opt.Bool("debug", false)
	opt.Bool("log-option", false, opt.Alias("l"))
	return opt
}

func Log(args []string) {
	opt := Options()
	remaining, err := opt.Parse(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
		os.Exit(1)
	}
	if opt.Called("help") {
		fmt.Fprintf(os.Stderr, opt.Help())
		os.Exit(1)
	}
	if opt.Called("debug") {
		logger.SetOutput(os.Stderr)
	}
	logger.Println(remaining)
	fmt.Printf("log output...\n")
}
