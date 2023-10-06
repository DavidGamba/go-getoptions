package lswrapper

import (
	"context"
	"io"
	"log"
	"os"
	"os/exec"

	"github.com/DavidGamba/go-getoptions"
)

var Logger = log.New(io.Discard, "log ", log.LstdFlags)

// NewCommand - Populate Options definition
func NewCommand(parent *getoptions.GetOpt) *getoptions.GetOpt {
	opt := parent.NewCommand("lswrapper", "wrapper to ls").SetCommandFn(Run)
	opt.SetUnknownMode(getoptions.Pass)
	opt.UnsetOptions()
	return opt
}

// Run - Command entry point
func Run(ctx context.Context, opt *getoptions.GetOpt, args []string) error {
	c := exec.CommandContext(ctx, "ls", args...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	err := c.Run()
	if err != nil {
		return err
	}
	return nil
}
