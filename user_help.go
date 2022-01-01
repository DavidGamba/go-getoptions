package getoptions

import (
	"context"
	"fmt"

	"github.com/DavidGamba/go-getoptions/internal/help"
	"github.com/DavidGamba/go-getoptions/internal/option"
)

// HelpSection - Indicates what portion of the help to return.
type HelpSection int

// Help Output Types
const (
	helpDefaultName HelpSection = iota
	HelpName
	HelpSynopsis
	HelpCommandList
	HelpOptionList
	HelpCommandInfo
)

// ErrorHelpCalled - Indicates the help has been handled.
var ErrorHelpCalled = fmt.Errorf("help called")

func getCurrentNodeName(n *programTree) string {
	if n.Parent != nil {
		parentName := getCurrentNodeName(n.Parent)
		return fmt.Sprintf("%s %s", parentName, n.Name)
	}
	return n.Name
}

// Help - Default help string that is composed of all available sections.
func (gopt *GetOpt) Help(sections ...HelpSection) string {
	return helpOutput(gopt.programTree, sections...)
}

func helpOutput(node *programTree, sections ...HelpSection) string {
	if len(sections) == 0 {
		// Print all in the following order
		sections = []HelpSection{helpDefaultName, HelpSynopsis, HelpCommandList, HelpOptionList, HelpCommandInfo}
	}
	helpTxt := ""

	scriptName := getCurrentNodeName(node)

	options := []*option.Option{}
	for k, option := range node.ChildOptions {
		// filter out aliases
		if k != option.Name {
			continue
		}
		options = append(options, option)
	}

	for _, section := range sections {
		switch section {
		// Default name only prints name if the name or description is set.
		// The explicit type always prints it.
		case helpDefaultName:
			if node.Parent != nil || node.Description != "" {
				helpTxt += help.Name("", scriptName, node.Description)
				helpTxt += "\n"
			}
		case HelpName:
			helpTxt += help.Name("", scriptName, node.Description)
			helpTxt += "\n"
		case HelpSynopsis:
			commands := []string{}
			for _, command := range node.ChildCommands {
				if command.Name == node.HelpCommandName {
					continue
				}
				commands = append(commands, command.Name)
			}
			helpTxt += help.Synopsis("", scriptName, node.SynopsisArgs, options, commands)
			helpTxt += "\n"
		case HelpCommandList:
			m := make(map[string]string)
			for _, command := range node.ChildCommands {
				if command.Name == node.HelpCommandName {
					continue
				}
				m[command.Name] = command.Description
			}
			commands := help.CommandList(m)
			if commands != "" {
				helpTxt += commands
				helpTxt += "\n"
			}
		case HelpOptionList:
			helpTxt += help.OptionList(options)
		case HelpCommandInfo:
			// Index of 1 because when there is a child command, help is always injected
			if node.HelpCommandName != "" && len(node.ChildCommands) > 1 {
				helpTxt += fmt.Sprintf("Use '%s help <command>' for extra details.\n", scriptName)
			}
		}
	}

	return helpTxt
}

// HelpCommand - Declares a help command and a help option.
// Additionally, it allows to define aliases to the help option.
//
// For example:
//
//     opt.HelpCommand("help", opt.Description("show this help"), opt.Alias("?"))
//
// NOTE: Define after all other commands have been defined.
func (gopt *GetOpt) HelpCommand(name string, fns ...ModifyFn) {
	// TODO: Think about panicking on double call to this method

	// Define help option
	gopt.Bool(name, false, fns...)

	cmdFn := func(parent *programTree) {
		suggestions := []string{}
		for k := range parent.ChildCommands {
			if k != name {
				suggestions = append(suggestions, k)
			}
		}
		cmd := &GetOpt{}
		command := &programTree{
			Name:            name,
			HelpCommandName: name,
			ChildCommands:   map[string]*programTree{},
			ChildOptions:    map[string]*option.Option{},
			Parent:          parent,
			Level:           parent.Level + 1,
			Suggestions:     suggestions,
		}
		cmd.programTree = command
		parent.AddChildCommand(name, command)
		cmd.SetCommandFn(runHelp)
		cmd.HelpSynopsisArgs("<topic>")
	}

	// set HelpCommandName
	runOnParentAndChildrenCommands(gopt.programTree, func(n *programTree) {
		n.HelpCommandName = name
	})

	// Add help command to all commands that have children
	runOnParentAndChildrenCommands(gopt.programTree, func(n *programTree) {
		if len(n.ChildCommands) > 0 && n.Name != name {
			cmdFn(n)
		}
	})

	copyOptionsFromParent(gopt.programTree)
}

func runHelp(ctx context.Context, opt *GetOpt, args []string) error {
	if len(args) > 0 {
		for _, command := range opt.programTree.Parent.ChildCommands {
			if command.Name == args[0] {
				fmt.Fprint(Writer, helpOutput(command))
				return ErrorHelpCalled
			}
		}
		return fmt.Errorf("no help topic for '%s'", args[0])
	}
	fmt.Fprint(Writer, helpOutput(opt.programTree.Parent))
	return ErrorHelpCalled
}

func runOnParentAndChildrenCommands(parent *programTree, fn func(*programTree)) {
	fn(parent)
	for _, command := range parent.ChildCommands {
		runOnParentAndChildrenCommands(command, fn)
	}
}
