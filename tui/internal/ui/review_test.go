package ui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestNavigationAndStableSelection(t *testing.T) {
	m := New([]Item{{ID: "i1"}, {ID: "i2"}})
	next, _ := m.Update(tea.KeyPressMsg(tea.Key{Text: "j", Code: 'j'}))
	if next.(Model).Selected != 1 {
		t.Fatal("expected second item")
	}
	next, _ = next.(Model).Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	got := next.(Model)
	if got.Selected != 1 || got.Width != 80 {
		t.Fatal("resize lost state")
	}
}

func TestViewContainsReviewContext(t *testing.T) {
	m := New([]Item{{ID: "i1", Statement: "Goal", Status: "unreviewed"}})
	m.Revision, m.Actor = "r1", "alice"
	if got := m.View().Content; got == "" {
		t.Fatal("empty view")
	}
}

func TestReviewModalCanCancelAndConfirm(t *testing.T) {
	m := New([]Item{{ID: "i1"}})
	next, _ := m.Update(tea.KeyPressMsg(tea.Key{Text: "a", Code: 'a'}))
	got := next.(Model)
	if got.Modal != "accept" || got.PendingAction != "accept" {
		t.Fatalf("expected accept modal: %#v", got)
	}
	next, _ = got.Update(tea.KeyPressMsg(tea.Key{Text: "esc", Code: 27}))
	got = next.(Model)
	if got.Modal != "" || got.Status != "" {
		t.Fatalf("escape should cancel modal: %#v", got)
	}
	next, _ = got.Update(tea.KeyPressMsg(tea.Key{Text: "x", Code: 'x'}))
	next, _ = next.(Model).Update(tea.KeyPressMsg(tea.Key{Text: "enter", Code: 13}))
	if next.(Model).Status != "reject confirmed for i1" {
		t.Fatalf("expected confirmation status: %#v", next.(Model))
	}
}
