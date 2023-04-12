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
	"regexp"
	"strings"
	"unicode"
	"unsafe"

	"github.com/DavidGamba/dgtools/buildutils"
	"github.com/DavidGamba/dgtools/fsmodtime"
	"github.com/DavidGamba/dgtools/run"
	"github.com/DavidGamba/go-getoptions"
	"github.com/DavidGamba/go-getoptions/dag"
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
		fmt.Fprintf(os.Stderr, "ERROR: failed to open plugin: %s\n", err)
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

// loadOptFns - loads all TaskFn functions from the plugin and adds them as commands to opt.
// If TM task map is defined, add the tasks to the map.
func loadOptFns(ctx context.Context, plug *plugin.Plugin, opt *getoptions.GetOpt, dir string) error {
	var tm **dag.TaskMap
	var tmOk bool
	tmDecl, err := plug.Lookup("TM")
	if err == nil {
		tm, tmOk = tmDecl.(*(*dag.TaskMap))
		// Logger.Printf("tm: %v, Ok: %v\n", tm, tmOk)
		if tmOk {
			*tm = dag.NewTaskMap()
		}
	}

	m := make(map[string]FuncDecl)
	err = GetFuncDeclForPackage(dir, &m)
	if err != nil {
		return fmt.Errorf("failed to inspect package: %w", err)
	}

	// Regex for description: fn-name - description
	re := regexp.MustCompile(`^\w\S+ -`)

	ot := NewOptTree(opt)

	for name, fd := range m {
		// Logger.Printf("inspecting %s\n", name)
		fn, err := plug.Lookup(name)
		if err != nil {
			return fmt.Errorf("failed to find %s function: %w", name, err)
		}
		tfn, ok := fn.(func(*getoptions.GetOpt) getoptions.CommandFn)
		if !ok {
			continue
		}
		description := strings.TrimSpace(fd.Description)
		if description != "" {
			// Logger.Printf("description '%s'\n", description)
			if re.MatchString(description) {
				// Get first word from string
				name = strings.Split(description, " ")[0]
				description = strings.TrimPrefix(description, name+" -")
				description = strings.TrimSpace(description)
			}
		} else {
			name = camelToKebab(name)
		}
		cmd := ot.AddCommand(name, description)
		fnr := tfn(cmd)
		cmd.SetCommandFn(fnr)
		if tmOk {
			// Logger.Printf("adding %s to TM\n", name)
			(*tm).Add(name, fnr)
		}
	}
	return nil
}

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

func findBakeFiles(ctx context.Context) (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}

	// First case, we are withing the bake folder
	base := filepath.Base(wd)
	if base == "bake" {
		err := build(".")
		if err != nil {
			return "", fmt.Errorf("failed to build: %w", err)
		}
		return "./bake.so", nil
	}

	// Second case, bake folder lives in CWD
	dir := filepath.Join(wd, "bake")
	if fi, err := os.Stat(dir); err == nil && fi.Mode().IsDir() {
		err := build(dir)
		if err != nil {
			return "", fmt.Errorf("failed to build: %w", err)
		}
		return filepath.Join(dir, "bake.so"), nil
	}

	// Third case, bake folder lives in module root
	modRoot, err := buildutils.GoModDir()
	if err != nil {
		return "", fmt.Errorf("failed to get go project root: %w", err)
	}
	dir = filepath.Join(modRoot, "bake")
	if fi, err := os.Stat(dir); err == nil && fi.Mode().IsDir() {
		err := build(dir)
		if err != nil {
			return "", fmt.Errorf("failed to build: %w", err)
		}
		return filepath.Join(dir, "bake.so"), nil
	}

	// Fourth case, bake folder lives in root of repo
	root, err := buildutils.GitRepoRoot()
	if err != nil {
		return "", fmt.Errorf("failed to get git repo root: %w", err)
	}
	dir = filepath.Join(root, "bake")
	if fi, err := os.Stat(dir); err == nil && fi.Mode().IsDir() {
		err := build(dir)
		if err != nil {
			return "", fmt.Errorf("failed to build: %w", err)
		}
		return filepath.Join(dir, "bake.so"), nil
	}

	return "", fmt.Errorf("bake directory not found")
}

func build(dir string) error {
	files, modified, err := fsmodtime.Target(os.DirFS(dir),
		[]string{"bake.so"},
		[]string{"*.go", "go.mod", "go.sum"})
	if err != nil {
		return err
	}
	if modified {
		Logger.Printf("Found modifications on %v, rebuilding...\n", files)
		// Debug flags
		// return run.CMD("go", "build", "-buildmode=plugin", "-o=bake.so", "-trimpath", "-gcflags", "all=-N -l").Dir(dir).Log().Run()
		return run.CMD("go", "build", "-buildmode=plugin", "-o=bake.so").Dir(dir).Log().Run()
	}
	return nil
}

func ListSymbolsRun(bakefile string) getoptions.CommandFn {
	return func(ctx context.Context, opt *getoptions.GetOpt, args []string) error {
		plug, err := plugin.Open(bakefile)
		if err != nil {
			return fmt.Errorf("failed to open plugin: %w", err)
		}
		inspectPlugin(plug)
		return nil
	}
}

// https://github.com/golang/go/issues/17823
type Plug struct {
	pluginpath string
	err        string        // set if plugin failed to load
	loaded     chan struct{} // closed when loaded
	syms       map[string]any
}

func inspectPlugin(p *plugin.Plugin) {
	pl := (*Plug)(unsafe.Pointer(p))

	Logger.Printf("Plugin %s exported symbols (%d): \n", pl.pluginpath, len(pl.syms))

	for name, pointers := range pl.syms {
		Logger.Printf("symbol: %s, pointer: %v, type: %v\n", name, pointers, reflect.TypeOf(pointers))
		if _, ok := pointers.(func(*getoptions.GetOpt) getoptions.CommandFn); ok {
			fmt.Printf("name: %s\n", name)
		}
	}
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
