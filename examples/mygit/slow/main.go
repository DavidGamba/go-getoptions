package slow

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/DavidGamba/go-getoptions"
)

var logger = log.New(ioutil.Discard, "show ", log.LstdFlags)

var iterations int

// New - Populate Options definition
func New(parent *getoptions.GetOpt) *getoptions.GetOpt {
	opt := parent.NewCommand("slow", "Run something in a very slow way (please cancel me with Ctrl-C)")
	opt.IntVar(&iterations, "iterations", 5)
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
