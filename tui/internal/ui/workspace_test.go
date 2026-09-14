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
