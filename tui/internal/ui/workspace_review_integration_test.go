package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

type recordingReviewExecutor struct {
	operation string
	itemID    string
}

func (e *recordingReviewExecutor) Execute(operation, itemID string, _ map[string]any) tea.Cmd {
	e.operation, e.itemID = operation, itemID
	return nil
}

func TestWorkspaceReviewRendersCanonicalItemsAndMovesSelection(t *testing.T) {
	m := NewWorkspace()
	m.Width, m.Height = 120, 40
	m.nav.push(ScreenReview)

	next, _ := m.Update(WorkspaceCanonicalMsg{
		IntentID:   "intent-1",
		RevisionID: "revision-1",
		Lifecycle:  "draft",
		IntentPath: "/tmp/intents/intent-1",
		Items: []Item{
			{ID: "item-1", Kind: "goal", Statement: "First statement", Status: "unreviewed"},
			{ID: "item-2", Kind: "constraint", Statement: "Second statement", Status: "unreviewed"},
		},
	})
	m = next.(WorkspaceModel)

	view := m.View().Content
	for _, expected := range []string{"First statement", "Second", "item-1 / goal"} {
		if !strings.Contains(view, expected) {
			t.Fatalf("workspace review missing %q in:\n%s", expected, view)
		}
	}

	next, _ = m.Update(workspaceKey("j"))
	m = next.(WorkspaceModel)
	if m.ReviewFlow.Selected != 1 || m.Review.SelectedID != "item-2" {
		t.Fatalf("selection not synchronized: flow=%d screen=%q", m.ReviewFlow.Selected, m.Review.SelectedID)
	}
	if view = m.View().Content; !strings.Contains(view, "item-2 / constraint") {
		t.Fatalf("selected item detail not rendered:\n%s", view)
	}
}

func TestWorkspaceReviewDelegatesMutationToReviewModel(t *testing.T) {
	m := NewWorkspace()
	m.Width, m.Height = 120, 40
	m.nav.push(ScreenReview)
	next, _ := m.Update(WorkspaceCanonicalMsg{
		IntentID:   "intent-1",
		RevisionID: "revision-1",
		Lifecycle:  "in_review",
		IntentPath: "/tmp/intents/intent-1",
		Items:      []Item{{ID: "item-1", Kind: "goal", Statement: "Review me", Status: "unreviewed"}},
	})
	m = next.(WorkspaceModel)
	executor := &recordingReviewExecutor{}
	m.ReviewFlow.Executor = executor

	next, _ = m.Update(workspaceKey("a"))
	m = next.(WorkspaceModel)
	if m.ReviewFlow.PendingAction != "accept" {
		t.Fatalf("pending action=%q", m.ReviewFlow.PendingAction)
	}
	next, _ = m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	m = next.(WorkspaceModel)
	if executor.operation != "accept_item" || executor.itemID != "item-1" {
		t.Fatalf("mutation=%q item=%q", executor.operation, executor.itemID)
	}
}
