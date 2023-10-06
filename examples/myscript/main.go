package main

import (
	"fmt"
	"io"
	"log"
	"os"

	"github.com/DavidGamba/go-getoptions"
)

var Logger = log.New(io.Discard, "DEBUG: ", log.LstdFlags)

func main() {
	os.Exit(program(os.Args))
}

func program(args []string) int {
	var debug bool
	var greetCount int
	var list map[string]string
	opt := getoptions.New()
	opt.Self("myscript", "Simple demo script")
	opt.Bool("help", false, opt.Alias("h", "?"))
	opt.BoolVar(&debug, "debug", false, opt.GetEnv("DEBUG"))
	opt.IntVar(&greetCount, "greet", 0,
		opt.Required(),
		opt.Description("Number of times to greet."))
	opt.StringMapVar(&list, "list", 1, 99,
		opt.Description("Greeting list by language."))
	remaining, err := opt.Parse(args[1:])
	if opt.Called("help") {
		fmt.Fprint(os.Stderr, opt.Help())
		return 1
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n\n", err)
		fmt.Fprint(os.Stderr, opt.Help(getoptions.HelpSynopsis))
		return 1
	}

	// Use the passed command line options... Enjoy!
	if debug {
		Logger.SetOutput(os.Stderr)
	}
	Logger.Printf("Unhandled CLI args: %v\n", remaining)

	// Use the int variable
	for i := 0; i < greetCount; i++ {
		fmt.Println("Hello World, from go-getoptions!")
	}

	// Use the map[string]string variable
	if len(list) > 0 {
		fmt.Printf("Greeting List:\n")
		for k, v := range list {
			fmt.Printf("\t%s=%s\n", k, v)
		}
	}
	return 0
}
