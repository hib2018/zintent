package ui

import (
	tea "charm.land/bubbletea/v2"
	"strings"
	"testing"
	"time"
)

func TestApprovalExactCandidateAndFreshKeyInput(t *testing.T) {
	m := NewApproval("i", "r", "rh", "ch", "ABC", nil)
	m.TokenID = "token"
	m.ExpiresAt = time.Now().Add(time.Minute)
	m.IncludedCount = 2
	m.ExcludedCount = 1
	next, _ := m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	m = next.(ApprovalModel)
	for _, r := range "ABC" {
		next, _ = m.Update(tea.KeyPressMsg(tea.Key{Text: string(r), Code: r}))
		m = next.(ApprovalModel)
	}
	if !m.Ready(time.Now()) {
		t.Fatalf("not ready: %+v", m)
	}
	approvalView := m.View()
	view := approvalView.Content
	if approvalView.Cursor == nil || approvalView.Cursor.X == 0 || approvalView.Cursor.Y == 0 {
		t.Fatalf("challenge cursor is not positioned at the response: %+v", approvalView.Cursor)
	}
	for _, v := range []string{"r", "rh", "ch", "2 / 1", "Response: ABC"} {
		if !strings.Contains(view, v) {
			t.Fatal(view)
		}
	}
}
func TestApprovalExpiryMismatchCancelAndBlockers(t *testing.T) {
	m := NewApproval("i", "r", "h", "c", "ABC", []string{"open comment"})
	m.TokenID = "t"
	m.Response = "ABC"
	if m.Ready(time.Now()) {
		t.Fatal("blocker ignored")
	}
	m.Blockers = nil
	m.ExpiresAt = time.Now().Add(-time.Second)
	if m.Ready(time.Now()) {
		t.Fatal("expiry ignored")
	}
	m.ExpiresAt = time.Now().Add(time.Minute)
	m.Response = "wrong"
	if m.Ready(time.Now()) {
		t.Fatal("mismatch ignored")
	}
	m.Invalidate()
	if m.TokenID != "" || m.Response != "" {
		t.Fatal("token retained")
	}
}
func TestSnapshotNavigationRequiresVerification(t *testing.T) {
	m := NewWorkspace()
	next, _ := m.Update(SnapshotIssuedMsg{Snapshot: SnapshotScreen{SnapshotID: "s", ApprovalID: "a", Verified: true}})
	if next.(WorkspaceModel).Screen() != ScreenSnapshot {
		t.Fatal("snapshot not opened")
	}
}
