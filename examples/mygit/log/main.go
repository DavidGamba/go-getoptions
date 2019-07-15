package log

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/DavidGamba/go-getoptions"
)

var logger = log.New(ioutil.Discard, "log ", log.LstdFlags)

// Opt - Options Definition struct
var Opt *getoptions.GetOpt

// Options - Populate Options definition
func Options() *getoptions.GetOpt {
	Opt = getoptions.New()
	Opt.Self("log", "Show commit logs")
	Opt.Bool("log-option", false, Opt.Alias("l"))
	return Opt
}

// Log - Command entry point
func Log(args []string) {
	remaining, err := Opt.Parse(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
		os.Exit(1)
	}
	if Opt.Called("help") {
		fmt.Fprintf(os.Stderr, Opt.Help())
		os.Exit(1)
	}
	if Opt.Called("debug") {
		logger.SetOutput(os.Stderr)
	}
	logger.Println(remaining)
	fmt.Printf("log output...\n")
}
