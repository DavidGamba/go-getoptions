package getoptions

import (
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
)

func getHelpName(n *programTree) string {
	if n.Parent != nil {
		parentName := getHelpName(n.Parent)
		return fmt.Sprintf("%s %s", parentName, n.Name)
	}
	return n.Name
}

func (gopt *GetOpt) Help(sections ...HelpSection) string {
	if len(sections) == 0 {
		// Print all in the following order
		sections = []HelpSection{helpDefaultName, HelpSynopsis, HelpCommandList, HelpOptionList}
	}
	helpTxt := ""

	scriptName := getHelpName(gopt.programTree)

	options := []*option.Option{}
	for k, option := range gopt.programTree.ChildOptions {
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
			if gopt.programTree.Parent != nil || gopt.programTree.Description != "" {
				helpTxt += help.Name("", scriptName, gopt.programTree.Description)
				helpTxt += "\n"
			}
		case HelpName:
			helpTxt += help.Name("", scriptName, gopt.programTree.Description)
			helpTxt += "\n"
		case HelpSynopsis:
			commands := []string{}
			for _, command := range gopt.programTree.ChildCommands {
				commands = append(commands, command.Name)
			}
			helpTxt += help.Synopsis("", scriptName, "", options, commands)
			helpTxt += "\n"
		// case HelpCommandList:
		// 	m := make(map[string]string)
		// 	for _, command := range gopt.commands {
		// 		m[command.name] = command.description
		// 	}
		// 	commands := help.CommandList(m)
		// 	if commands != "" {
		// 		helpTxt += commands
		// 		helpTxt += "\n"
		// 	}
		case HelpOptionList:
			helpTxt += help.OptionList(options)
		}
	}

	return helpTxt
}

func (gopt *GetOpt) HelpCommand(name string, description string) *GetOpt {
	cmd := gopt.NewCommand(name, description)
	return cmd
}
