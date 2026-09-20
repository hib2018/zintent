package ui

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// renderPane follows the compact framed-pane layout used by ztasks. Width is
// measured in terminal cells so Japanese text and status icons keep borders
// aligned.
func renderPane(title, content string, width, height int, focused bool) []string {
	width = max(12, width)
	height = max(3, height)
	inner := width - 2
	label := " " + title + " "
	if focused {
		label = "[ " + title + " ]"
	}
	label = fitCells(label, inner)
	lines := []string{"┌" + label + strings.Repeat("─", max(0, inner-ansi.StringWidth(label))) + "┐"}
	contentLines := wrapPaneContent(content, inner)
	for row := 0; row < height-2; row++ {
		value := ""
		if row < len(contentLines) {
			value = fitCells(contentLines[row], inner)
		}
		lines = append(lines, "│"+value+strings.Repeat(" ", max(0, inner-ansi.StringWidth(value)))+"│")
	}
	return append(lines, "└"+strings.Repeat("─", inner)+"┘")
}

func wrapPaneContent(content string, width int) []string {
	if width <= 0 || content == "" {
		return nil
	}
	var result []string
	for _, line := range strings.Split(strings.TrimSuffix(content, "\n"), "\n") {
		if line == "" {
			result = append(result, "")
			continue
		}
		result = append(result, strings.Split(ansi.Wrap(line, width, " "), "\n")...)
	}
	return result
}

func joinPanes(left, right []string) string {
	rows := min(len(left), len(right))
	joined := make([]string, rows)
	for row := 0; row < rows; row++ {
		joined[row] = left[row] + " " + right[row]
	}
	return strings.Join(joined, "\n")
}

func fitCells(value string, width int) string {
	if width <= 0 {
		return ""
	}
	return ansi.Truncate(value, width, "")
}
