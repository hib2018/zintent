package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestApprovalBlocksWhenFindingsExist(t *testing.T) {
	m := NewApproval("i", "r", "rh", "ah", "challenge", []string{"open_comment"})
	if m.Eligible() {
		t.Fatal("expected blocker")
	}
	next, _ := m.Update(tea.KeyPressMsg(tea.Key{Text: "enter", Code: 13}))
	if !strings.Contains(next.(ApprovalModel).Status, "blocked") {
		t.Fatal("approval should remain blocked")
	}
}

func TestApprovalShowsExactRevisionAndCancellation(t *testing.T) {
	m := NewApproval("i", "r-1", "hash-r", "hash-a", "abc123", nil)
	if !strings.Contains(m.View().Content, "hash-r") || !strings.Contains(m.View().Content, "abc123") {
		t.Fatal("missing approval details")
	}
	next, _ := m.Update(tea.KeyPressMsg(tea.Key{Text: "esc", Code: 27}))
	if next.(ApprovalModel).Status != "approval cancelled" {
		t.Fatal("expected cancellation")
	}
}
