package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/DavidGamba/go-getoptions"
)

var InputArgs []string
var Dir string

var Logger = log.New(os.Stderr, "", log.LstdFlags)

func main() {
	os.Exit(program(os.Args))
}

func program(args []string) int {
	ctx, cancel, done := getoptions.InterruptContext()
	defer func() { cancel(); <-done }()

	opt := getoptions.New()
	opt.Self("bake", "Go Build + Something like Make = Bake ¯\\_(ツ)_/¯")
	opt.SetUnknownMode(getoptions.Pass)
	opt.Bool("quiet", false, opt.GetEnv("QUIET"))

	dir, err := findBakeDir(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
		return 1
	}
	Dir = dir

	// bakefile, plug, err := loadPlugin(ctx)
	// if err != nil {
	// 	fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
	// 	return 1
	// }

	// err = loadOptFns(ctx, plug, opt, filepath.Dir(bakefile))
	// if err != nil {
	// 	fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
	// 	return 1
	// }

	InputArgs = args[1:]
	err = LoadAst(ctx, opt, dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
		return 1
	}

	b := opt.NewCommand("_bake", "")

	bld := b.NewCommand("list-descriptions", "lists descriptions")
	bld.SetCommandFn(ListDescriptionsRun(dir))

	bast := b.NewCommand("show-ast", "show raw-ish ast")
	bast.SetCommandFn(ShowASTRun(dir))

	bastList := b.NewCommand("list-ast", "list parsed ast")
	bastList.SetCommandFn(LoadASTRun(dir))

	opt.HelpCommand("help", opt.Alias("?"))
	remaining, err := opt.Parse(args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
		return 1
	}
	if opt.Called("quiet") {
		Logger.SetOutput(io.Discard)
	}

	err = opt.Dispatch(ctx, remaining)
	if err != nil {
		if errors.Is(err, getoptions.ErrorHelpCalled) {
			return 1
		}
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
		return 1
	}
	return 0
}

func ListDescriptionsRun(dir string) getoptions.CommandFn {
	return func(ctx context.Context, opt *getoptions.GetOpt, args []string) error {
		m := make(map[string]FuncDecl)
		err := GetFuncDeclForPackage(dir, &m)
		if err != nil {
			return fmt.Errorf("failed to inspect package: %w", err)
		}
		for name, fd := range m {
			fmt.Printf("%s: %s\n", name, fd.Description)
		}

		return nil
	}
}

func ShowASTRun(dir string) getoptions.CommandFn {
	return func(ctx context.Context, opt *getoptions.GetOpt, args []string) error {
		err := PrintAst(dir)
		if err != nil {
			return fmt.Errorf("failed to inspect package: %w", err)
		}
		return nil
	}
}

func LoadASTRun(dir string) getoptions.CommandFn {
	return func(ctx context.Context, opt *getoptions.GetOpt, args []string) error {
		err := LoadAst(ctx, opt, dir)
		if err != nil {
			return fmt.Errorf("failed to inspect package: %w", err)
		}
		return nil
	}
}
