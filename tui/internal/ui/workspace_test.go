package ui

import (
	tea "charm.land/bubbletea/v2"
	"testing"
)

func workspaceKey(value string) tea.KeyPressMsg {
	return tea.KeyPressMsg(tea.Key{Text: value, Code: []rune(value)[0]})
}

func TestWorkspaceNavigationAndModalExclusivity(t *testing.T) {
	m := NewWorkspace()
	next, _ := m.Update(workspaceKey("enter"))
	m = next.(WorkspaceModel)
	if m.Screen() != ScreenDashboard {
		t.Fatalf("screen=%s", m.Screen())
	}
	m.Modal = ModalEditing
	next, _ = m.Update(workspaceKey("r"))
	m = next.(WorkspaceModel)
	if m.Screen() != ScreenDashboard {
		t.Fatal("modal must own keys")
	}
	next, _ = m.Update(workspaceKey("esc"))
	m = next.(WorkspaceModel)
	if m.Modal != ModalClosed {
		t.Fatal("escape must close modal first")
	}
}

func TestWorkspaceRejectsKeyRelease(t *testing.T) {
	m := NewWorkspace()
	next, _ := m.Update(tea.KeyReleaseMsg(tea.Key{Text: "q", Code: 'q'}))
	if next.(WorkspaceModel).Quitting {
		t.Fatal("key release must not trigger action")
	}
}

func TestWorkspaceUsesAlternateScreen(t *testing.T) {
	if !NewWorkspace().View().AltScreen {
		t.Fatal("workspace must own alternate screen")
	}
}

func TestWorkspaceStableIDSelectionAndFallback(t *testing.T) {
	m := NewWorkspace().ReloadRecords([]string{"i-1", "i-2"})
	m.SelectedID = "i-2"
	m = m.ReloadRecords([]string{"i-0", "i-2", "i-3"})
	if m.SelectedID != "i-2" {
		t.Fatalf("selection=%q", m.SelectedID)
	}
	m = m.ReloadRecords([]string{"i-0", "i-3"})
	if m.SelectedID != "i-0" {
		t.Fatalf("fallback=%q", m.SelectedID)
	}
}

func TestWorkspaceSuppressesRepeatedConfirm(t *testing.T) {
	m := NewWorkspace()
	m.Modal = ModalConfirming
	var ok bool
	m, ok = m.BeginConfirm("request-1")
	if !ok || m.Modal != ModalSubmitting {
		t.Fatal("first confirmation must submit")
	}
	_, ok = m.BeginConfirm("request-2")
	if ok {
		t.Fatal("repeated confirmation must be suppressed")
	}
}

func TestWorkspaceCursorOnlyWhileEditing(t *testing.T) {
	m := NewWorkspace()
	if m.View().Cursor != nil {
		t.Fatal("cursor must be hidden outside input")
	}
	m.IntentList.FilterEditing = true
	if m.View().Cursor == nil {
		t.Fatal("text filtering must show cursor")
	}
	m.IntentList.FilterEditing = false
	m.Modal = ModalEditing
	if m.View().Cursor != nil {
		t.Fatal("Draft selection is keyboard navigation, not text input")
	}
}
