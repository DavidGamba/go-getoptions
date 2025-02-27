package complete

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/DavidGamba/go-getoptions"
)

var Logger = log.New(io.Discard, "log ", log.LstdFlags)

// NewCommand - Populate Options definition
func NewCommand(parent *getoptions.GetOpt) *getoptions.GetOpt {
	opt := parent.NewCommand("complete", "Example completions")
	opt.SetCommandFn(Run)
	opt.String("completeme", "", opt.SuggestedValuesFn(func(target, partial string) []string {
		fmt.Fprintf(os.Stderr, "\npartial: %v\n", partial)
		return []string{"complete", "completeme", "completeme2"}
	}))
	opt.ArgCompletions("dev-east", "dev-west", "staging-east", "prod-east", "prod-west", "prod-south")
	opt.ArgCompletionsFns(func(target string, prev []string, s string) []string {
		if len(prev) == 0 {
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
		}
		if len(prev) == 1 {
			Logger.Printf("prev: %v\n", prev)
			if strings.HasPrefix(prev[0], "dev-hola/") {
				return []string{"second-hola/a", "second-hola/b", "second-hola/" + target}
			}
			return []string{"second-arg-a", "second-arg-b"}
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
