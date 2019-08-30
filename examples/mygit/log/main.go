package log

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/DavidGamba/go-getoptions"
)

var logger = log.New(ioutil.Discard, "log ", log.LstdFlags)

// Options - Populate Options definition
func Options() *getoptions.GetOpt {
	opt := getoptions.NewCommand().Self("log", "Show commit logs")
	opt.Bool("log-option", false, opt.Alias("l"))
	return opt
}

// Log - Command entry point
func Log(opt *getoptions.GetOpt, args []string) error {
	remaining, err := opt.Parse(args)
	if err != nil {
		return err
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
	return nil
}
