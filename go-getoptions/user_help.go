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

func getHelpName(n *programTree) string {
	if n.Parent != nil {
		parentName := getHelpName(n.Parent)
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

	scriptName := getHelpName(node)

	options := []*option.Option{}
	for k, option := range node.ChildOptions {
		// filter out aliases
		if k == option.Name {
			options = append(options, option)
		}
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
			helpTxt += help.Synopsis("", scriptName, "", options, commands)
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
			if node.HelpCommandName != "" {
				helpTxt += fmt.Sprintf("Use '%s help <command>' for extra details.\n", scriptName)
			}
		}
	}

	return helpTxt
}

func (gopt *GetOpt) HelpCommand(name string, description string) *GetOpt {
	// TODO: Think about panicking on double call to this method

	gopt.programTree.HelpCommandName = name
	for _, command := range gopt.programTree.ChildCommands {
		command.HelpCommandName = name
	}

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
	return cmd
}

func runHelp(ctx context.Context, opt *GetOpt, args []string) error {
	fmt.Fprint(Writer, helpOutput(opt.programTree.Parent))
	return nil
}
