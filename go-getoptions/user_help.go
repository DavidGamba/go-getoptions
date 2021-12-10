package getoptions

import (
	"context"
	"fmt"

	"github.com/DavidGamba/go-getoptions/help"
	"github.com/DavidGamba/go-getoptions/option"
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
		// filter out unknown options
		if option.Unknown {
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
			if node.HelpCommandName != "" && len(node.ChildCommands) > 0 {
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
//     opt.HelpCommand("help", "show this help", opt.Alias("?"))
//
// NOTE: commands must be declared after all options are declared.
func (gopt *GetOpt) HelpCommand(name string, description string, fns ...ModifyFn) *GetOpt {
	// Question: How do I determine the name of the help option so -h or -? work with the command?
	// Maybe I need to add an extra parameter for the help option.
	// Or do we assume they are called the same?

	// TODO: Think about panicking on double call to this method

	gopt.programTree.HelpCommandName = name
	for _, command := range gopt.programTree.ChildCommands {
		command.HelpCommandName = name
	}

	gopt.Bool(name, false, append([]ModifyFn{gopt.Description(description)}, fns...)...)

	cmd := &GetOpt{}
	command := &programTree{
		Type:            argTypeCommand,
		Name:            name,
		Description:     description,
		HelpCommandName: gopt.programTree.HelpCommandName,
		ChildCommands:   map[string]*programTree{},
		ChildOptions:    map[string]*option.Option{},
		Parent:          gopt.programTree,
		Level:           gopt.programTree.Level + 1,
	}
	cmd.programTree = command
	gopt.programTree.AddChildCommand(name, command)
	cmd.SetCommandFn(runHelp)
	cmd.HelpSynopsisArgs("<topic>")
	copyOptionsFromParent(gopt.programTree, false)
	return cmd
}

func runHelp(ctx context.Context, opt *GetOpt, args []string) error {
	if len(args) > 0 {
		for _, command := range opt.programTree.Parent.ChildCommands {
			fmt.Printf("command: %s\n", command.Name)
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
