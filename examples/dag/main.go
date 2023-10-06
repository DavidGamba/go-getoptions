package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/DavidGamba/go-getoptions"
	"github.com/DavidGamba/go-getoptions/dag"
)

var TM *dag.TaskMap

var Logger = log.New(os.Stderr, "", log.LstdFlags)

func main() {
	os.Exit(program(os.Args))
}

func program(args []string) int {
	opt := getoptions.New()
	opt.Bool("quiet", false)
	opt.Bool("dot", false, opt.Description("Generate graphviz dot diagram"))
	opt.SetUnknownMode(getoptions.Pass)
	opt.NewCommand("build", "build project artifacts").SetCommandFn(Build)
	opt.NewCommand("clean", "clean project artifacts").SetCommandFn(Clean)
	opt.HelpCommand("help", opt.Alias("?"))
	remaining, err := opt.Parse(args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
		return 1
	}
	if opt.Called("quiet") {
		Logger.SetOutput(io.Discard)
	}

	ctx, cancel, done := getoptions.InterruptContext()
	defer func() { cancel(); <-done }()

	TM = dag.NewTaskMap()
	TM.Add("bt1", buildTask1)
	TM.Add("bt2", buildTask2)
	TM.Add("bt3", buildTask3)
	TM.Add("ct1", cleanTask1)
	TM.Add("ct2", cleanTask2)
	TM.Add("ct3", cleanTask3)

	err = opt.Dispatch(ctx, remaining)
	if err != nil {
		if errors.Is(err, getoptions.ErrorHelpCalled) {
			return 1
		}
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
		if errors.Is(err, getoptions.ErrorParsing) {
			fmt.Fprintf(os.Stderr, "\n"+opt.Help())
		}
		return 1
	}
	return 0
}

func Build(ctx context.Context, opt *getoptions.GetOpt, args []string) error {
	Logger.Printf("Running build command")
	g := dag.NewGraph("build graph")
	g.TaskDependensOn(TM.Get("bt3"), TM.Get("bt1"), TM.Get("bt2"))
	err := g.Validate(TM)
	if err != nil {
		return err
	}
	if opt.Called("dot") {
		fmt.Printf("%s\n", g)
		return nil
	}
	return g.Run(ctx, opt, args)
}

func Clean(ctx context.Context, opt *getoptions.GetOpt, args []string) error {
	Logger.Printf("Running clean command")
	g := dag.NewGraph("clean graph")
	g.TaskDependensOn(TM.Get("ct1"), TM.Get("ct3"))
	g.TaskDependensOn(TM.Get("ct2"), TM.Get("ct3"))
	err := g.Validate(TM)
	if err != nil {
		return err
	}
	if opt.Called("dot") {
		fmt.Printf("%s\n", g)
		return nil
	}
	return g.Run(ctx, opt, args)
}

func buildTask1(ctx context.Context, opt *getoptions.GetOpt, args []string) error {
	fmt.Printf("Build first artifact\n")
	time.Sleep(1 * time.Second)
	return nil
}

func buildTask2(ctx context.Context, opt *getoptions.GetOpt, args []string) error {
	fmt.Printf("Build second artifact\n")
	time.Sleep(1 * time.Second)
	return nil
}

func buildTask3(ctx context.Context, opt *getoptions.GetOpt, args []string) error {
	fmt.Printf("Build third artifact, depends on 1 and 2\n")
	time.Sleep(1 * time.Second)
	return nil
}

func cleanTask1(ctx context.Context, opt *getoptions.GetOpt, args []string) error {
	fmt.Printf("Clean first artifact, 3 must not exist\n")
	time.Sleep(1 * time.Second)
	return nil
}

func cleanTask2(ctx context.Context, opt *getoptions.GetOpt, args []string) error {
	fmt.Printf("Clean second artifact, 3 must not exist\n")
	time.Sleep(1 * time.Second)
	return nil
}

func cleanTask3(ctx context.Context, opt *getoptions.GetOpt, args []string) error {
	fmt.Printf("Clean third artifact\n")
	time.Sleep(1 * time.Second)
	return nil
}
