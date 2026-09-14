package ui

import (
	tea "charm.land/bubbletea/v2"
	"testing"
)

func key(text string) tea.KeyPressMsg {
	return tea.KeyPressMsg(tea.Key{Text: text, Code: rune(text[0])})
}

func TestResumeRestoresStableItemAndBlockers(t *testing.T) {
	m := New([]Item{{ID: "i-1", Status: "accepted"}, {ID: "i-2", Status: "unreviewed"}})
	m = m.Restore("r-2", "in_review", "i-2", m.Items)
	if m.Selected != 1 || m.Revision != "r-2" {
		t.Fatalf("resume did not restore stable selection: %#v", m)
	}
	if len(m.ResumeBlockers()) != 1 {
		t.Fatalf("expected one blocker: %#v", m.ResumeBlockers())
	}
}

func TestApprovedMutationIsGovernedByCoreExecutor(t *testing.T) {
	m := New([]Item{{ID: "i-1", Status: "accepted"}})
	m.Lifecycle = "approved"
	m.Executor = fakeExecutor{}
	next, cmd := m.Update(key("a"))
	_, cmd = next.(Model).Update(key("enter"))
	if cmd == nil {
		t.Fatal("approved mutation must still dispatch through the core executor")
	}
}
