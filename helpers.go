package getoptions

import (
	"fmt"

	"github.com/DavidGamba/go-getoptions/text"
)

// GetRequiredArg - Get the next argument from the args list and error if it doesn't exist.
// By default the error will include the HelpSynopsis section but it can be overriden with the list of sections or getoptions.HelpNone.
//
// If the arguments have been named with `opt.HelpSynopsisArg` then the error will include the argument name.
func (gopt *GetOpt) GetRequiredArg(args []string, sections ...HelpSection) (string, []string, error) {
	if len(args) < 1 {
		if len(gopt.programTree.SynopsisArgs) > gopt.programTree.SynopsisArgsIdx {
			argName := gopt.programTree.SynopsisArgs[gopt.programTree.SynopsisArgsIdx].Arg
			fmt.Fprintf(Writer, text.ErrorMissingRequiredNamedArgument+"\n", argName)
		} else {
			fmt.Fprintf(Writer, "%s\n", text.ErrorMissingRequiredArgument)
		}
		if sections != nil {
			fmt.Fprintf(Writer, "%s", gopt.Help(sections...))
		} else {
			fmt.Fprintf(Writer, "%s", gopt.Help(HelpSynopsis))
		}
		gopt.programTree.SynopsisArgsIdx++
		return "", args, ErrorHelpCalled
	}
	gopt.programTree.SynopsisArgsIdx++
	return args[0], args[1:], nil
}
