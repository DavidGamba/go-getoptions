package getoptions

import (
	"fmt"
	"strings"
)

// ArgTree - represents the CLI arg tree
type ArgTree []*Arg

func (tree ArgTree) String() string {
	out := ""
	for _, arg := range []*Arg(tree) {
		out += arg.String() + "\n"
	}
	return out
}

// Arg - represents a single string value argument
type Arg struct {
	Value    string
	IsOption bool
	Parent   *Arg
	Children ArgTree
}

func (arg *Arg) String() string {
	out := fmt.Sprintf("Value: %s, IsOption: %v", arg.Value, arg.IsOption)
	cOut := []string{}
	for _, child := range arg.Children {
		cOut = append(cOut, child.String())
	}
	if len(cOut) > 0 {
		out += ", Children: [" + strings.Join(cOut, ", ") + "]"
	}
	return out
}

// ParseArgs -
func ParseArgs(args []string, mode Mode) ArgTree {
	tree := ArgTree{}
	Debug.Printf("ParseArgs: %v\n", args)
	for _, arg := range args {
		a := &Arg{}
		// TODO: isOption should return an option pair, something like option and arg pairs.
		// The reason for it to be allowed to return multiple options is when we have bundling.
		// That would unlock budnling with values for example: -w1024h768
		options, argument, is := isOption(arg, mode)
		Debug.Printf("options: %v, argument: %s, is: %v\n", options, argument, is)
		if is {
			a.Value = options[0]
			a.IsOption = is
			if argument != "" {
				a.Children = append(a.Children, &Arg{Value: argument, IsOption: false, Parent: a})
			}
		} else {
			a.Value = arg
		}
		tree = append(tree, a)
	}
	return tree
}
