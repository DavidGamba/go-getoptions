package getoptions

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/DavidGamba/go-getoptions/internal/option"
)

// User facing tree construction tests.

func setupOpt() *GetOpt {
	opt := New()
	opt.String("rootopt1", "")

	cmd1 := opt.NewCommand("cmd1", "")
	cmd1.String("cmd1opt1", "")
	cmd2 := opt.NewCommand("cmd2", "")
	cmd2.String("cmd2opt1", "")

	sub1cmd1 := cmd1.NewCommand("sub1cmd1", "")
	sub1cmd1.String("sub1cmd1opt1", "")

	sub2cmd1 := cmd1.NewCommand("sub2cmd1", "")
	sub2cmd1.String("-", "")
	return opt
}

func TestStr(t *testing.T) {
	n := setupOpt().programTree
	str := n.str()
	if str.Name != "go-getoptions.test" {
		t.Errorf("wrong value: %s\n", stringPT(n))
	}
}

func TestTrees(t *testing.T) {
	buf := setupLogging()

	t.Run("programTree", func(t *testing.T) {
		root := &programTree{
			Type:            argTypeProgname,
			Name:            filepath.Base(os.Args[0]),
			ChildCommands:   map[string]*programTree{},
			ChildOptions:    map[string]*option.Option{},
			GlobalOptionMap: map[string]string{},
		}
		rootopt1Data := ""
		rootopt1 := option.New("rootopt1", option.StringType, &rootopt1Data)
		cmd1 := &programTree{
			Type:            argTypeCommand,
			Name:            "cmd1",
			Parent:          root,
			ChildCommands:   map[string]*programTree{},
			ChildOptions:    map[string]*option.Option{},
			GlobalOptionMap: map[string]string{},
			Level:           1,
		}
		cmd1opt1Data := ""
		cmd1opt1 := option.New("cmd1opt1", option.StringType, &cmd1opt1Data)
		sub1cmd1 := &programTree{
			Type:            argTypeCommand,
			Name:            "sub1cmd1",
			Parent:          cmd1,
			ChildCommands:   map[string]*programTree{},
			ChildOptions:    map[string]*option.Option{},
			GlobalOptionMap: map[string]string{},
			Level:           2,
		}
		sub1cmd1opt1Data := ""
		sub1cmd1opt1 := option.New("sub1cmd1opt1", option.StringType, &sub1cmd1opt1Data)
		sub2cmd1 := &programTree{
			Type:            argTypeCommand,
			Name:            "sub2cmd1",
			Parent:          cmd1,
			ChildCommands:   map[string]*programTree{},
			ChildOptions:    map[string]*option.Option{},
			GlobalOptionMap: map[string]string{},
			Level:           2,
		}
		sub2cmd1opt1Data := ""
		sub2cmd1opt1 := option.New("-", option.StringType, &sub2cmd1opt1Data)
		cmd2 := &programTree{
			Type:            argTypeCommand,
			Name:            "cmd2",
			Parent:          root,
			ChildCommands:   map[string]*programTree{},
			ChildOptions:    map[string]*option.Option{},
			GlobalOptionMap: map[string]string{},
			Level:           1,
		}
		cmd2opt1Data := ""
		cmd2opt1 := option.New("cmd2opt1", option.StringType, &cmd2opt1Data)

		root.ChildOptions["rootopt1"] = rootopt1
		root.ChildCommands["cmd1"] = cmd1
		root.ChildCommands["cmd2"] = cmd2

		root.GlobalOptionMap["rootopt1"] = "rootopt1"

		// rootopt1Copycmd1 := rootopt1.Copy().SetParent(cmd1)
		rootopt1Copycmd1 := rootopt1
		cmd1.ChildOptions["rootopt1"] = rootopt1Copycmd1
		cmd1.ChildOptions["cmd1opt1"] = cmd1opt1
		cmd1.ChildCommands["sub1cmd1"] = sub1cmd1
		cmd1.ChildCommands["sub2cmd1"] = sub2cmd1

		cmd1.GlobalOptionMap["rootopt1"] = "rootopt1"
		cmd1.GlobalOptionMap["cmd1opt1"] = "cmd1opt1"

		// rootopt1Copycmd2 := rootopt1.Copy().SetParent(cmd2)
		rootopt1Copycmd2 := rootopt1
		cmd2.ChildOptions["rootopt1"] = rootopt1Copycmd2
		cmd2.ChildOptions["cmd2opt1"] = cmd2opt1

		cmd2.GlobalOptionMap["rootopt1"] = "rootopt1"
		cmd2.GlobalOptionMap["cmd2opt1"] = "cmd2opt1"

		// rootopt1Copysub1cmd1 := rootopt1.Copy().SetParent(sub1cmd1)
		rootopt1Copysub1cmd1 := rootopt1
		// cmd1opt1Copysub1cmd1 := cmd1opt1.Copy().SetParent(sub1cmd1)
		cmd1opt1Copysub1cmd1 := cmd1opt1

		sub1cmd1.ChildOptions["rootopt1"] = rootopt1Copysub1cmd1
		sub1cmd1.ChildOptions["cmd1opt1"] = cmd1opt1Copysub1cmd1
		sub1cmd1.ChildOptions["sub1cmd1opt1"] = sub1cmd1opt1

		sub1cmd1.GlobalOptionMap["rootopt1"] = "rootopt1"
		sub1cmd1.GlobalOptionMap["cmd1opt1"] = "cmd1opt1"
		sub1cmd1.GlobalOptionMap["sub1cmd1opt1"] = "sub1cmd1opt1"

		// rootopt1Copysub2cmd1 := rootopt1.Copy().SetParent(sub2cmd1)
		rootopt1Copysub2cmd1 := rootopt1
		// cmd1opt1Copysub2cmd1 := cmd1opt1.Copy().SetParent(sub2cmd1)
		cmd1opt1Copysub2cmd1 := cmd1opt1
		sub2cmd1.ChildOptions["rootopt1"] = rootopt1Copysub2cmd1
		sub2cmd1.ChildOptions["cmd1opt1"] = cmd1opt1Copysub2cmd1
		sub2cmd1.ChildOptions["-"] = sub2cmd1opt1

		sub2cmd1.GlobalOptionMap["rootopt1"] = "rootopt1"
		sub2cmd1.GlobalOptionMap["cmd1opt1"] = "cmd1opt1"
		sub2cmd1.GlobalOptionMap["-"] = "-"

		tree := setupOpt().programTree
		if !reflect.DeepEqual(root, tree) {
			t.Errorf(spewToFileDiff(t, root, tree))
			t.Fatalf(programTreeError(root, tree))
		}

		n, err := getNode(tree)
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}

		if !reflect.DeepEqual(root, n) {
			t.Errorf(spewToFileDiff(t, root, n))
			t.Fatalf(programTreeError(root, tree))
		}

		n, err = getNode(tree, []string{}...)
		if err != nil {
			t.Errorf("unexpected error: %s", err)
		}

		if !reflect.DeepEqual(root, n) {
			t.Errorf(spewToFileDiff(t, root, n))
			t.Fatalf(programTreeError(root, n))
		}

		n, err = getNode(tree, "cmd1")
		if err != nil {
			t.Errorf("unexpected error: %s", err)
		}

		if !reflect.DeepEqual(cmd1, n) {
			t.Errorf(spewToFileDiff(t, cmd1, n))
			t.Fatalf(programTreeError(cmd1, n))
		}

		n, err = getNode(tree, "cmd1", "sub1cmd1")
		if err != nil {
			t.Errorf("unexpected error: %s", err)
		}

		if !reflect.DeepEqual(sub1cmd1, n) {
			t.Errorf(spewToFileDiff(t, sub1cmd1, n))
			t.Fatalf(programTreeError(sub1cmd1, n))
		}

		n, err = getNode(tree, "cmd2")
		if err != nil {
			t.Errorf("unexpected error: %s", err)
		}

		if !reflect.DeepEqual(cmd2, n) {
			t.Errorf(spewToFileDiff(t, cmd2, n))
			t.Fatalf(programTreeError(cmd2, n))
		}

	})

	t.Cleanup(func() { t.Log(buf.String()) })
}

func TestCompletion(t *testing.T) {
	fn := func(ctx context.Context, opt *GetOpt, args []string) error {
		return nil
	}
	called := false
	exitFn = func(code int) { called = true }

	cleanup := func() {
		os.Setenv("COMP_LINE", "")
		completionWriter = os.Stdout
		called = false
	}

	tests := []struct {
		name     string
		setup    func()
		expected string
	}{
		{"option", func() { os.Setenv("COMP_LINE", "test --f") }, "--f\n--flag\n"},
		{"command", func() { os.Setenv("COMP_LINE", "test h") }, "help \n"},
		{"command", func() { os.Setenv("COMP_LINE", "test help ") }, "log\nshow\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			s := ""
			buf := bytes.NewBufferString(s)
			completionWriter = buf

			opt := New()
			opt.Bool("flag", false, opt.Alias("f"))
			logCmd := opt.NewCommand("log", "").SetCommandFn(fn)
			logCmd.NewCommand("sub-log", "").SetCommandFn(fn)
			showCmd := opt.NewCommand("show", "").SetCommandFn(fn)
			showCmd.NewCommand("sub-show", "").SetCommandFn(fn)
			opt.HelpCommand("help", "")
			_, err := opt.Parse([]string{})
			if err != nil {
				t.Errorf("Unexpected error: %s", err)
			}
			if !called {
				t.Errorf("COMP_LINE set and exit wasn't called")
			}
			if buf.String() != tt.expected {
				t.Errorf("Error\ngot: '%s', expected: '%s'\n", buf.String(), tt.expected)
				t.Errorf("diff:\n%s", firstDiff(buf.String(), tt.expected))
			}
			cleanup()
		})
	}
}
