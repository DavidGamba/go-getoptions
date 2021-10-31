package log

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"strings"

	"github.com/DavidGamba/go-getoptions"
)

var Logger = log.New(ioutil.Discard, "log ", log.LstdFlags)

// NewCommand - Populate Options definition
func NewCommand(parent *getoptions.GetOpt) *getoptions.GetOpt {
	opt := parent.NewCommand("log", "Show application logs")
	opt.String("level", "INFO", opt.Description("filter debug level"), opt.ValidValues("ERROR", "DEBUG", "INFO"))
	opt.SetCommandFn(Run)
	return opt
}

// Run - Command entry point
func Run(ctx context.Context, opt *getoptions.GetOpt, args []string) error {
	Logger.Printf("log args: %v\n", args)
	filterLevel := opt.Value("level").(string)

	logLines := []string{
		`1900/01/01 01:01:01  INFO beginning of logs`,
		`1900/01/01 01:01:02  DEBUG user 'david.gamba' failed login attempt`,
		`1900/01/01 01:01:03  INFO user 'david.gamba' logged in`,
		`1900/01/01 01:01:04  ERROR request by user 'david.gamba' crashed the system`,
	}

	if !opt.Called("level") {
		fmt.Println(strings.Join(logLines, "\n"))
	} else {
		for _, e := range logLines {
			switch filterLevel {
			case "DEBUG":
				fmt.Println(e)
			case "INFO":
				if strings.Contains(e, "INFO") || strings.Contains(e, "ERROR") {
					fmt.Println(e)
				}
			case "ERROR":
				if strings.Contains(e, "ERROR") {
					fmt.Println(e)
				}
			}
		}
	}

	return nil
}
