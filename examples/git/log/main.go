package log

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"

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
	opt.SetUnknownMode("pass")
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
	cmd := exec.Command("git", append([]string{"log"}, remaining...)...)
	stdoutStderr, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", stdoutStderr)
		os.Exit(1)
	}
	fmt.Printf("%s\n", stdoutStderr)
}
