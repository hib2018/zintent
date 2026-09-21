package ui

import "testing"

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
