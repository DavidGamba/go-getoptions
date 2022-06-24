package getoptions

import (
	"errors"
	"fmt"
)

// ErrorHelpCalled - Indicates the help has been handled.
var ErrorHelpCalled = fmt.Errorf("help called")

// ErrorParsing - Indicates that there was an error with cli args parsing
var ErrorParsing = errors.New("")
