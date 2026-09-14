package ui

import (
	tea "charm.land/bubbletea/v2"
	"testing"
)

func TestQuitRestoresModelWithoutMutation(t *testing.T) {
	m := New([]Item{{ID: "i-1", Status: "accepted"}})
	next, cmd := m.Update(tea.KeyPressMsg(tea.Key{Text: "q", Code: 'q'}))
	if !next.(Model).Quitting || cmd == nil {
		t.Fatal("quit must request terminal shutdown")
	}
	if next.(Model).Items[0].Status != "accepted" {
		t.Fatal("quit must not mutate items")
	}
}
