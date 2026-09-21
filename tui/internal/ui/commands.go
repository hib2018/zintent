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
			return WorkspaceCanonicalMsg{Err: errors.New(response.Error.Message)}
		}
		var result struct {
			Data struct {
				Intent struct {
					IntentID        string `json:"intent_id"`
					RevisionID      string `json:"revision_id"`
					Lifecycle       string `json:"lifecycle_state"`
					RevisionPayload struct {
						Items []struct {
							ID         string          `json:"item_id"`
							Kind       string          `json:"kind"`
							Statement  string          `json:"statement"`
							Status     string          `json:"review_status"`
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
		if err := json.Unmarshal(response.Result, &result); err != nil {
			return WorkspaceCanonicalMsg{Err: err}
		}
		items := make([]Item, 0, len(result.Data.Intent.RevisionPayload.Items))
		for _, item := range result.Data.Intent.RevisionPayload.Items {
			items = append(items, Item{ID: item.ID, Kind: item.Kind, Statement: item.Statement, Status: item.Status, Provenance: provenanceText(item.Provenance), Rationale: item.Rationale})
		}
		comments := make([]CommentRecord, 0, len(result.Data.Intent.RevisionPayload.Comments))
		for _, comment := range result.Data.Intent.RevisionPayload.Comments {
			comments = append(comments, CommentRecord{ID: comment.ID, TargetItemID: comment.TargetItemID, Body: comment.Body, Status: comment.Status})
		}
		return WorkspaceCanonicalMsg{IntentID: result.Data.Intent.IntentID, RevisionID: result.Data.Intent.RevisionID, Lifecycle: result.Data.Intent.Lifecycle, IntentPath: filepath.Join(c.WorkspacePath, intentPath), Items: items, Comments: comments}
	}
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
			return WorkspaceListMsg{Err: errors.New(response.Error.Message)}
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
			return DraftPreviewMsg{Err: errors.New(response.Error.Message)}
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
			return DraftImportedMsg{Err: errors.New(response.Error.Message)}
		}
		return DraftImportedMsg{IntentID: preview.IntentID, Destination: preview.Destination}
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
