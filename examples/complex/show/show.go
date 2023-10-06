package show

import (
	"context"
	"fmt"
	"io"
	"log"

	"github.com/DavidGamba/go-getoptions"
)

var Logger = log.New(io.Discard, "show ", log.LstdFlags)

// NewCommand - Populate Options definition
func NewCommand(parent *getoptions.GetOpt) *getoptions.GetOpt {
	opt := parent.NewCommand("show", "Show various types of objects")
	opt.Bool("show-option", false)
	opt.String("password", "", opt.GetEnv("PASSWORD"), opt.Alias("p"))
	opt.SetCommandFn(Run)
	return opt
}

// Run - Command entry point
func Run(ctx context.Context, opt *getoptions.GetOpt, args []string) error {
	Logger.Printf("args to show: %v\n", args)
	fmt.Printf("show output... %v\n", args)
	if opt.Called("show-option") {
		fmt.Printf("show option was called...\n")
	}
	if opt.Called("password") {
		fmt.Printf("The secret was... %s\n", opt.Value("password"))
	}
	return nil
}
