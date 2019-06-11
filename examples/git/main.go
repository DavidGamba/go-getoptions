package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/DavidGamba/go-getoptions"
	gitlog "github.com/DavidGamba/go-getoptions/examples/git/log"
	gitshow "github.com/DavidGamba/go-getoptions/examples/git/show"
)

var logger = log.New(ioutil.Discard, "", log.LstdFlags)

func synopsis() {
	synopsis := `git [--help] <command> <args>

Commands:

	log        Show commit logs
	show       Show various types of objects
`
	fmt.Fprintln(os.Stderr, synopsis)
}

var commandList = []string{
	"log",
	"show",
}

func contains(s []string, x string) bool {
	for _, e := range s {
		if x == e {
			return true
		}
	}
	return false
}

func main() {
	opt := getoptions.New()
	opt.Bool("help", false, opt.Alias("?"))
	opt.Bool("debug", false)
	opt.SetRequireOrder()
	opt.SetUnknownMode("pass")
	remaining, err := opt.Parse(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
		os.Exit(1)
	}
	if opt.Called("debug") {
		logger.SetOutput(os.Stderr)
	}
	log.Println(remaining)

	if len(remaining) == 0 {
		synopsis()
		os.Exit(1)
	}
	command := remaining[0]
	remaining = remaining[1:]

	if opt.Called("help") && contains(commandList, command) {
		remaining = []string{"--help"}
	} else if command == "help" && len(remaining) >= 1 && contains(commandList, remaining[0]) {
		command = remaining[0]
		remaining[0] = "--help"
	} else if opt.Called("help") || command == "help" {
		synopsis()
		os.Exit(1)
	}

	switch command {
	case "log":
		gitlog.Log(remaining)
	case "show":
		gitshow.Show(remaining)
	default:
		fmt.Fprintf(os.Stderr, "ERROR: '%s' is not a git command\n", command)
		os.Exit(1)
	}
}
