package dag

import (
	"fmt"
	"strings"
)

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
	if !g.UseColor {
		return format
	}
	if !strings.HasSuffix(format, "\n") {
		return fmt.Sprintf("\033[%sm%s\033[0m", color, format)
	}

	format = format[:len(format)-1]
	return fmt.Sprintf("\033[%sm%s\033[0m\n", color, format)
}
