package getoptions

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

func (gopt *GetOpt) Help(sections ...HelpSection) string {
	return ""
}
