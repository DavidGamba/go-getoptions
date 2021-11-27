package greet

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/DavidGamba/go-getoptions/go-getoptions"
)

var Logger = log.New(ioutil.Discard, "log ", log.LstdFlags)

// NewCommand - Populate Options definition
func NewCommand(parent *getoptions.GetOpt) *getoptions.GetOpt {
	opt := parent.NewCommand("greet", "Subcommands example")
	// TODO: Auto inherit help command when there are subcommands
	opt.HelpCommand("help", "", opt.Alias("?"))
	opt.SetCommandFn(Run)
	en := opt.NewCommand("en", "greet in English").SetCommandFn(RunEnglish)
	en.String("name", "")
	es := opt.NewCommand("es", "greet in Spanish").SetCommandFn(RunSpanish)
	es.String("name", "")
	return opt
}

// Run - Command entry point
func Run(ctx context.Context, opt *getoptions.GetOpt, args []string) error {
	fmt.Fprint(os.Stderr, opt.Help())
	return getoptions.ErrorHelpCalled
}

func RunEnglish(ctx context.Context, opt *getoptions.GetOpt, args []string) error {
	if opt.Called("name") {
		fmt.Printf("Hello %s!\n", opt.Value("name"))
	}
	fmt.Printf("Hello!\n")
	return nil
}

func RunSpanish(ctx context.Context, opt *getoptions.GetOpt, args []string) error {
	if opt.Called("name") {
		fmt.Printf("Hola %s!\n", opt.Value("name"))
	}
	fmt.Printf("Hola!\n")
	return nil
}
