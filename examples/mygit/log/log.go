package log

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/DavidGamba/go-getoptions"
)

var logger = log.New(ioutil.Discard, "log ", log.LstdFlags)

// New - Populate Options definition
func New(parent *getoptions.GetOpt) *getoptions.GetOpt {
	opt := parent.NewCommand("log", "Show commit logs")
	opt.Bool("log-option", false, opt.Alias("l"))
	return opt
}

// Run - Command entry point
func Run(ctx context.Context, opt *getoptions.GetOpt, args []string) error {
	if opt.Called("help") {
		fmt.Fprintf(os.Stderr, opt.Help())
		os.Exit(1)
	}
	if opt.Called("debug") {
		logger.SetOutput(os.Stderr)
	}
	logger.Println(args)
	fmt.Printf("log output...\n")
	return nil
}
