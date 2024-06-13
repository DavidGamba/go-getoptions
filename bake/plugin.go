package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"plugin"
	"regexp"
	"strings"

	"github.com/DavidGamba/go-getoptions"
	"github.com/DavidGamba/go-getoptions/dag"
)

func loadPlugin(ctx context.Context) (string, *plugin.Plugin, error) {
	dir, err := findBakeDir(ctx)
	if err != nil {
		return "", nil, err
	}

	err = buildPlugin(dir)
	if err != nil {
		return "", nil, fmt.Errorf("failed to build: %w", err)
	}
	bakefile := filepath.Join(dir, "bake.so")

	plug, err := plugin.Open(bakefile)
	if err != nil {
		_ = os.Remove(bakefile)
		return bakefile, nil, fmt.Errorf("failed to open plugin, try again: %w", err)
	}
	return bakefile, plug, nil
}

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
		cmd := ot.AddCommand(name, name, description)
		fnr := tfn(cmd)
		cmd.SetCommandFn(fnr)
		if tmOk {
			// Logger.Printf("adding %s to TM\n", name)
			(*tm).Add(name, fnr)
		}
	}
	return nil
}
