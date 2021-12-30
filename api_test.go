package getoptions

import (
	"reflect"
	"testing"
)

func TestParseCLIArgs(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		mode        Mode
		expected    *programTree
		completions completions
		err         error
	}{

		{"empty", nil, Normal, setupOpt().programTree, []string{}, nil},

		{"empty", []string{}, Normal, setupOpt().programTree, []string{}, nil},

		{"text", []string{"txt", "txt2"}, Normal, func() *programTree {
			n := setupOpt().programTree
			n.ChildText = append(n.ChildText, "txt")
			n.ChildText = append(n.ChildText, "txt2")
			return n
		}(), []string{}, nil},

		{"command", []string{"cmd1"}, Normal, func() *programTree {
			n, err := getNode(setupOpt().programTree, "cmd1")
			if err != nil {
				panic(err)
			}
			return n
		}(), []string{}, nil},

		{"text to command", []string{"cmd1", "txt", "txt2"}, Normal, func() *programTree {
			n, err := getNode(setupOpt().programTree, "cmd1")
			if err != nil {
				panic(err)
			}
			n.ChildText = append(n.ChildText, "txt")
			n.ChildText = append(n.ChildText, "txt2")
			return n
		}(), []string{}, nil},

		{"text to sub command", []string{"cmd1", "sub1cmd1", "txt", "txt2"}, Normal, func() *programTree {
			n, err := getNode(setupOpt().programTree, "cmd1", "sub1cmd1")
			if err != nil {
				panic(err)
			}
			n.ChildText = append(n.ChildText, "txt")
			n.ChildText = append(n.ChildText, "txt2")
			return n
		}(), []string{}, nil},

		{"option with arg", []string{"--rootopt1=hello", "txt", "txt2"}, Normal, func() *programTree {
			n := setupOpt().programTree
			opt, ok := n.ChildOptions["rootopt1"]
			if !ok {
				t.Fatalf("not found")
			}
			opt.Called = true
			opt.UsedAlias = "rootopt1"
			err := opt.Save("hello")
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			n.ChildText = append(n.ChildText, "txt")
			n.ChildText = append(n.ChildText, "txt2")
			return n
		}(), []string{}, nil},

		{"option", []string{"--rootopt1", "hello", "txt", "txt2"}, Normal, func() *programTree {
			n := setupOpt().programTree
			opt, ok := n.ChildOptions["rootopt1"]
			if !ok {
				t.Fatalf("not found")
			}
			opt.Called = true
			opt.UsedAlias = "rootopt1"
			err := opt.Save("hello")
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			n.ChildText = append(n.ChildText, "txt")
			n.ChildText = append(n.ChildText, "txt2")
			return n
		}(), []string{}, nil},

		{"option error missing argument", []string{"--rootopt1"}, Normal, func() *programTree {
			n := setupOpt().programTree
			opt, ok := n.ChildOptions["rootopt1"]
			if !ok {
				t.Fatalf("not found")
			}
			opt.Called = true
			opt.UsedAlias = "rootopt1"
			return n
		}(), []string{}, ErrorMissingArgument},

		{"terminator", []string{"--", "--opt1", "txt", "txt2"}, Normal, func() *programTree {
			n := setupOpt().programTree
			n.ChildText = append(n.ChildText, "--opt1")
			n.ChildText = append(n.ChildText, "txt")
			n.ChildText = append(n.ChildText, "txt2")
			return n
		}(), []string{}, nil},

		{"lonesome dash", []string{"cmd1", "sub2cmd1", "-", "txt", "txt2"}, Normal, func() *programTree {
			tree := setupOpt().programTree
			n, err := getNode(tree, "cmd1", "sub2cmd1")
			if err != nil {
				t.Fatalf("unexpected error: %s, %s", err, stringPT(n))
			}
			opt, ok := n.ChildOptions["-"]
			if !ok {
				t.Fatalf("not found: %s", stringPT(n))
			}
			opt.Called = true
			opt.UsedAlias = "-"
			err = opt.Save("txt")
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			n.ChildText = append(n.ChildText, "txt2")
			return n
		}(), []string{"-", "--cmd1opt1", "--rootopt1"}, nil},

		{"root option to command", []string{"cmd1", "--rootopt1", "hello"}, Normal, func() *programTree {
			tree := setupOpt().programTree
			n, err := getNode(tree, "cmd1")
			if err != nil {
				t.Fatalf("unexpected error: %s, %s", err, stringPT(n))
			}
			opt, ok := n.ChildOptions["rootopt1"]
			if !ok {
				t.Fatalf("not found: %s", stringPT(n))
			}
			opt.Called = true
			opt.UsedAlias = "rootopt1"
			err = opt.Save("hello")
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			return n
		}(), []string{}, nil},

		{"root option to subcommand", []string{"cmd1", "sub2cmd1", "--rootopt1", "hello"}, Normal, func() *programTree {
			tree := setupOpt().programTree
			n, err := getNode(tree, "cmd1", "sub2cmd1")
			if err != nil {
				t.Fatalf("unexpected error: %s, %s", err, stringPT(n))
			}
			opt, ok := n.ChildOptions["rootopt1"]
			if !ok {
				t.Fatalf("not found: %s", stringPT(n))
			}
			opt.Called = true
			opt.UsedAlias = "rootopt1"
			err = opt.Save("hello")
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			return n
		}(), []string{}, nil},

		{"option to subcommand", []string{"cmd1", "sub1cmd1", "--sub1cmd1opt1=hello"}, Normal, func() *programTree {
			tree := setupOpt().programTree
			n, err := getNode(tree, "cmd1", "sub1cmd1")
			if err != nil {
				t.Fatalf("unexpected error: %s, %s", err, stringPT(n))
			}
			opt, ok := n.ChildOptions["sub1cmd1opt1"]
			if !ok {
				t.Fatalf("not found: %s", stringPT(n))
			}
			opt.Called = true
			opt.UsedAlias = "sub1cmd1opt1"
			err = opt.Save("hello")
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			return n
		}(), []string{}, nil},

		{"option to subcommand", []string{"cmd1", "sub1cmd1", "--sub1cmd1opt1", "hello"}, Normal, func() *programTree {
			tree := setupOpt().programTree
			n, err := getNode(tree, "cmd1", "sub1cmd1")
			if err != nil {
				t.Fatalf("unexpected error: %s, %s", err, stringPT(n))
			}
			opt, ok := n.ChildOptions["sub1cmd1opt1"]
			if !ok {
				t.Fatalf("not found: %s", stringPT(n))
			}
			opt.Called = true
			opt.UsedAlias = "sub1cmd1opt1"
			err = opt.Save("hello")
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			return n
		}(), []string{}, nil},

		{"option argument with dash", []string{"cmd1", "sub1cmd1", "--sub1cmd1opt1", "-hello"}, Normal, func() *programTree {
			tree := setupOpt().programTree
			n, err := getNode(tree, "cmd1", "sub1cmd1")
			if err != nil {
				t.Fatalf("unexpected error: %s, %s", err, stringPT(n))
			}
			opt, ok := n.ChildOptions["sub1cmd1opt1"]
			if !ok {
				t.Fatalf("not found: %s", stringPT(n))
			}
			opt.Called = true
			opt.UsedAlias = "sub1cmd1opt1"
			return n
		}(), []string{}, ErrorMissingArgument},

		// {"command", []string{"--opt1", "cmd1", "--cmd1opt1"}, Normal, &programTree{
		// 	Type:   argTypeProgname,
		// 	Name:   os.Args[0],
		// 	option: option{Args: []string{"--opt1", "cmd1", "--cmd1opt1"}},
		// 	Children: []*programTree{
		// 		{
		// 			Type:     argTypeOption,
		// 			Name:     "opt1",
		// 			option:   option{Args: []string{}},
		// 			Children: []*programTree{},
		// 		},
		// 		{
		// 			Type:   argTypeCommand,
		// 			Name:   "cmd1",
		// 			option: option{Args: []string{}},
		// 			Children: []*programTree{
		// 				{
		// 					Type:     argTypeOption,
		// 					Name:     "cmd1opt1",
		// 					option:   option{Args: []string{}},
		// 					Children: []*programTree{},
		// 				},
		// 			},
		// 		},
		// 	},
		// }},
		// {"subcommand", []string{"--opt1", "cmd1", "--cmd1opt1", "sub1cmd1", "--sub1cmd1opt1"}, Normal, &programTree{
		// 	Type:   argTypeProgname,
		// 	Name:   os.Args[0],
		// 	option: option{Args: []string{"--opt1", "cmd1", "--cmd1opt1", "sub1cmd1", "--sub1cmd1opt1"}},
		// 	Children: []*programTree{
		// 		{
		// 			Type:     argTypeOption,
		// 			Name:     "opt1",
		// 			option:   option{Args: []string{}},
		// 			Children: []*programTree{},
		// 		},
		// 		{
		// 			Type:   argTypeCommand,
		// 			Name:   "cmd1",
		// 			option: option{Args: []string{}},
		// 			Children: []*programTree{
		// 				{
		// 					Type:     argTypeOption,
		// 					Name:     "cmd1opt1",
		// 					option:   option{Args: []string{}},
		// 					Children: []*programTree{},
		// 				},
		// 				{
		// 					Type:   argTypeCommand,
		// 					Name:   "sub1cmd1",
		// 					option: option{Args: []string{}},
		// 					Children: []*programTree{
		// 						{
		// 							Type:     argTypeOption,
		// 							Name:     "sub1cmd1opt1",
		// 							option:   option{Args: []string{}},
		// 							Children: []*programTree{},
		// 						},
		// 					},
		// 				},
		// 			},
		// 		},
		// 	},
		// }},
		// {"arg", []string{"hello", "world"}, Normal, &programTree{
		// 	Type:   argTypeProgname,
		// 	Name:   os.Args[0],
		// 	option: option{Args: []string{"hello", "world"}},
		// 	Children: []*programTree{
		// 		{
		// 			Type:     argTypeText,
		// 			Name:     "hello",
		// 			option:   option{Args: []string{}},
		// 			Children: []*programTree{},
		// 		},
		// 		{
		// 			Type:     argTypeText,
		// 			Name:     "world",
		// 			option:   option{Args: []string{}},
		// 			Children: []*programTree{},
		// 		},
		// 	},
		// }},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			logTestOutput := setupTestLogging(t)
			defer logTestOutput()

			tree := setupOpt().programTree
			argTree, _, err := parseCLIArgs(false, tree, test.args, test.mode)
			checkError(t, err, test.err)
			if !reflect.DeepEqual(test.expected, argTree) {
				t.Errorf(spewToFileDiff(t, test.expected, argTree))
				t.Fatalf(programTreeError(test.expected, argTree))
			}
		})

		// This might be too annoying to maintain
		// t.Run("completion "+test.name, func(t *testing.T) {
		// 	logTestOutput := setupTestLogging(t)
		// 	defer logTestOutput()
		//
		// 	tree := setupOpt().programTree
		// 	_, comps, err := parseCLIArgs(true, tree, test.args, test.mode)
		// 	checkError(t, err, test.err)
		// 	if !reflect.DeepEqual(test.completions, comps) {
		// 		t.Fatalf("expected completions: \n%v\n got: \n%v\n", test.completions, comps)
		// 	}
		// })
	}
}

func TestParseCLIArgsCompletions(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		mode        Mode
		completions completions
		err         error
	}{

		{"empty", nil, Normal, []string{"cmd1", "cmd2"}, nil},

		{"empty", []string{}, Normal, []string{"cmd1", "cmd2"}, nil},

		{"text", []string{"txt"}, Normal, []string{}, nil},

		{"command", []string{"cmd"}, Normal, []string{"cmd1", "cmd2"}, nil},

		{"command", []string{"cmd1"}, Normal, []string{"cmd1 "}, nil},

		{"command", []string{"cmd1", ""}, Normal, []string{"sub1cmd1", "sub2cmd1"}, nil},

		{"command", []string{"cmd1", "sub"}, Normal, []string{"sub1cmd1", "sub2cmd1"}, nil},

		{"command", []string{"cmd1", "sub1"}, Normal, []string{"sub1cmd1 "}, nil},

		{"text to command", []string{"cmd1", "txt"}, Normal, []string{}, nil},

		{"text to sub command", []string{"cmd1", "sub1cmd1", "txt"}, Normal, []string{}, nil},

		{"option", []string{"-"}, Normal, []string{"--rootopt1=", "--rootopt1=<string>"}, nil},

		{"option", []string{"--"}, Normal, []string{"--rootopt1=", "--rootopt1=<string>"}, nil},

		{"option", []string{"--r"}, Normal, []string{"--rootopt1=", "--rootopt1=<string>"}, nil},

		{"option", []string{"--rootopt1"}, Normal, []string{"--rootopt1=", "--rootopt1=<string>"}, nil},

		{"option with arg", []string{"--rootopt1=hello"}, Normal, []string{}, nil},

		{"option", []string{"--rootopt1", "hello"}, Normal, []string{}, nil},

		{"terminator", []string{"--", "--opt1"}, Normal, []string{}, nil},

		{"lonesome dash", []string{"cmd1", "sub2cmd1", "-"}, Normal, []string{"-", "--cmd1opt1=", "--rootopt1="}, nil},

		{"root option to command", []string{"cmd1", "--rootopt1", "hello"}, Normal, []string{}, nil},

		{"root option to subcommand", []string{"cmd1", "sub2cmd1", "--rootopt1", "hello"}, Normal, []string{}, nil},

		{"option to subcommand", []string{"cmd1", "sub1cmd1", "--sub1cmd1opt1=hello"}, Normal, []string{}, nil},

		{"option to subcommand", []string{"cmd1", "sub1cmd1", "--sub1cmd1opt1", "hello"}, Normal, []string{}, nil},

		{"option argument with dash", []string{"cmd1", "sub1cmd1", "--sub1cmd1opt1", "-hello"}, Normal, []string{}, ErrorMissingArgument},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			logTestOutput := setupTestLogging(t)
			defer logTestOutput()

			tree := setupOpt().programTree
			_, comps, err := parseCLIArgs(true, tree, test.args, test.mode)
			checkError(t, err, test.err)
			if !reflect.DeepEqual(test.completions, comps) {
				t.Fatalf("expected completions: \n%#v\n got: \n%#v\n", test.completions, comps)
			}
		})
	}
}
