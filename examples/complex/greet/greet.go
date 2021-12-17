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
	opt := parent.NewCommand("greet", "Subcommands example").SetCommandFn(Run)
	en := opt.NewCommand("en", "greet in English").SetCommandFn(RunEnglish)
	en.String("name", "", opt.Required(""))
	es := opt.NewCommand("es", "greet in Spanish").SetCommandFn(RunSpanish)
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
