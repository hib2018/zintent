package ui

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/hib2018/zintent/tui/internal/protocol"
	"github.com/hib2018/zintent/tui/internal/runner"
)

// CoreCommands is the only mutation gateway used by the review model. It
// sends one bounded request to the Zig core and turns the response into a
// Bubble Tea message; the UI never edits an artifact directly.
type CoreCommands struct {
	Core       runner.Core
	IntentPath string
	Expected   string
	Actor      map[string]any
}

func (c *CoreCommands) SetExpected(revision string) { c.Expected = revision }

type ActionResultMsg struct {
	Operation string
	ItemID    string
	Response  protocol.Response
	Err       error
}

type ReloadResultMsg struct {
	Response protocol.Response
	Err      error
}

type WorkspaceListMsg struct {
	Entries []IntentEntry
	Err     error
}
type DraftListMsg struct {
	Root    string
	Entries []DraftEntry
	Err     error
}
type DraftPreviewMsg struct {
	SourceHash, IntentID, Destination, Token string
	Findings                                 []string
	Err                                      error
}
type DraftImportedMsg struct {
	IntentID, Destination string
	Err                   error
}
type HistoryLoadedMsg struct {
	History HistoryScreen
	Err     error
}
type RevisionLoadedMsg struct {
	Revision RevisionRecord
	Err      error
}
type DiffLoadedMsg struct {
	BaseID, TargetID string
	Changes          []ItemChange
	Err              error
}
type ValidationLoadedMsg struct {
	Findings []ValidationFinding
	Err      error
}
type SnapshotLoadedMsg struct {
	Snapshot SnapshotScreen
	Err      error
}
type RecoveryLoadedMsg struct {
	Recovery RecoveryScreen
	Err      error
}
type RecoveryCleanupMsg struct {
	Removed int
	Code    string
	Err     error
}

// OpenIntent resolves only a direct-child path supplied by the verified
// workspace listing, then asks the core for canonical state.
func (c WorkspaceCoreCommands) OpenIntent(intentPath string) tea.Cmd {
	return func() tea.Msg {
		if filepath.IsAbs(intentPath) || filepath.Base(intentPath) != intentPath || intentPath == "." {
			return WorkspaceCanonicalMsg{Err: errors.New("invalid workspace Intent path")}
		}
		response, err := c.run("show_intent", map[string]any{"intent_path": filepath.Join(c.WorkspacePath, intentPath)})
		if err != nil {
			return WorkspaceCanonicalMsg{Err: err}
		}
		if !response.OK {
			return WorkspaceCanonicalMsg{Err: responseError(response)}
		}
		return decodeWorkspaceIntent(response.Result, filepath.Join(c.WorkspacePath, intentPath))
	}
}

func decodeWorkspaceIntent(raw json.RawMessage, intentPath string) WorkspaceCanonicalMsg {
	var result struct {
		Data struct {
			Intent struct {
				RevisionID      string `json:"revision_id"`
				RevisionPayload struct {
					IntentID  string `json:"intent_id"`
					Lifecycle string `json:"lifecycle_state"`
					Items     []struct {
						ID         string          `json:"item_id"`
						Kind       string          `json:"kind"`
						Statement  string          `json:"statement"`
						Status     string          `json:"review_status"`
						Included   bool            `json:"included_in_approval"`
						Provenance json.RawMessage `json:"provenance"`
						Rationale  string          `json:"rationale"`
					} `json:"items"`
					Comments []struct {
						ID           string `json:"comment_id"`
						TargetItemID string `json:"target_item_id"`
						Body         string `json:"body"`
						Status       string `json:"status"`
					} `json:"comments"`
				} `json:"revision_payload"`
			} `json:"intent"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return WorkspaceCanonicalMsg{Err: err}
	}
	payload := result.Data.Intent.RevisionPayload
	items := make([]Item, 0, len(payload.Items))
	for _, item := range payload.Items {
		items = append(items, Item{ID: item.ID, Kind: item.Kind, Statement: item.Statement, Status: item.Status, Provenance: provenanceText(item.Provenance), Rationale: item.Rationale, Excluded: !item.Included})
	}
	comments := make([]CommentRecord, 0, len(payload.Comments))
	for _, comment := range payload.Comments {
		comments = append(comments, CommentRecord{ID: comment.ID, TargetItemID: comment.TargetItemID, Body: comment.Body, Status: comment.Status})
	}
	return WorkspaceCanonicalMsg{IntentID: payload.IntentID, RevisionID: result.Data.Intent.RevisionID, Lifecycle: payload.Lifecycle, IntentPath: intentPath, Items: items, Comments: comments}
}

type WorkspaceCoreCommands struct {
	Core          runner.Core
	WorkspacePath string
	DraftRoot     string
	Actor         map[string]any
}

// ReviewCommands returns the same mutation gateway used by the standalone
// review TUI. Workspace review must not duplicate domain transitions.
func (c WorkspaceCoreCommands) ReviewCommands(intentPath, expectedRevision string) *CoreCommands {
	return &CoreCommands{Core: c.Core, IntentPath: intentPath, Expected: expectedRevision, Actor: c.Actor}
}

func (c WorkspaceCoreCommands) LoadHistory(intentPath string) tea.Cmd {
	return func() tea.Msg {
		response, err := c.run("list_revisions", map[string]any{"intent_path": intentPath})
		if err != nil {
			return HistoryLoadedMsg{Err: err}
		}
		if !response.OK {
			return HistoryLoadedMsg{Err: responseError(response)}
		}
		var result struct {
			Data struct {
				Reachable []json.RawMessage `json:"reachable"`
				Orphans   []json.RawMessage `json:"orphans"`
			} `json:"data"`
		}
		if err := json.Unmarshal(response.Result, &result); err != nil {
			return HistoryLoadedMsg{Err: err}
		}
		history := HistoryScreen{}
		for _, raw := range result.Data.Reachable {
			history.Revisions = append(history.Revisions, decodeRevisionRecord(raw, true))
		}
		for _, raw := range result.Data.Orphans {
			history.Orphans = append(history.Orphans, decodeRevisionRecord(raw, false))
		}
		return HistoryLoadedMsg{History: history}
	}
}

func (c WorkspaceCoreCommands) LoadRevision(intentPath, revisionID string) tea.Cmd {
	return func() tea.Msg {
		response, err := c.run("inspect_revision", map[string]any{"intent_path": intentPath, "revision_id": revisionID})
		if err != nil {
			return RevisionLoadedMsg{Err: err}
		}
		if !response.OK {
			return RevisionLoadedMsg{Err: responseError(response)}
		}
		var result struct {
			Data struct {
				Revision json.RawMessage `json:"revision"`
			} `json:"data"`
		}
		if err := json.Unmarshal(response.Result, &result); err != nil {
			return RevisionLoadedMsg{Err: err}
		}
		return RevisionLoadedMsg{Revision: decodeRevisionRecord(result.Data.Revision, true)}
	}
}

func (c WorkspaceCoreCommands) LoadDiff(intentPath, baseID, targetID string) tea.Cmd {
	return func() tea.Msg {
		response, err := c.run("diff_revisions", map[string]any{"intent_path": intentPath, "from_revision_id": baseID, "to_revision_id": targetID})
		if err != nil {
			return DiffLoadedMsg{Err: err}
		}
		if !response.OK {
			return DiffLoadedMsg{Err: responseError(response)}
		}
		var result struct {
			Data struct {
				From    string `json:"from_revision_id"`
				To      string `json:"to_revision_id"`
				Changes []struct {
					RecordID string `json:"record_id"`
					Before   string `json:"before"`
					After    string `json:"after"`
				} `json:"changes"`
			} `json:"data"`
		}
		if err := json.Unmarshal(response.Result, &result); err != nil {
			return DiffLoadedMsg{Err: err}
		}
		changes := make([]ItemChange, 0, len(result.Data.Changes))
		for _, change := range result.Data.Changes {
			changes = append(changes, ItemChange{ItemID: change.RecordID, Before: change.Before, After: change.After})
		}
		return DiffLoadedMsg{BaseID: result.Data.From, TargetID: result.Data.To, Changes: changes}
	}
}

func (c WorkspaceCoreCommands) LoadValidation(intentPath string) tea.Cmd {
	return func() tea.Msg {
		response, err := c.run("validate_intent", map[string]any{"intent_path": intentPath})
		if err != nil {
			return ValidationLoadedMsg{Err: err}
		}
		if !response.OK {
			err := responseError(response)
			code := "validation_failed"
			message := err.Error()
			if response.Error != nil {
				code, message = response.Error.Code, response.Error.Message
			}
			return ValidationLoadedMsg{Findings: []ValidationFinding{{Code: code, Severity: "blocking", Message: message}}}
		}
		return ValidationLoadedMsg{Findings: []ValidationFinding{}}
	}
}

func (c WorkspaceCoreCommands) LoadSnapshot(intentPath, snapshotID string) tea.Cmd {
	return func() tea.Msg {
		snapshotID = strings.TrimSuffix(filepath.Base(snapshotID), ".json")
		if snapshotID == "" || snapshotID == "." {
			return SnapshotLoadedMsg{Err: errors.New("no approved snapshot is linked to this Intent")}
		}
		response, err := c.run("inspect_snapshot", map[string]any{"intent_path": intentPath, "snapshot_id": snapshotID})
		if err != nil {
			return SnapshotLoadedMsg{Err: err}
		}
		if !response.OK {
			return SnapshotLoadedMsg{Err: responseError(response)}
		}
		var result struct {
			Data struct {
				Verified bool `json:"verified"`
				Snapshot struct {
					SnapshotID string `json:"snapshot_id"`
					Approval   struct {
						ApprovalID          string `json:"approval_id"`
						ApprovedRevisionID  string `json:"approved_revision_id"`
						ApprovedContentHash string `json:"approved_content_hash"`
						ApprovingActor      struct {
							ActorID string `json:"actor_id"`
						} `json:"approving_actor"`
					} `json:"approval"`
				} `json:"snapshot"`
			} `json:"data"`
		}
		if err := json.Unmarshal(response.Result, &result); err != nil {
			return SnapshotLoadedMsg{Err: err}
		}
		snapshot := result.Data.Snapshot
		return SnapshotLoadedMsg{Snapshot: SnapshotScreen{SnapshotID: snapshot.SnapshotID, ApprovalID: snapshot.Approval.ApprovalID, RevisionID: snapshot.Approval.ApprovedRevisionID, ActorID: snapshot.Approval.ApprovingActor.ActorID, ApprovedContentHash: snapshot.Approval.ApprovedContentHash, Verified: result.Data.Verified}}
	}
}

func (c WorkspaceCoreCommands) LoadRecovery(intentPath string) tea.Cmd {
	return func() tea.Msg {
		response, err := c.run("recovery_status", map[string]any{"intent_path": intentPath})
		if err != nil {
			return RecoveryLoadedMsg{Err: err}
		}
		if !response.OK {
			return RecoveryLoadedMsg{Err: responseError(response)}
		}
		var result struct {
			Data struct {
				RevisionID string `json:"observed_revision_id"`
				HeadHash   string `json:"observed_head_hash"`
				Token      string `json:"recovery_token"`
				ExpiresIn  int    `json:"expires_in_seconds"`
				Candidates []struct {
					ID           string `json:"candidate_id"`
					RelativePath string `json:"relative_path"`
					Kind         string `json:"kind"`
					Size         int64  `json:"size"`
					Hash         string `json:"content_hash"`
				} `json:"temporary_candidates"`
			} `json:"data"`
		}
		if err := json.Unmarshal(response.Result, &result); err != nil {
			return RecoveryLoadedMsg{Err: err}
		}
		recovery := RecoveryScreen{RevisionID: result.Data.RevisionID, HeadHash: result.Data.HeadHash, Token: result.Data.Token, ExpiresAt: time.Now().Add(time.Duration(result.Data.ExpiresIn) * time.Second)}
		for _, candidate := range result.Data.Candidates {
			recovery.Temporary = append(recovery.Temporary, RecoveryCandidate{ID: candidate.ID, RelativePath: candidate.RelativePath, Kind: candidate.Kind, Size: candidate.Size, Hash: candidate.Hash})
		}
		return RecoveryLoadedMsg{Recovery: recovery}
	}
}

func (c WorkspaceCoreCommands) CleanupRecovery(intentPath string, recovery RecoveryScreen) tea.Cmd {
	return func() tea.Msg {
		response, err := c.run("cleanup_temporary_files", map[string]any{"intent_path": intentPath, "expected_revision_id": recovery.RevisionID, "expected_head_hash": recovery.HeadHash, "operation_id": fmt.Sprintf("cleanup-%d", time.Now().UnixNano()), "recovery_token": recovery.Token, "candidate_ids": recovery.SelectedIDs(), "actor": c.Actor})
		if err != nil {
			return RecoveryCleanupMsg{Err: err}
		}
		if !response.OK {
			code := ""
			if response.Error != nil {
				code = response.Error.Code
			}
			return RecoveryCleanupMsg{Code: code, Err: responseError(response)}
		}
		var result struct {
			Data struct {
				Removed int `json:"removed_count"`
			} `json:"data"`
		}
		if err := json.Unmarshal(response.Result, &result); err != nil {
			return RecoveryCleanupMsg{Err: err}
		}
		return RecoveryCleanupMsg{Removed: result.Data.Removed}
	}
}

func responseError(response protocol.Response) error {
	if response.Error != nil {
		return errors.New(response.Error.Code + ": " + response.Error.Message)
	}
	return errors.New("core operation failed")
}

func decodeRevisionRecord(raw json.RawMessage, reachable bool) RevisionRecord {
	var revision struct {
		ID            string  `json:"revision_id"`
		ParentID      *string `json:"parent_revision_id"`
		Hash          string  `json:"revision_hash"`
		Created       string  `json:"created_at"`
		OperationType string  `json:"operation_type"`
		ActorID       string  `json:"actor_id"`
		Lifecycle     string  `json:"lifecycle_state"`
		Actor         struct {
			ID string `json:"actor_id"`
		} `json:"actor"`
		Operation struct {
			Type string `json:"type"`
		} `json:"operation"`
		Payload struct {
			Lifecycle string `json:"lifecycle_state"`
		} `json:"revision_payload"`
	}
	_ = json.Unmarshal(raw, &revision)
	parent := ""
	if revision.ParentID != nil {
		parent = *revision.ParentID
	}
	operationType, actorID, lifecycle := revision.OperationType, revision.ActorID, revision.Lifecycle
	if operationType == "" {
		operationType = revision.Operation.Type
	}
	if actorID == "" {
		actorID = revision.Actor.ID
	}
	if lifecycle == "" {
		lifecycle = revision.Payload.Lifecycle
	}
	return RevisionRecord{ID: revision.ID, ParentID: parent, Hash: revision.Hash, OperationType: operationType, ActorID: actorID, CreatedAt: revision.Created, Lifecycle: lifecycle, Reachable: reachable}
}

func (c WorkspaceCoreCommands) ListDrafts() tea.Cmd {
	return func() tea.Msg {
		root, err := filepath.Abs(c.DraftRoot)
		if err != nil {
			return DraftListMsg{Root: c.DraftRoot, Err: err}
		}
		entries := make([]DraftEntry, 0)
		err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if path == root {
				return nil
			}
			if entry.Type()&os.ModeSymlink != 0 {
				if entry.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if entry.IsDir() {
				if strings.HasPrefix(entry.Name(), ".") {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.EqualFold(filepath.Ext(entry.Name()), ".json") {
				return nil
			}
			relative, relErr := filepath.Rel(root, path)
			if relErr != nil || strings.HasPrefix(relative, "..") {
				return nil
			}
			entries = append(entries, DraftEntry{Name: filepath.ToSlash(relative), Path: path})
			if len(entries) >= 1000 {
				return filepath.SkipAll
			}
			return nil
		})
		if err != nil {
			return DraftListMsg{Root: root, Err: err}
		}
		sort.SliceStable(entries, func(i, j int) bool { return entries[i].Name < entries[j].Name })
		return DraftListMsg{Root: root, Entries: entries}
	}
}

func provenanceText(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	var text string
	if json.Unmarshal(raw, &text) == nil {
		return text
	}
	var provenance struct {
		ContentOrigin  string `json:"content_origin"`
		OperationType  string `json:"operation_type"`
		OperationActor *struct {
			ActorID string `json:"actor_id"`
		} `json:"operation_actor"`
	}
	if json.Unmarshal(raw, &provenance) == nil && (provenance.ContentOrigin != "" || provenance.OperationType != "") {
		parts := make([]string, 0, 3)
		if provenance.ContentOrigin != "" {
			parts = append(parts, "origin    : "+provenance.ContentOrigin)
		}
		if provenance.OperationType != "" {
			parts = append(parts, "operation : "+provenance.OperationType)
		}
		if provenance.OperationActor != nil && provenance.OperationActor.ActorID != "" {
			parts = append(parts, "actor     : "+provenance.OperationActor.ActorID)
		}
		return strings.Join(parts, "\n")
	}
	var compact bytes.Buffer
	if json.Compact(&compact, raw) == nil {
		return compact.String()
	}
	return string(raw)
}

func (c WorkspaceCoreCommands) run(operation string, payload map[string]any) (protocol.Response, error) {
	payload["operation"] = operation
	body, err := json.Marshal(payload)
	if err != nil {
		return protocol.Response{}, err
	}
	return c.Core.Run(context.Background(), protocol.Request{ProtocolVersion: protocol.Version, RequestID: fmt.Sprintf("workspace-%d", time.Now().UnixNano()), Operation: operation, PayloadSchema: "zintent.command/1", Payload: body})
}
func (c WorkspaceCoreCommands) List() tea.Cmd {
	return func() tea.Msg {
		response, err := c.run("list_intents", map[string]any{"workspace_path": c.WorkspacePath})
		if err != nil {
			return WorkspaceListMsg{Err: err}
		}
		if !response.OK {
			return WorkspaceListMsg{Err: responseError(response)}
		}
		var result struct {
			Data struct {
				Entries []struct {
					IntentID          string `json:"intent_id"`
					DisplayName       string `json:"display_name"`
					IntentPath        string `json:"intent_path"`
					CurrentRevisionID string `json:"current_revision_id"`
					LifecycleState    string `json:"lifecycle_state"`
					ApprovalState     string `json:"approval_state"`
					SnapshotID        string `json:"snapshot_id"`
					BlockerCount      int    `json:"blocker_count"`
				} `json:"entries"`
				Findings []struct {
					RecordID string `json:"record_id"`
					Message  string `json:"message"`
				} `json:"findings"`
			} `json:"data"`
		}
		if err := json.Unmarshal(response.Result, &result); err != nil {
			return WorkspaceListMsg{Err: err}
		}
		entries := []IntentEntry{}
		for _, e := range result.Data.Entries {
			entries = append(entries, IntentEntry{ID: e.IntentID, DisplayName: e.DisplayName, Path: e.IntentPath, Revision: e.CurrentRevisionID, Lifecycle: e.LifecycleState, ApprovalState: e.ApprovalState, SnapshotID: e.SnapshotID, BlockerCount: e.BlockerCount})
		}
		for _, f := range result.Data.Findings {
			entries = append(entries, IntentEntry{ID: f.RecordID, DisplayName: f.RecordID, Corrupt: true, Finding: f.Message})
		}
		return WorkspaceListMsg{Entries: entries}
	}
}
func (c WorkspaceCoreCommands) InspectDraft(source string) tea.Cmd {
	return func() tea.Msg {
		response, err := c.run("inspect_draft", map[string]any{"workspace_path": c.WorkspacePath, "source_path": source, "actor": c.Actor})
		if err != nil {
			return DraftPreviewMsg{Err: err}
		}
		if !response.OK {
			return DraftPreviewMsg{Err: responseError(response)}
		}
		var result struct {
			Data struct {
				SourceHash          string `json:"source_hash"`
				ProposedIntentID    string `json:"proposed_intent_id"`
				ProposedDestination string `json:"proposed_destination"`
				ImportToken         string `json:"import_token"`
				Findings            []struct {
					Message string `json:"message"`
				} `json:"findings"`
			} `json:"data"`
		}
		if err := json.Unmarshal(response.Result, &result); err != nil {
			return DraftPreviewMsg{Err: err}
		}
		findings := []string{}
		for _, f := range result.Data.Findings {
			findings = append(findings, f.Message)
		}
		return DraftPreviewMsg{SourceHash: result.Data.SourceHash, IntentID: result.Data.ProposedIntentID, Destination: result.Data.ProposedDestination, Token: result.Data.ImportToken, Findings: findings}
	}
}
func (c WorkspaceCoreCommands) ImportDraft(preview ImportModal) tea.Cmd {
	return func() tea.Msg {
		response, err := c.run("import_draft", map[string]any{"workspace_path": c.WorkspacePath, "source_path": preview.SourcePath, "destination": preview.Destination, "import_token": preview.Token, "operation_id": fmt.Sprintf("import-%d", time.Now().UnixNano()), "actor": c.Actor})
		if err != nil {
			return DraftImportedMsg{Err: err}
		}
		if !response.OK {
			return DraftImportedMsg{Err: responseError(response)}
		}
		return DraftImportedMsg{IntentID: preview.IntentID, Destination: preview.Destination}
	}
}

func (c CoreCommands) ExecuteComment(operation, commentID, reason string) tea.Cmd {
	return func() tea.Msg {
		payload := map[string]any{"operation": operation, "intent_path": c.IntentPath, "expected_revision_id": c.Expected, "operation_id": fmt.Sprintf("tui-%d", time.Now().UnixNano()), "actor": c.Actor, "comment_id": commentID, "reason": reason}
		body, err := json.Marshal(payload)
		if err != nil {
			return ActionResultMsg{Operation: operation, ItemID: commentID, Err: err}
		}
		response, err := c.Core.Run(context.Background(), protocol.Request{ProtocolVersion: protocol.Version, RequestID: fmt.Sprintf("tui-%d", time.Now().UnixNano()), Operation: operation, PayloadSchema: "zintent.command/1", Payload: body})
		return ActionResultMsg{Operation: operation, ItemID: commentID, Response: response, Err: err}
	}
}

func (c CoreCommands) Execute(operation, itemID string, extra map[string]any) tea.Cmd {
	return func() tea.Msg {
		payload := map[string]any{
			"operation":            operation,
			"intent_path":          c.IntentPath,
			"expected_revision_id": c.Expected,
			"operation_id":         fmt.Sprintf("tui-%d", time.Now().UnixNano()),
			"actor":                c.Actor,
		}
		if itemID != "" {
			payload["item_id"] = itemID
		}
		for key, value := range extra {
			payload[key] = value
		}
		body, err := json.Marshal(payload)
		if err != nil {
			return ActionResultMsg{Operation: operation, ItemID: itemID, Err: err}
		}
		response, err := c.Core.Run(context.Background(), protocol.Request{
			ProtocolVersion: protocol.Version,
			RequestID:       fmt.Sprintf("tui-%d", time.Now().UnixNano()),
			Operation:       operation,
			PayloadSchema:   "zintent.command/1",
			Payload:         body,
		})
		return ActionResultMsg{Operation: operation, ItemID: itemID, Response: response, Err: err}
	}
}

// Reload reads canonical state after every successful mutation. It is kept as
// a separate request so a stale response can never be applied to the in-memory
// model.
func (c CoreCommands) Reload() tea.Cmd {
	return func() tea.Msg {
		body, err := json.Marshal(map[string]any{"operation": "show_intent", "intent_path": c.IntentPath})
		if err != nil {
			return ReloadResultMsg{Err: err}
		}
		response, err := c.Core.Run(context.Background(), protocol.Request{
			ProtocolVersion: protocol.Version,
			RequestID:       fmt.Sprintf("tui-reload-%d", time.Now().UnixNano()),
			Operation:       "show_intent",
			PayloadSchema:   "zintent.command/1",
			Payload:         body,
		})
		return ReloadResultMsg{Response: response, Err: err}
	}
}

// PreviewEdit and ApplyEdit expose the two-step edit contract used by both
// the CLI and TUI. ApplyEdit accepts only the token returned by PreviewEdit;
// the core performs the final binding and stale check.
func (c CoreCommands) PreviewEdit(itemID, statement string) tea.Cmd {
	return c.Execute("preview_edit", itemID, map[string]any{"statement": statement})
}

func (c CoreCommands) ApplyEdit(itemID, statement, previewToken string) tea.Cmd {
	return c.Execute("edit_item", itemID, map[string]any{"statement": statement, "preview_token": previewToken})
}
