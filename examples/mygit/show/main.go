package show

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/DavidGamba/go-getoptions"
)

var logger = log.New(ioutil.Discard, "show ", log.LstdFlags)

// Opt - Options Definition
var Opt *getoptions.GetOpt

// Options - Populate Options definition
func Options() *getoptions.GetOpt {
	Opt = getoptions.New().Self("show", "Show various types of objects")
	Opt.Bool("show-option", false)
	return Opt
}

// Show - Command entry point
func Show(args []string) {
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
	fmt.Printf("show output... %v\n", remaining)
}
