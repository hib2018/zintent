package ui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/hib2018/zintent/tui/internal/protocol"
)

type resumeExecutor struct{}

func (resumeExecutor) List() tea.Cmd                   { return nil }
func (resumeExecutor) ListDrafts() tea.Cmd             { return nil }
func (resumeExecutor) InspectDraft(string) tea.Cmd     { return nil }
func (resumeExecutor) ImportDraft(ImportModal) tea.Cmd { return nil }
func (resumeExecutor) OpenIntent(string) tea.Cmd {
	return func() tea.Msg {
		return WorkspaceCanonicalMsg{IntentID: "intent-a", RevisionID: "r1", Lifecycle: "in_review", Items: []Item{{ID: "done", Status: "accepted"}, {ID: "open", Status: "unreviewed"}}}
	}
}

func TestProvenanceTextAcceptsObjectContract(t *testing.T) {
	got := provenanceText(json.RawMessage(`{"content_origin":"human","operation_id":"op-1"}`))
	if got != "origin=human" {
		t.Fatal(got)
	}
	if provenanceText(json.RawMessage(`"legacy"`)) != "legacy" {
		t.Fatal("legacy string provenance broke")
	}
}

func TestWorkspaceStaleRevisionDiscardsPendingStateAndReloadsCanonical(t *testing.T) {
	m := NewWorkspace()
	m.WorkspaceExecutor = resumeExecutor{}
	m.IntentPath = "/tmp/intents/intent-a"
	m.ReviewFlow.Modal, m.ReviewFlow.PendingAction, m.ReviewFlow.Input = "submitting", "edit_item", "stale text"
	m.ReviewFlow.PreviewToken, m.ReviewFlow.ApprovalToken = "preview", "approval"
	m.CommentAction, m.CommentInput = "resolve", "reason"
	next, cmd := m.Update(ActionResultMsg{Response: protocol.Response{Error: &protocol.Error{Code: "stale_revision", Message: "stale"}}})
	m = next.(WorkspaceModel)
	if cmd == nil {
		t.Fatal("stale revision must trigger a canonical reload")
	}
	if m.ReviewFlow.Modal != "" || m.ReviewFlow.Input != "" || m.ReviewFlow.PreviewToken != "" || m.CommentAction != "" {
		t.Fatal("stale pending input and capabilities were retained")
	}
	if _, ok := cmd().(WorkspaceCanonicalMsg); !ok {
		t.Fatal("reload command did not request canonical Intent")
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

func TestDraftPickerListsSiblingDraftJSONAndSelectsWithKeys(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "nested"), 0700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"b.json", "nested/a.json", "ignore.txt"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("{}"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	outside := filepath.Join(t.TempDir(), "outside.json")
	if err := os.WriteFile(outside, []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "linked.json")); err != nil {
		t.Fatal(err)
	}
	msg := (WorkspaceCoreCommands{DraftRoot: root}).ListDrafts()().(DraftListMsg)
	if msg.Err != nil || len(msg.Entries) != 2 {
		t.Fatalf("entries=%v err=%v", msg.Entries, msg.Err)
	}
	if msg.Entries[0].Name != "b.json" || msg.Entries[1].Name != "nested/a.json" {
		t.Fatalf("order=%v", msg.Entries)
	}
	picker := (DraftPicker{Root: root}).Reload(msg.Entries)
	picker.Move(1)
	if picker.SelectedPath != msg.Entries[1].Path || !strings.Contains(picker.View(), "→ nested/a.json") {
		t.Fatalf("picker=%+v\n%s", picker, picker.View())
	}
}

func TestWorkspaceNOpensDraftPickerThenInspectsSelection(t *testing.T) {
	m := NewWorkspace()
	m.WorkspaceExecutor = resumeExecutor{}
	next, cmd := m.Update(workspaceKey("n"))
	m = next.(WorkspaceModel)
	if cmd != nil {
		t.Fatal("fake executor returns nil list command")
	}
	next, _ = m.Update(DraftListMsg{Root: "/project/draft", Entries: []DraftEntry{{Name: "one.json", Path: "/project/draft/one.json"}}})
	m = next.(WorkspaceModel)
	if m.Modal != ModalEditing || m.Drafts.SelectedPath == "" {
		t.Fatalf("picker not opened: %+v", m)
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
