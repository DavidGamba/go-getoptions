package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/DavidGamba/go-getoptions"
)

var logger = log.New(os.Stderr, "DEBUG: ", log.LstdFlags)

func main() {
	// Declare the variables you want your options to update
	var debug bool
	var greetCount int

	// Declare the GetOptions object
	opt := getoptions.New()

	// Options definition
	opt.Bool("help", false, opt.Alias("h", "?")) // Aliases can be defined
	opt.BoolVar(&debug, "debug", false)
	opt.IntVar(&greetCount, "greet", 0,
		opt.Required(), // Mark option as required
		opt.Description("Number of times to greet."), // Set the automated help description
		opt.ArgName("number"),                        // Change the help synopsis arg from <int> to <number>
	)
	greetings := opt.StringMap("list", 1, 99,
		opt.Description("Greeting list by language."),
		opt.ArgName("lang=msg"), // Change the help synopsis arg from <key=value> to <lang=msg>
	)

	// Parse cmdline arguments or any provided []string
	remaining, err := opt.Parse(os.Args[1:])

	// Handle help before handling user errors
	if opt.Called("help") {
		fmt.Fprintf(os.Stderr, opt.Help())
		os.Exit(1)
	}

	// Handle user errors
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n\n", err)
		fmt.Fprintf(os.Stderr, opt.HelpSynopsis())
		os.Exit(1)
	}
	if !debug {
		logger.SetOutput(ioutil.Discard)
	}
	logger.Printf("Remaining: %v\n", remaining)

	for i := 0; i < greetCount; i++ {
		fmt.Println("Hello World, from go-getoptions!")
	}
	if len(greetings) > 0 {
		fmt.Printf("Greeting List:\n")
		for k, v := range greetings {
			fmt.Printf("\t%s=%s\n", k, v)
		}
	}
}
