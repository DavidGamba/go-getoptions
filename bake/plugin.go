package main

import (
	"context"
	"fmt"
	"plugin"
	"reflect"
	"regexp"
	"strings"
	"unsafe"

	"github.com/DavidGamba/go-getoptions"
	"github.com/DavidGamba/go-getoptions/dag"
)

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
