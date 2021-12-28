package greet

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/DavidGamba/go-getoptions"
)

var Logger = log.New(ioutil.Discard, "log ", log.LstdFlags)

// NewCommand - Populate Options definition
func NewCommand(parent *getoptions.GetOpt) *getoptions.GetOpt {
	opt := parent.NewCommand("message", "Subcommands example").SetCommandFn(Run)
	GreetNewCommand(opt)
	ByeNewCommand(opt)
	return opt
}

func GreetNewCommand(parent *getoptions.GetOpt) *getoptions.GetOpt {
	opt := parent.NewCommand("greet", "Hi in multiple languages").SetCommandFn(Run)
	en := opt.NewCommand("en", "greet in English").SetCommandFn(RunEnglish)
	en.String("name", "", opt.Required(""))
	es := opt.NewCommand("es", "greet in Spanish").SetCommandFn(RunSpanish)
	es.String("name", "", opt.Required(""))
	return opt
}

func ByeNewCommand(parent *getoptions.GetOpt) *getoptions.GetOpt {
	opt := parent.NewCommand("bye", "Bye in multiple languages").SetCommandFn(Run)
	en := opt.NewCommand("en", "bye in English").SetCommandFn(RunByeEnglish)
	en.String("name", "", opt.Required(""))
	es := opt.NewCommand("es", "bye in Spanish").SetCommandFn(RunByeSpanish)
	es.String("name", "", opt.Required(""))
	return opt
}

// Run - Command entry point
func Run(ctx context.Context, opt *getoptions.GetOpt, args []string) error {
	fmt.Fprint(os.Stderr, opt.Help())
	return getoptions.ErrorHelpCalled
}

func RunEnglish(ctx context.Context, opt *getoptions.GetOpt, args []string) error {
	fmt.Printf("Hello %s!\n", opt.Value("name"))
	return nil
}

func RunSpanish(ctx context.Context, opt *getoptions.GetOpt, args []string) error {
	fmt.Printf("Hola %s!\n", opt.Value("name"))
	return nil
}

func RunByeEnglish(ctx context.Context, opt *getoptions.GetOpt, args []string) error {
	fmt.Printf("Bye %s!\n", opt.Value("name"))
	return nil
}

func RunByeSpanish(ctx context.Context, opt *getoptions.GetOpt, args []string) error {
	fmt.Printf("Adios %s!\n", opt.Value("name"))
	return nil
}
