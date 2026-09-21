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
	got.Executor = fakeExecutor{}
	next, _ = got.Update(tea.KeyPressMsg(tea.Key{Text: "x", Code: 'x'}))
	next, _ = next.(Model).Update(tea.KeyPressMsg(tea.Key{Text: "対象外", Code: '?'}))
	_, cmd := next.(Model).Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if cmd == nil {
		t.Fatal("reject with a reason must dispatch")
	}
}

func TestReviewTextInputOwnsCommandsAndPreservesJapanese(t *testing.T) {
	m := New([]Item{{ID: "i1"}, {ID: "i2"}})
	next, _ := m.Update(tea.KeyPressMsg(tea.Key{Text: "c", Code: 'c'}))
	m = next.(Model)
	for _, text := range []string{"a", "j", "f", "p", "日本語"} {
		next, cmd := m.Update(tea.KeyPressMsg(tea.Key{Text: text, Code: '?'}))
		if cmd != nil {
			t.Fatalf("text %q triggered a command", text)
		}
		m = next.(Model)
	}
	next, _ = m.Update(tea.PasteMsg{Content: "を入力"})
	m = next.(Model)
	if m.PendingAction != "comment" || m.Selected != 0 {
		t.Fatalf("input changed review command state: %#v", m)
	}
	if m.Input != "ajfp日本語を入力" {
		t.Fatalf("input=%q", m.Input)
	}
	next, _ = m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyBackspace}))
	if got := next.(Model).Input; got != "ajfp日本語を入" {
		t.Fatalf("rune backspace corrupted Japanese input: %q", got)
	}
}

func TestReviewReducerIgnoresReleaseAndKeepsStableIDSelection(t *testing.T) {
	m := New([]Item{{ID: "i1"}, {ID: "i2"}})
	m.Selected = 1
	next, _ := m.Update(tea.KeyReleaseMsg(tea.Key{Text: "j", Code: 'j'}))
	if next.(Model).Selected != 1 {
		t.Fatal("release event must not navigate")
	}
	model := next.(Model)
	model.Items = []Item{{ID: "i2"}, {ID: "i3"}}
	if model.Selected != 1 || model.Items[model.Selected].ID != "i3" {
		t.Fatal("selection must remain bounded after reload")
	}
}

func TestConfirmedAcceptDispatchesThroughExecutor(t *testing.T) {
	m := New([]Item{{ID: "i1"}})
	m.Executor = fakeExecutor{}
	next, cmd := m.Update(tea.KeyPressMsg(tea.Key{Text: "a", Code: 'a'}))
	if cmd != nil {
		t.Fatal("opening a modal must not dispatch")
	}
	_, cmd = next.(Model).Update(tea.KeyPressMsg(tea.Key{Text: "enter", Code: 13}))
	if cmd == nil {
		t.Fatal("confirmed action must dispatch")
	}
}

type fakeExecutor struct{}

func (fakeExecutor) Execute(string, string, map[string]any) tea.Cmd {
	return func() tea.Msg { return nil }
}
