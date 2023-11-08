package complete

import (
	"context"
	"io"
	"log"
	"strings"

	"github.com/DavidGamba/go-getoptions"
)

var Logger = log.New(io.Discard, "log ", log.LstdFlags)

// NewCommand - Populate Options definition
func NewCommand(parent *getoptions.GetOpt) *getoptions.GetOpt {
	opt := parent.NewCommand("complete", "Example completions")
	opt.SetCommandFn(Run)
	opt.CustomCompletion("dev-east", "dev-west", "staging-east", "prod-east", "prod-west", "prod-south")
	opt.CustomCompletionFn(func(target, s string) []string {
		if strings.HasPrefix("dev-", s) {
			return []string{"dev-hola/", "dev-hello"}
		}
		if strings.HasPrefix("dev-h", s) {
			return []string{"dev-hola/", "dev-hello"}
		}
		if strings.HasPrefix("dev-hello", s) {
			return []string{"dev-hello"}
		}
		if strings.HasPrefix("dev-hola/", s) {
			return []string{"dev-hola/a", "dev-hola/b", "dev-hola/" + target}
		}
		return []string{}
	})
	return opt
}

// Run - Command entry point
func Run(ctx context.Context, opt *getoptions.GetOpt, args []string) error {
	Logger.Printf("args: %v\n", args)
	return nil
}
