package log

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/DavidGamba/go-getoptions"
)

var logger = log.New(ioutil.Discard, "", log.LstdFlags)

func synopsis() {
	synopsis := `NAME
	git-log - Show commit logs
USAGE
	log [--help]
`
	fmt.Fprintln(os.Stderr, synopsis)
}

func Log(args []string) {
	opt := getoptions.New()
	opt.Bool("help", false, opt.Alias("?"))
	opt.Bool("debug", false)
	remaining, err := opt.Parse(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
		os.Exit(1)
	}
	if opt.Called("help") {
		synopsis()
		os.Exit(1)
	}
	if opt.Called("debug") {
		logger.SetOutput(os.Stderr)
	}
	log.Println(remaining)
	fmt.Printf("log output...\n")
}
