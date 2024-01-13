package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"log"
	"os"
	"path/filepath"
	"plugin"
	"reflect"
	"strings"
	"unicode"

	"github.com/DavidGamba/go-getoptions"
	"golang.org/x/tools/go/packages"
)

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

	bakefile, err := findBakeFiles(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
		return 1
	}

	plug, err := plugin.Open(bakefile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: failed to open plugin, try again: %s\n", err)
		_ = os.Remove(bakefile)
		return 1
	}

	// err = loadAndRunTaskDefinitionFn(ctx, plug, opt)
	// if err != nil {
	// 	fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
	// 	return 1
	// }

	err = loadOptFns(ctx, plug, opt, filepath.Dir(bakefile))
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
		return 1
	}

	b := opt.NewCommand("_bake", "")

	bls := b.NewCommand("list-symbols", "lists symbols")
	bls.SetCommandFn(ListSymbolsRun(bakefile))

	bld := b.NewCommand("list-descriptions", "lists descriptions")
	bld.SetCommandFn(ListDescriptionsRun(bakefile))

	opt.HelpCommand("help", opt.Alias("?"))
	remaining, err := opt.Parse(args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
		return 1
	}
	if opt.Called("quiet") {
		logger, err := plug.Lookup("Logger")
		if err == nil {
			var l **log.Logger
			l, ok := logger.(*(*log.Logger))
			if ok {
				(*l).SetOutput(io.Discard)
			} else {
				Logger.Printf("failed to convert Logger: %s\n", reflect.TypeOf(logger))
			}
		} else {
			Logger.Printf("failed to find Logger\n")
		}
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

func ListDescriptionsRun(bakefile string) getoptions.CommandFn {
	return func(ctx context.Context, opt *getoptions.GetOpt, args []string) error {
		Logger.Printf("bakefile: %s\n", bakefile)
		dir := filepath.Dir(bakefile)
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

type FuncDecl struct {
	Description string
}

// m: map of function name to function information
func GetFuncDeclForPackage(dir string, m *map[string]FuncDecl) error {
	if m == nil {
		return fmt.Errorf("map is nil")
	}
	cfg := &packages.Config{Mode: packages.NeedFiles | packages.NeedSyntax, Dir: dir}
	pkgs, err := packages.Load(cfg, ".")
	if err != nil {
		return fmt.Errorf("failed to load packages: %w", err)
	}
	for _, pkg := range pkgs {
		// Logger.Println(pkg.ID, pkg.GoFiles)
		for _, file := range pkg.GoFiles {
			// Logger.Printf("file: %s\n", file)
			// parse file
			fset := token.NewFileSet()
			fset.AddFile(file, fset.Base(), len(file))
			f, err := parser.ParseFile(fset, file, nil, parser.ParseComments)
			if err != nil {
				return fmt.Errorf("failed to parse file: %w", err)
			}
			// inspect file
			ast.Inspect(f, func(n ast.Node) bool {
				switch x := n.(type) {
				case *ast.FuncDecl:
					if x.Name.IsExported() {
						name := x.Name.Name
						description := x.Doc.Text()
						(*m)[name] = FuncDecl{Description: description}
					}
				}
				return true
			})
		}
	}
	return nil
}
