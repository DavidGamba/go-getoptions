package main

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/DavidGamba/dgtools/run"
	"github.com/DavidGamba/go-getoptions"
)

// type TaskDefinitionFn func(ctx context.Context, opt *getoptions.GetOpt) error
// type TaskFn func(*getoptions.GetOpt) getoptions.CommandFn

type OptTree struct {
	Root *OptNode
}

type OptNode struct {
	Name        string
	Opt         *getoptions.GetOpt
	Children    map[string]*OptNode
	Parent      string
	Description string
	FnName      string
}

func NewOptTree(opt *getoptions.GetOpt) *OptTree {
	return &OptTree{
		Root: &OptNode{
			Name:        "",
			Parent:      "",
			Opt:         opt,
			Description: "",
			Children:    make(map[string]*OptNode),
			FnName:      "",
		},
	}
}

// Regex for description: fn-name - description
var descriptionRe = regexp.MustCompile(`^\w\S+ -`)

func (ot *OptTree) AddCommand(fnName, name, description string) *getoptions.GetOpt {
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
			Name:        key,
			Parent:      node.Name,
			Opt:         cmd,
			Children:    make(map[string]*OptNode),
			Description: desc,
			FnName:      fnName,
		}
		node = node.Children[key]
		if len(keys) == i+1 {
			cmd.SetCommandFn(func(ctx context.Context, opt *getoptions.GetOpt, args []string) error {
				// TODO: Run os.exec call to the built binary with keys as the arguments
				fmt.Printf("Running %v\n", InputArgs)
				c := []string{"./bake"}
				run.CMD(append(c, InputArgs...)...).Dir(Dir).Run()
				return nil
			})
		}
	}
	return cmd
}

func (ot *OptTree) String() string {
	return ot.Root.String()
}

func (on *OptNode) String() string {
	out := ""
	parent := on.Parent
	if parent == "" {
		parent = "opt"
	}
	if on.Name != "" {
		out += fmt.Sprintf("%s := %s.NewCommand(\"%s\", \"%s\")\n", on.Name, parent, on.Name, on.Description)
		out += fmt.Sprintf("%s.SetCommandFn(%s(%s))\n", on.Name, on.FnName, on.Name)
	}
	for _, child := range on.Children {
		out += child.String()
	}
	return out
}
