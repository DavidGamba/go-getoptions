package getoptions_test

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/DavidGamba/go-getoptions"
)

var logger = log.New(ioutil.Discard, "DEBUG: ", log.LstdFlags)

func Example() {
	// Declare the variables you want your options to update
	var debug bool
	var greetCount int
	var list map[string]string

	// Declare the GetOptions object
	opt := getoptions.New()

	// Options definition
	opt.Bool("help", false, opt.Alias("h", "?")) // Aliases can be defined
	opt.BoolVar(&debug, "debug", false)
	opt.IntVar(&greetCount, "greet", 0,
		opt.Required(),
		opt.Description("Number of times to greet."), // Set the automated help description
		opt.ArgName("number"),                        // Change the help synopsis arg from the default <int> to <number>
	)
	opt.StringMapVar(&list, "list", 1, 99,
		opt.Description("Greeting list by language."),
		opt.ArgName("lang=msg"), // Change the help synopsis arg from <key=value> to <lang=msg>
	)

	// // Parse cmdline arguments os.Args[1:]
	remaining, err := opt.Parse([]string{"-g", "2", "-l", "en='Hello World'", "es='Hola Mundo'"})

	// Handle help before handling user errors
	if opt.Called("help") {
		fmt.Fprint(os.Stderr, opt.Help())
		os.Exit(1)
	}

	// Handle user errors
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n\n", err)
		fmt.Fprint(os.Stderr, opt.Help(getoptions.HelpSynopsis))
		os.Exit(1)
	}

	// Use the passed command line options... Enjoy!
	if debug {
		logger.SetOutput(os.Stderr)
	}
	logger.Printf("Unhandled CLI args: %v\n", remaining)

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

	// Unordered output:
	// Hello World, from go-getoptions!
	// Hello World, from go-getoptions!
	// Greeting List:
	//	en='Hello World'
	//	es='Hola Mundo'
}
