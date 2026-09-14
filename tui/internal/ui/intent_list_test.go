package ui

import (
	"os"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
)

type resumeExecutor struct{}

func (resumeExecutor) List() tea.Cmd                   { return nil }
func (resumeExecutor) InspectDraft(string) tea.Cmd     { return nil }
func (resumeExecutor) ImportDraft(ImportModal) tea.Cmd { return nil }
func (resumeExecutor) OpenIntent(string) tea.Cmd {
	return func() tea.Msg {
		return WorkspaceCanonicalMsg{IntentID: "intent-a", RevisionID: "r1", Lifecycle: "in_review", Items: []Item{{ID: "done", Status: "accepted"}, {ID: "open", Status: "unreviewed"}}}
	}
}

func TestWorkspaceOpenResumesAtFirstUnresolvedItem(t *testing.T) {
	m := NewWorkspace()
	m.WorkspaceExecutor = resumeExecutor{}
	m.IntentList = m.IntentList.Reload([]IntentEntry{{ID: "intent-a", Path: "intent-a"}})
	next, cmd := m.Update(workspaceKey("enter"))
	m = next.(WorkspaceModel)
	if cmd == nil {
		t.Fatal("open must load canonical state")
	}
	next, _ = m.Update(cmd())
	m = next.(WorkspaceModel)
	if m.Screen() != ScreenDashboard || m.Review.SelectedID != "open" {
		t.Fatalf("screen=%s selected=%s", m.Screen(), m.Review.SelectedID)
	}
}

func TestIntentListSearchStableSelectionAndCorruption(t *testing.T) {
	s := IntentListScreen{SelectedID: "b", Filter: "review"}.Reload([]IntentEntry{{ID: "b", DisplayName: "review beta", Lifecycle: "in_review"}, {ID: "a", DisplayName: "draft", Lifecycle: "draft"}, {ID: "corrupt", DisplayName: "review bad", Corrupt: true, Finding: "bad HEAD"}})
	if s.SelectedID != "b" || len(s.Visible()) != 2 {
		t.Fatalf("%+v", s)
	}
	if !strings.Contains(s.View(), "CORRUPT") {
		t.Fatal(s.View())
	}
	s = s.Reload([]IntentEntry{{ID: "a", DisplayName: "review alpha"}})
	if s.SelectedID != "a" {
		t.Fatal("fallback failed")
	}
}
func TestImportModalConfirmationExpiryAndFailureReset(t *testing.T) {
	m := ImportModal{Phase: ModalConfirming, Token: "token", Destination: "intent-a", ExpiresAt: time.Now().Add(time.Minute)}
	if !m.CanSubmit(time.Now()) {
		t.Fatal()
	}
	m.Findings = []string{"collision"}
	if m.CanSubmit(time.Now()) {
		t.Fatal("blocker ignored")
	}
	m.ResetFailure("failed")
	if m.Token != "" || m.Phase != ModalError {
		t.Fatal()
	}
}
func TestIntentListGolden(t *testing.T) {
	golden, err := os.ReadFile("testdata/intent-list.golden")
	if err != nil {
		t.Fatal(err)
	}
	view := (IntentListScreen{Entries: []IntentEntry{{ID: "intent-a", Lifecycle: "draft", BlockerCount: 2}}, SelectedID: "intent-a"}).View()
	for _, line := range strings.Split(strings.TrimSpace(string(golden)), "\n") {
		if !strings.Contains(view, line) {
			t.Fatalf("missing %q: %s", line, view)
		}
	}
}
