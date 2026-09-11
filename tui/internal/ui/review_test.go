package ui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestNavigationAndStableSelection(t *testing.T) {
	m := New([]Item{{ID: "i1"}, {ID: "i2"}})
	next, _ := m.Update(tea.KeyPressMsg{Key: "j"})
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
