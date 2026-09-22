package ui

import (
	"strings"
	"testing"
)

func TestSelectionColorAppliesOnlyToTopLine(t *testing.T) {
	got := highlightTopLine("top\nchild\nchild-2")
	lines := strings.Split(got, "\n")
	if !strings.Contains(lines[0], "\x1b[") || strings.Contains(lines[1], "\x1b[") || strings.Contains(lines[2], "\x1b[") {
		t.Fatalf("selection color leaked beyond top line: %q", got)
	}
}

func TestReviewSelectionColorsOnlyStatusAndKind(t *testing.T) {
	got := selectedReviewItemText("→ ", Item{Kind: "goal", Status: "accepted", Statement: "statement remains plain"})
	styleStart := strings.Index(got, "\x1b[")
	styleEnd := strings.Index(got, "\x1b[m")
	if styleStart <= strings.Index(got, "→") || styleEnd < styleStart || strings.Index(got, "statement remains plain") < styleEnd {
		t.Fatalf("selection escaped status/kind label: %q", got)
	}
}

func TestReviewItemTextKeepsStatusWithStatementAndSeparatesItems(t *testing.T) {
	got := reviewItemText("→ ", Item{Kind: "goal", Status: "unreviewed", Statement: "内容を確認する"})
	want := "→ [unreviewed] goal  内容を確認する\n\n"
	if got != want {
		t.Fatalf("item layout=%q want=%q", got, want)
	}
}

func TestViewNarrowAndLongStatementsRemainReadable(t *testing.T) {
	m := New([]Item{{ID: "i-1", Kind: "goal", Statement: "a very long statement that must remain visible", Status: "unreviewed"}})
	m.Width, m.Height = 24, 10
	content := m.View().Content
	if content == "" || len(content) < 24 {
		t.Fatal("narrow view should still render meaningful content")
	}
}

func TestViewEmptyFindingsAndMonochromeText(t *testing.T) {
	m := New(nil)
	m.Findings = nil
	content := m.View().Content
	if content == "" || content != m.View().Content {
		t.Fatal("view must be deterministic without findings or color")
	}
}
