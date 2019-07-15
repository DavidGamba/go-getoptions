package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/DavidGamba/go-getoptions"
	gitlog "github.com/DavidGamba/go-getoptions/examples/mygit/log"
	gitshow "github.com/DavidGamba/go-getoptions/examples/mygit/show"
)

var logger = log.New(ioutil.Discard, "", log.LstdFlags)

func main() {
	opt := getoptions.New()
	opt.Bool("help", false, opt.Alias("?"))
	opt.Bool("debug", false)
	opt.SetRequireOrder()
	opt.SetUnknownMode(getoptions.Pass)
	opt.Command(gitlog.Options().SetOption(opt.Option("help"), opt.Option("debug")))
	opt.Command(gitshow.Options().SetOption(opt.Option("help"), opt.Option("debug")))
	opt.Command(getoptions.New().Self("help", "Show help").CustomCompletion([]string{"log", "show"}))
	remaining, err := opt.Parse(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
		os.Exit(1)
	}
	if opt.Called("debug") {
		logger.SetOutput(os.Stderr)
	}
	logger.Printf("Remaning cli args: %v", remaining)

	// No commands given, defaults to show help
	if len(remaining) == 0 {
		fmt.Fprintf(os.Stderr, opt.Help())
		fmt.Fprintf(os.Stderr, "Use '%s help <command>' for extra details!\n", filepath.Base(os.Args[0]))
		os.Exit(1)
	}

	// First remaining argument is the command
	command, remaining := remaining[0], remaining[1:]

	handleCommand(opt, command, remaining)
}

func handleCommand(opt *getoptions.GetOpt, command string, args []string) {
	switch command {
	case "log":
		gitlog.Log(args)
	case "show":
		gitshow.Show(args)
	case "help":
		if len(args) >= 1 {
			handleCommand(opt, args[0], []string{"--help"})
			os.Exit(1)
		}
		fmt.Fprintln(os.Stderr, opt.Help())
		fmt.Fprintf(os.Stderr, "Use '%s help <command>' for extra details!\n", filepath.Base(os.Args[0]))
		os.Exit(1)
	default:
		fmt.Fprintf(os.Stderr, "ERROR: '%s' is not a git command\n", command)
		os.Exit(1)
	}
}
