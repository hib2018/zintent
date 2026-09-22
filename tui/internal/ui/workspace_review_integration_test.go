package ui

import (
	"encoding/json"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/hib2018/zintent/tui/internal/protocol"
)

type recordingReviewExecutor struct {
	operation string
	itemID    string
	reason    string
	count     int
}

func (e *recordingReviewExecutor) Execute(operation, itemID string, _ map[string]any) tea.Cmd {
	e.operation, e.itemID = operation, itemID
	e.count++
	return nil
}

func (e *recordingReviewExecutor) ExecuteComment(operation, commentID, reason string) tea.Cmd {
	e.operation, e.itemID, e.reason = operation, commentID, reason
	e.count++
	return nil
}

func TestCanonicalReloadKeepsIntentListBlockersInSync(t *testing.T) {
	m := NewWorkspace()
	m.IntentList = m.IntentList.Reload([]IntentEntry{{ID: "intent-1", BlockerCount: 0}})
	next, _ := m.Update(WorkspaceCanonicalMsg{IntentID: "intent-1", RevisionID: "revision-1", Lifecycle: "in_review", IntentPath: "/tmp/intents/intent-1", Items: []Item{{ID: "item-1", Status: "unreviewed"}, {ID: "item-2", Status: "unreviewed"}}, Comments: []CommentRecord{{ID: "comment-1", Status: "open"}}})
	m = next.(WorkspaceModel)
	if got := m.IntentList.Entries[0].BlockerCount; got != 3 {
		t.Fatalf("initial blocker count=%d, want 3", got)
	}
	next, _ = m.Update(WorkspaceCanonicalMsg{IntentID: "intent-1", RevisionID: "revision-2", Lifecycle: "review_complete", IntentPath: "/tmp/intents/intent-1", Items: []Item{{ID: "item-1", Status: "accepted"}, {ID: "item-2", Status: "accepted"}}, Comments: []CommentRecord{{ID: "comment-1", Status: "resolved"}}})
	m = next.(WorkspaceModel)
	if got := m.IntentList.Entries[0].BlockerCount; got != 0 {
		t.Fatalf("reloaded blocker count=%d, want 0", got)
	}
}

func TestWorkspaceDashboardReflectsCanonicalState(t *testing.T) {
	m := NewWorkspace()
	m.Width, m.Height = 120, 40
	m.nav.push(ScreenDashboard)
	next, _ := m.Update(WorkspaceCanonicalMsg{IntentID: "intent-1", RevisionID: "revision-1", Lifecycle: "in_review", IntentPath: "/tmp/intents/intent-1", Items: []Item{{ID: "item-1", Status: "unreviewed"}}, Comments: []CommentRecord{{ID: "comment-1", Status: "open"}}})
	m = next.(WorkspaceModel)
	view := m.View().Content
	for _, expected := range []string{"DASHBOARD", "in_review", "Blockers  : 2", "r Review", "h History", "R Recovery"} {
		if !strings.Contains(view, expected) {
			t.Fatalf("dashboard missing %q in:\n%s", expected, view)
		}
	}
}

func TestWorkspaceCommentsResolveThroughCoreExecutor(t *testing.T) {
	executor := &recordingReviewExecutor{}
	m := NewWorkspace()
	m.IntentPath, m.Revision = "/tmp/intents/intent-1", "revision-1"
	m.ReviewFlow.Executor = executor
	m.Comments = CommentsScreen{Records: []CommentRecord{{ID: "comment-1", TargetItemID: "item-1", Body: "clarify", Status: "open"}}, SelectedID: "comment-1"}
	m.nav.push(ScreenComments)

	next, _ := m.Update(workspaceKey("r"))
	m = next.(WorkspaceModel)
	cursor := m.View().Cursor
	if m.CommentAction != "resolve" || cursor == nil {
		t.Fatal("resolve must open an exclusive closure-reason input")
	}
	if cursor.X == 0 || cursor.Y == 0 {
		t.Fatalf("closure-reason cursor remained at top-left: %+v", cursor.Position)
	}
	next, _ = m.Update(tea.KeyPressMsg(tea.Key{Text: "対応済み", Code: '?'}))
	m = next.(WorkspaceModel)
	next, _ = m.Update(workspaceKey("enter"))
	m = next.(WorkspaceModel)
	if executor.operation != "resolve_comment" || executor.itemID != "comment-1" || executor.reason != "対応済み" || executor.count != 1 {
		t.Fatalf("comment operation was not delegated: %#v", executor)
	}
}

func TestWorkspaceCompletionNavigatesToSelectedBlocker(t *testing.T) {
	m := NewWorkspace()
	m.IntentPath, m.Revision = "/tmp/intents/intent-1", "revision-1"
	m.ReviewFlow.Items = []Item{{ID: "item-1", Status: "unreviewed"}}
	m.ReviewFlow.Comments = []CommentRecord{{ID: "comment-1", Status: "open"}}
	m.Comments = CommentsScreen{Records: append([]CommentRecord(nil), m.ReviewFlow.Comments...)}
	m.updateCompletionState()
	m.nav.push(ScreenCompletion)

	next, _ := m.Update(workspaceKey("enter"))
	m = next.(WorkspaceModel)
	if m.Screen() != ScreenReview || m.Review.SelectedID != "item-1" {
		t.Fatalf("item blocker did not navigate to review: screen=%s item=%s", m.Screen(), m.Review.SelectedID)
	}
	m.nav.back()
	m.Completion.Selected = 1
	next, _ = m.Update(workspaceKey("enter"))
	m = next.(WorkspaceModel)
	if m.Screen() != ScreenComments || m.Comments.SelectedID != "comment-1" {
		t.Fatalf("comment blocker did not navigate to comments: screen=%s comment=%s", m.Screen(), m.Comments.SelectedID)
	}
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
			{ID: "item-1", Kind: "goal", Statement: "First statement", Status: "unreviewed", Provenance: "origin    : human\noperation : accept_item\nactor     : alice"},
			{ID: "item-2", Kind: "constraint", Statement: "Second statement", Status: "unreviewed"},
		},
	})
	m = next.(WorkspaceModel)

	view := m.View().Content
	for _, expected := range []string{"First statement", "Second", "Kind       : goal", "PROVENANCE", "origin    : human", "operation : accept_item", "actor     : alice"} {
		if !strings.Contains(view, expected) {
			t.Fatalf("workspace review missing %q in:\n%s", expected, view)
		}
	}

	next, _ = m.Update(workspaceKey("j"))
	m = next.(WorkspaceModel)
	if m.ReviewFlow.Selected != 1 || m.Review.SelectedID != "item-2" {
		t.Fatalf("selection not synchronized: flow=%d screen=%q", m.ReviewFlow.Selected, m.Review.SelectedID)
	}
	if view = m.View().Content; !strings.Contains(view, "Kind       : constraint") {
		t.Fatalf("selected item detail not rendered:\n%s", view)
	}
}

func TestWorkspaceReviewShowsCommentInputWithoutClipping(t *testing.T) {
	m := NewWorkspace()
	m.Width, m.Height = 120, 40
	m.nav.push(ScreenReview)
	next, _ := m.Update(WorkspaceCanonicalMsg{
		IntentID:   "intent-ja",
		RevisionID: "revision-ja",
		Lifecycle:  "in_review",
		IntentPath: "/tmp/intents/intent-ja",
		Items:      []Item{{ID: "item-ja", Kind: "goal", Statement: "日本語の項目", Status: "unreviewed"}},
	})
	m = next.(WorkspaceModel)

	next, _ = m.Update(workspaceKey("c"))
	m = next.(WorkspaceModel)
	view := m.View()
	for _, expected := range []string{"ACTION", "Type   : comment", "Comment:"} {
		if !strings.Contains(view.Content, expected) {
			t.Fatalf("comment input missing %q in:\n%s", expected, view.Content)
		}
	}
	if view.Cursor == nil {
		t.Fatal("comment input must expose a cursor")
	}

	next, _ = m.Update(tea.KeyPressMsg(tea.Key{Text: "日本語", Code: '?'}))
	m = next.(WorkspaceModel)
	next, _ = m.Update(tea.PasteMsg{Content: "コメント"})
	m = next.(WorkspaceModel)
	if view = m.View(); !strings.Contains(view.Content, "日本語コメント") {
		t.Fatalf("Japanese input was not preserved:\n%s", view.Content)
	}
}

func TestWorkspaceBackPreservesCurrentIntentSelection(t *testing.T) {
	m := NewWorkspace()
	m.Width, m.Height = 120, 40
	m.IntentList = m.IntentList.Reload([]IntentEntry{
		{ID: "intent-en", DisplayName: "English draft", Lifecycle: "draft"},
		{ID: "intent-ja", DisplayName: "日本語ドラフト", Lifecycle: "draft"},
	})
	m.nav.push(ScreenReview)
	next, _ := m.Update(WorkspaceCanonicalMsg{
		IntentID:   "intent-ja",
		RevisionID: "revision-ja",
		Lifecycle:  "draft",
		IntentPath: "/tmp/intents/intent-ja",
		Items:      []Item{{ID: "item-ja", Kind: "goal", Statement: "日本語の項目", Status: "unreviewed"}},
	})
	m = next.(WorkspaceModel)

	next, _ = m.Update(workspaceKey("esc"))
	m = next.(WorkspaceModel)
	next, _ = m.Update(workspaceKey("esc"))
	m = next.(WorkspaceModel)
	if m.Screen() != ScreenIntentList || m.IntentList.SelectedID != "intent-ja" {
		t.Fatalf("back changed current selection: screen=%s selected=%q", m.Screen(), m.IntentList.SelectedID)
	}
	if view := m.View().Content; !strings.Contains(view, "日本語ドラフト") {
		t.Fatalf("current Japanese Intent not shown after back:\n%s", view)
	}
}

func TestWorkspaceApproveFromDashboardOpensChallengeFlow(t *testing.T) {
	m := NewWorkspace()
	m.Width, m.Height = 120, 40
	m.nav.push(ScreenDashboard)
	next, _ := m.Update(WorkspaceCanonicalMsg{
		IntentID:   "intent-1",
		RevisionID: "revision-1",
		Lifecycle:  "review_complete",
		IntentPath: "/tmp/intents/intent-1",
		Items:      []Item{{ID: "item-1", Kind: "goal", Statement: "Approved candidate", Status: "accepted"}},
	})
	m = next.(WorkspaceModel)
	executor := &recordingReviewExecutor{}
	m.ReviewFlow.Executor = executor

	next, _ = m.Update(workspaceKey("p"))
	m = next.(WorkspaceModel)
	if m.Screen() != ScreenReview || executor.operation != "prepare_approval" {
		t.Fatalf("approval did not enter the review challenge flow: screen=%s operation=%q", m.Screen(), executor.operation)
	}

	next, _ = m.Update(ActionResultMsg{
		Operation: "prepare_approval",
		Response:  protocol.Response{OK: true, Result: json.RawMessage(`{"data":{"confirmation":{"token_id":"token-1","challenge":"APPROVE-123456","confirmed_revision_id":"revision-1","confirmed_revision_hash":"revision-hash","approved_content_hash":"content-hash"}}}`)},
	})
	m = next.(WorkspaceModel)
	challengeView := m.View()
	view := challengeView.Content
	if challengeView.Cursor == nil || challengeView.Cursor.X == 0 || challengeView.Cursor.Y == 0 {
		t.Fatalf("workspace challenge cursor remained at top-left: %+v", challengeView.Cursor)
	}
	for _, expected := range []string{"APPROVAL CHALLENGE", "Revision hash", "Approved content hash", "APPROVE-123456", "Response"} {
		if !strings.Contains(view, expected) {
			t.Fatalf("approval input missing %q in:\n%s", expected, view)
		}
	}

	next, _ = m.Update(tea.KeyPressMsg(tea.Key{Text: "APPROVE-123456", Code: '?'}))
	m = next.(WorkspaceModel)
	next, _ = m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	m = next.(WorkspaceModel)
	if executor.operation != "approve_intent" || executor.count != 2 || m.ReviewFlow.Modal != "submitting" {
		t.Fatalf("challenge did not dispatch approval: operation=%q count=%d modal=%q", executor.operation, executor.count, m.ReviewFlow.Modal)
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
