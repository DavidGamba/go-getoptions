package slow

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"time"

	"github.com/DavidGamba/go-getoptions/go-getoptions"
)

var Logger = log.New(ioutil.Discard, "show ", log.LstdFlags)

var iterations int

// NewCommand - Populate Options definition
func NewCommand(parent *getoptions.GetOpt) *getoptions.GetOpt {
	opt := parent.NewCommand("slow", "Run something in a very slow way (please cancel me with Ctrl-C)")
	opt.IntVar(&iterations, "iterations", 5)
	opt.SetCommandFn(Run)
	return opt
}

// Run - Command entry point
func Run(ctx context.Context, opt *getoptions.GetOpt, args []string) error {
	Logger.Printf("args to slow: %v\n", args)
	if opt.Called("iterations") {
		fmt.Printf("iterations overriden with: %d\n", opt.Value("iterations"))
	}
	for i := 0; i < iterations; i++ {
		select {
		case <-ctx.Done():
			fmt.Println("Cleaning up...")
			return nil
		default:
		}
		fmt.Printf("Sleeping: %d\n", i)
		time.Sleep(1 * time.Second)
	}
	return nil
}
