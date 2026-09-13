package ui

import (
	"context"
	"encoding/json"
	"fmt"
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
