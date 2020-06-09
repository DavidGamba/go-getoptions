package show

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/DavidGamba/go-getoptions"
)

var logger = log.New(ioutil.Discard, "show ", log.LstdFlags)

// New - Populate Options definition
func New(parent *getoptions.GetOpt) *getoptions.GetOpt {
	opt := parent.NewCommand("show", "Show various types of objects")
	opt.Bool("show-option", false)
	opt.String("password", "", opt.GetEnv("PASSWORD"), opt.Alias("p"))
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
	fmt.Printf("show output... %v\n", args)
	return nil
}
