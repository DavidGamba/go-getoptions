package getoptions

import (
	"reflect"
	"testing"
	// "github.com/davecgh/go-spew/spew"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name string
		args []string
		tree ArgTree
	}{
		{"single", []string{"str"}, ArgTree{
			{Value: "str", IsOption: false},
		}},
		{"single", []string{"--opt"}, ArgTree{
			{Value: "opt", IsOption: true},
		}},
		{"single", []string{"--opt=hello"}, func() ArgTree {
			a := &Arg{Value: "opt", IsOption: true}
			a.Children = ArgTree{{Value: "hello", IsOption: false, Parent: a}}
			return ArgTree{a}
		}()},
		{"multi", []string{"str", "str2"}, ArgTree{
			{Value: "str", IsOption: false},
			{Value: "str2", IsOption: false},
		}},
		{"multi", []string{"--opt", "--opt2"}, ArgTree{
			{Value: "opt", IsOption: true},
			{Value: "opt2", IsOption: true},
		}},
		{"multi", []string{"--opt=hello", "--opt2=hola"}, func() ArgTree {
			a := &Arg{Value: "opt", IsOption: true}
			a.Children = ArgTree{{Value: "hello", IsOption: false, Parent: a}}
			b := &Arg{Value: "opt2", IsOption: true}
			b.Children = ArgTree{{Value: "hola", IsOption: false, Parent: b}}
			return ArgTree{a, b}
		}()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := setupLogging()
			tree := ParseArgs(tt.args, Normal)
			if !reflect.DeepEqual(tree, tt.tree) {
				t.Errorf("Expected:\n%s\nGot:\n%s\n", tt.tree.String(), tree.String())
				// spew.Dump(tt.tree)
				// spew.Dump(tree)
			}
			t.Log(buf.String())
		})
	}
}
