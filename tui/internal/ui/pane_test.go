package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestRenderPaneKeepsUnicodeBordersAligned(t *testing.T) {
	for number, line := range renderPane("詳細", "日本語の長い内容🚀\nsecond", 32, 6, true) {
		if got := ansi.StringWidth(line); got != 32 {
			t.Fatalf("line %d width=%d: %q", number, got, line)
		}
	}
}

func TestWorkspaceUsesFramedNavigationMainAndStatusPanes(t *testing.T) {
	m := NewWorkspace()
	m.Width, m.Height = 100, 28
	output := m.View().Content
	for _, title := range []string{"Workspace", "Navigation", "[ intents ]", "Status"} {
		if !strings.Contains(output, title) {
			t.Fatalf("missing pane %q:\n%s", title, output)
		}
	}
	if strings.Count(output, "┌") < 4 || strings.Count(output, "┘") < 4 {
		t.Fatalf("panes are not fully framed:\n%s", output)
	}
}

func TestReviewUsesListAndDetailPanes(t *testing.T) {
	m := New([]Item{{ID: "I-001", Kind: "goal", Status: "unreviewed", Statement: "確認する"}})
	m.Width, m.Height = 100, 28
	output := m.View().Content
	for _, title := range []string{"Review", "[ Items ]", "Item Detail"} {
		if !strings.Contains(output, title) {
			t.Fatalf("missing pane %q:\n%s", title, output)
		}
	}
}
