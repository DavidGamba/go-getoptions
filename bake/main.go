package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"unicode"

	"github.com/DavidGamba/go-getoptions"
)

var inputArgs []string

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

	inputArgs = args[1:]
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

type TaskDefinitionFn func(ctx context.Context, opt *getoptions.GetOpt) error
type TaskFn func(*getoptions.GetOpt) getoptions.CommandFn

type OptTree struct {
	Root *OptNode
}

type OptNode struct {
	Name     string
	Opt      *getoptions.GetOpt
	Children map[string]*OptNode
}

func NewOptTree(opt *getoptions.GetOpt) *OptTree {
	return &OptTree{
		Root: &OptNode{
			Name:     "root",
			Opt:      opt,
			Children: make(map[string]*OptNode),
		},
	}
}

func (ot *OptTree) AddCommand(name, description string) *getoptions.GetOpt {
	keys := strings.Split(name, ":")
	// Logger.Printf("keys: %v\n", keys)
	node := ot.Root
	var cmd *getoptions.GetOpt
	for i, key := range keys {
		n, ok := node.Children[key]
		if ok {
			// Logger.Printf("key: %v already defined, parent: %s\n", key, node.Name)
			node = n
			cmd = n.Opt
			if len(keys) == i+1 {
				cmd.Self(key, description)
			}
			continue
		}
		// Logger.Printf("key: %v not defined, parent: %s\n", key, node.Name)
		desc := ""
		if len(keys) == i+1 {
			desc = description
		}
		cmd = node.Opt.NewCommand(key, desc)
		node.Children[key] = &OptNode{
			Name:     key,
			Opt:      cmd,
			Children: make(map[string]*OptNode),
		}
		node = node.Children[key]
		if len(keys) == i+1 {
			cmd.SetCommandFn(func(ctx context.Context, opt *getoptions.GetOpt, args []string) error {
				// TODO: Run os.exec call to the built binary with keys as the arguments
				fmt.Printf("Running %v\n", inputArgs)
				return nil
			})
		}
	}
	return cmd
}

func camelToKebab(camel string) string {
	var buffer bytes.Buffer
	for i, ch := range camel {
		if unicode.IsUpper(ch) && i > 0 && !unicode.IsUpper([]rune(camel)[i-1]) {
			buffer.WriteRune('-')
		}
		buffer.WriteRune(unicode.ToLower(ch))
	}
	return buffer.String()
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
