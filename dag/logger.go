package dag

import (
	"fmt"
	"io"
	"log"
	"os"

	"strings"
)

// var Logger = NewColorLogger()

var Logger = log.New(os.Stderr, "", log.LstdFlags)

type ColorLogger struct {
	logger *log.Logger
	Color  string
}

func NewColorLogger() *ColorLogger {
	return &ColorLogger{
		logger: log.New(os.Stderr, "", log.LstdFlags),
		Color:  "35",
	}
}

func (l *ColorLogger) Printf(format string, v ...interface{}) {
	if l.Color == "" {
		l.logger.Printf(format, v...)
		return
	} else {
		// l.logger.Printf(fmt.Sprintf("\033[%sm%s\033[0m", l.Color, format), v...)
		// l.logger.Printf(fmt.Sprintf("\033[34m%sx", format), v...)
		l.logger.Printf("\033[%sm%s\033[0m", l.Color, fmt.Sprintf(format, v...))
	}
}

func (l *ColorLogger) SetOutput(w io.Writer) {
	l.logger.SetOutput(w)
}

func (g *Graph) colorInfo(format string) string {
	return g.color(g.InfoColor, format)
}

func (g *Graph) colorInfoBold(format string) string {
	return g.color(g.InfoBoldColor, format)
}

func (g *Graph) colorError(format string) string {
	return g.color(g.ErrorColor, format)
}

func (g *Graph) colorErrorBold(format string) string {
	return g.color(g.ErrorBoldColor, format)
}

func (g *Graph) color(color string, format string) string {
	if g.UseColor {
		return format
	}
	if !strings.HasSuffix(format, "\n") {
		return fmt.Sprintf("\033[%sm%s\033[0m", color, format)
	}

	format = format[:len(format)-1]
	return fmt.Sprintf("\033[%sm%s\033[0m\n", color, format)
}
