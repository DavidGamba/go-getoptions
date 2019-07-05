package show

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/DavidGamba/go-getoptions"
)

var logger = log.New(ioutil.Discard, "show ", log.LstdFlags)

func Options() *getoptions.GetOpt {
	opt := getoptions.New()
	opt.Self("show", "Show various types of objects")
	opt.Bool("help", false, opt.Alias("?"))
	opt.Bool("debug", false)
	opt.Bool("show-option", false)
	return opt
}

func Show(args []string) {
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
	fmt.Printf("show output...\n")
}
