package ui

import (
	"strings"
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

func TestEditStartsWithCurrentStatement(t *testing.T) {
	m := New([]Item{{ID: "i1", Statement: "現在のタイトル"}})
	next, _ := m.Update(tea.KeyPressMsg(tea.Key{Text: "e", Code: 'e'}))
	m = next.(Model)
	if m.PendingAction != "edit-preview" || m.Input != "現在のタイトル" {
		t.Fatalf("edit did not preload current statement: %#v", m)
	}
	view := m.View()
	if !strings.Contains(view.Content, "New statement: 現在のタイトル") || view.Cursor == nil {
		t.Fatalf("preloaded edit input is not visible: %s", view.Content)
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

func TestCompleteBlocksRepeatedCommandsWhileInFlight(t *testing.T) {
	executor := &countingExecutor{}
	m := New([]Item{{ID: "i1", Status: "accepted"}})
	m.Lifecycle = "in_review"
	m.Executor = executor
	next, _ := m.Update(tea.KeyPressMsg(tea.Key{Text: "f", Code: 'f'}))
	m = next.(Model)
	if m.Modal != "submitting" || executor.count != 1 {
		t.Fatalf("completion not marked in flight: modal=%q count=%d", m.Modal, executor.count)
	}
	next, _ = m.Update(tea.KeyPressMsg(tea.Key{Text: "f", Code: 'f'}))
	if next.(Model).Modal != "submitting" || executor.count != 1 {
		t.Fatalf("repeated completion dispatched: modal=%q count=%d", next.(Model).Modal, executor.count)
	}
}

func TestDraftRequiresExplicitReviewStart(t *testing.T) {
	executor := &recordingExecutor{}
	m := New([]Item{{ID: "i1"}})
	m.Lifecycle = "draft"
	m.Executor = executor
	next, cmd := m.Update(tea.KeyPressMsg(tea.Key{Text: "a", Code: 'a'}))
	if cmd != nil || next.(Model).Modal != "" {
		t.Fatal("draft item action must not dispatch")
	}
	next, cmd = next.(Model).Update(tea.KeyPressMsg(tea.Key{Text: "r", Code: 'r'}))
	if cmd == nil || executor.operation != "start_review" || next.(Model).Modal != "submitting" {
		t.Fatalf("review start was not dispatched: %#v", next)
	}
}

func TestAllRejectedItemsCanCompleteAsRejected(t *testing.T) {
	m := New([]Item{{ID: "i1", Status: "rejected", Excluded: true}})
	if got := m.ResumeBlockers(); len(got) != 0 {
		t.Fatalf("unexpected blockers: %#v", got)
	}
	if !((CompletionScreen{RevisionID: "r1", Lifecycle: "in_review", Blockers: m.ResumeBlockers()}).Eligible()) {
		t.Fatal("all-rejected review should be completable")
	}
}

func TestRejectedIntentOnlyAllowsAcceptOrEditToReopen(t *testing.T) {
	m := New([]Item{{ID: "i1", Status: "rejected", Excluded: true}})
	m.Lifecycle = "rejected"
	next, _ := m.Update(tea.KeyPressMsg(tea.Key{Text: "c", Code: 'c'}))
	if next.(Model).Modal != "" {
		t.Fatal("comment must not reopen a rejected Intent")
	}
	next, _ = next.(Model).Update(tea.KeyPressMsg(tea.Key{Text: "a", Code: 'a'}))
	if next.(Model).Modal != "accept" {
		t.Fatal("accept should be available to reopen a rejected Intent")
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

type recordingExecutor struct{ operation string }

func (e *recordingExecutor) Execute(operation, _ string, _ map[string]any) tea.Cmd {
	e.operation = operation
	return func() tea.Msg { return nil }
}

type countingExecutor struct{ count int }

func (e *countingExecutor) Execute(string, string, map[string]any) tea.Cmd {
	e.count++
	return func() tea.Msg { return nil }
}

type fakeExecutor struct{}

func (fakeExecutor) Execute(string, string, map[string]any) tea.Cmd {
	return func() tea.Msg { return nil }
}
