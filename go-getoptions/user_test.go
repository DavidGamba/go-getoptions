package getoptions

import (
	"os"
	"reflect"
	"testing"

	"github.com/DavidGamba/go-getoptions/option"
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

func TestTrees(t *testing.T) {
	buf := setupLogging()

	t.Run("programTree", func(t *testing.T) {
		root := &programTree{
			Type:          argTypeProgname,
			Name:          os.Args[0],
			ChildCommands: map[string]*programTree{},
			ChildOptions:  map[string]*option.Option{},
		}
		rootopt1Data := ""
		rootopt1 := option.New("rootopt1", option.StringType, &rootopt1Data)
		cmd1 := &programTree{
			Type:          argTypeCommand,
			Name:          "cmd1",
			Parent:        root,
			ChildCommands: map[string]*programTree{},
			ChildOptions:  map[string]*option.Option{},
			Level:         1,
		}
		cmd1opt1Data := ""
		cmd1opt1 := option.New("cmd1opt1", option.StringType, &cmd1opt1Data)
		sub1cmd1 := &programTree{
			Type:          argTypeCommand,
			Name:          "sub1cmd1",
			Parent:        cmd1,
			ChildCommands: map[string]*programTree{},
			ChildOptions:  map[string]*option.Option{},
			Level:         2,
		}
		sub1cmd1opt1Data := ""
		sub1cmd1opt1 := option.New("sub1cmd1opt1", option.StringType, &sub1cmd1opt1Data)
		sub2cmd1 := &programTree{
			Type:          argTypeCommand,
			Name:          "sub2cmd1",
			Parent:        cmd1,
			ChildCommands: map[string]*programTree{},
			ChildOptions:  map[string]*option.Option{},
			Level:         2,
		}
		sub2cmd1opt1Data := ""
		sub2cmd1opt1 := option.New("-", option.StringType, &sub2cmd1opt1Data)
		cmd2 := &programTree{
			Type:          argTypeCommand,
			Name:          "cmd2",
			Parent:        root,
			ChildCommands: map[string]*programTree{},
			ChildOptions:  map[string]*option.Option{},
			Level:         1,
		}
		cmd2opt1Data := ""
		cmd2opt1 := option.New("cmd2opt1", option.StringType, &cmd2opt1Data)

		root.ChildOptions["rootopt1"] = rootopt1
		root.ChildCommands["cmd1"] = cmd1
		root.ChildCommands["cmd2"] = cmd2

		// rootopt1Copycmd1 := rootopt1.Copy().SetParent(cmd1)
		rootopt1Copycmd1 := rootopt1
		cmd1.ChildOptions["rootopt1"] = rootopt1Copycmd1
		cmd1.ChildOptions["cmd1opt1"] = cmd1opt1
		cmd1.ChildCommands["sub1cmd1"] = sub1cmd1
		cmd1.ChildCommands["sub2cmd1"] = sub2cmd1

		// rootopt1Copycmd2 := rootopt1.Copy().SetParent(cmd2)
		rootopt1Copycmd2 := rootopt1
		cmd2.ChildOptions["rootopt1"] = rootopt1Copycmd2
		cmd2.ChildOptions["cmd2opt1"] = cmd2opt1

		// rootopt1Copysub1cmd1 := rootopt1.Copy().SetParent(sub1cmd1)
		rootopt1Copysub1cmd1 := rootopt1
		// cmd1opt1Copysub1cmd1 := cmd1opt1.Copy().SetParent(sub1cmd1)
		cmd1opt1Copysub1cmd1 := cmd1opt1

		sub1cmd1.ChildOptions["rootopt1"] = rootopt1Copysub1cmd1
		sub1cmd1.ChildOptions["cmd1opt1"] = cmd1opt1Copysub1cmd1
		sub1cmd1.ChildOptions["sub1cmd1opt1"] = sub1cmd1opt1

		// rootopt1Copysub2cmd1 := rootopt1.Copy().SetParent(sub2cmd1)
		rootopt1Copysub2cmd1 := rootopt1
		// cmd1opt1Copysub2cmd1 := cmd1opt1.Copy().SetParent(sub2cmd1)
		cmd1opt1Copysub2cmd1 := cmd1opt1
		sub2cmd1.ChildOptions["rootopt1"] = rootopt1Copysub2cmd1
		sub2cmd1.ChildOptions["cmd1opt1"] = cmd1opt1Copysub2cmd1
		sub2cmd1.ChildOptions["-"] = sub2cmd1opt1

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

func TestDefinitionPanics(t *testing.T) {
	recoverFn := func() {
		t.Helper()
		if r := recover(); r == nil {
			t.Errorf("definition did not panic")
		}
	}
	t.Run("Option double defined", func(t *testing.T) {
		defer recoverFn()
		opt := New()
		opt.Bool("flag", false)
		opt.Bool("flag", false)
	})
	t.Run("Option double defined by alias", func(t *testing.T) {
		defer recoverFn()
		opt := New()
		opt.Bool("flag", false)
		opt.Bool("fleg", false, opt.Alias("flag"))
	})
	t.Run("Alias double defined", func(t *testing.T) {
		defer recoverFn()
		opt := New()
		opt.Bool("flag", false, opt.Alias("f"))
		opt.Bool("fleg", false, opt.Alias("f"))
	})
	t.Run("Option double defined across commands", func(t *testing.T) {
		defer recoverFn()
		opt := New()
		opt.Bool("flag", false)
		cmd := opt.NewCommand("cmd", "")
		cmd.Bool("flag", false)
	})
	t.Run("Option double defined across commands by alias", func(t *testing.T) {
		defer recoverFn()
		opt := New()
		opt.Bool("flag", false)
		cmd := opt.NewCommand("cmd", "")
		cmd.Bool("fleg", false, opt.Alias("flag"))
	})
	t.Run("Alias double defined across commands", func(t *testing.T) {
		defer recoverFn()
		opt := New()
		opt.Bool("flag", false, opt.Alias("f"))
		cmd := opt.NewCommand("cmd", "")
		cmd.Bool("fleg", false, opt.Alias("f"))
	})
	t.Run("Command double defined", func(t *testing.T) {
		defer recoverFn()
		opt := New()
		opt.NewCommand("cmd", "")
		opt.NewCommand("cmd", "")
	})
	t.Run("Option name is empty", func(t *testing.T) {
		defer recoverFn()
		New().Bool("", false)
	})
	t.Run("Command name is empty", func(t *testing.T) {
		defer recoverFn()
		opt := New()
		opt.NewCommand("", "")
	})
}

func TestOptionWrongMinMax(t *testing.T) {
	recoverFn := func() {
		t.Helper()
		if r := recover(); r == nil {
			t.Errorf("wrong min/max definition did not panic")
		}
	}

	t.Run("StringSlice min < 1", func(t *testing.T) {
		defer recoverFn()
		New().StringSlice("ss", 0, 1)
	})
	t.Run("IntSlice min < 1", func(t *testing.T) {
		defer recoverFn()
		New().IntSlice("ss", 0, 1)
	})

	t.Run("StringSlice max < 1", func(t *testing.T) {
		defer recoverFn()
		New().StringSlice("ss", 1, 0)
	})
	t.Run("IntSlice max < 1", func(t *testing.T) {
		defer recoverFn()
		New().IntSlice("ss", 1, 0)
	})

	t.Run("StringSlice min > max", func(t *testing.T) {
		defer recoverFn()
		New().StringSlice("ss", 2, 1)
	})
	t.Run("IntSlice min > max", func(t *testing.T) {
		defer recoverFn()
		New().IntSlice("ss", 2, 1)
	})
}
