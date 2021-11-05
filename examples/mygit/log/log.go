package log

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"

	"github.com/DavidGamba/go-getoptions/go-getoptions"
)

var Logger = log.New(ioutil.Discard, "log ", log.LstdFlags)

// New - Populate Options definition
func New(parent *getoptions.GetOpt) *getoptions.GetOpt {
	opt := parent.NewCommand("log", "Show commit logs")
	opt.Bool("log-option", false, opt.Alias("l"))
	return opt
}

// Run - Command entry point
func Run(ctx context.Context, opt *getoptions.GetOpt, args []string) error {
	Logger.Printf("args to log: %v\n", args)
	fmt.Printf("log output...\n")
	if opt.Called("log-option") {
		fmt.Printf("log option was called...\n")
	}
	return nil
}
