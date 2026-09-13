package ui

import (
	"encoding/json"

	tea "charm.land/bubbletea/v2"
)

type Item struct {
	ID, Kind, Statement, Status string
}

type Model struct {
	IntentID, Revision, Lifecycle, Actor string
	Items                                []Item
	Selected                             int
	Width, Height                        int
	Findings                             []string
	Quitting                             bool
	Modal                                string
	PendingAction                        string
	Status                               string
	Executor                             interface {
		Execute(operation, itemID string, extra map[string]any) tea.Cmd
	}
	IntentPath, ExpectedRevision string
}

func New(items []Item) Model { return Model{Items: items, Lifecycle: "draft"} }

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.Quitting = true
			return m, tea.Quit
		case "j", "down":
			if m.Selected+1 < len(m.Items) {
				m.Selected++
			}
		case "k", "up":
			if m.Selected > 0 {
				m.Selected--
			}
		case "a", "e", "c", "x":
			if len(m.Items) > 0 {
				m.PendingAction = map[string]string{"a": "accept", "e": "edit-preview", "c": "comment", "x": "reject"}[msg.String()]
				m.Modal = m.PendingAction
			}
		case "enter":
			if m.Modal != "" {
				itemID := m.Items[m.Selected].ID
				m.Status = m.PendingAction + " requested for " + itemID
				if m.Executor != nil {
					operation := map[string]string{"accept": "accept_item", "reject": "reject_item"}[m.PendingAction]
					if operation != "" {
						m.Modal, m.PendingAction = "", ""
						return m, m.Executor.Execute(operation, itemID, nil)
					}
				}
				m.Status = m.PendingAction + " confirmed for " + itemID
				m.Modal, m.PendingAction = "", ""
			}
		case "esc":
			m.Modal, m.PendingAction = "", ""
		}
	case ActionResultMsg:
		m.Modal, m.PendingAction = "", ""
		if msg.Err != nil {
			m.Status = "mutation failed: " + msg.Err.Error()
		} else if !msg.Response.OK {
			if msg.Response.Error != nil {
				m.Status = msg.Response.Error.Code + ": " + msg.Response.Error.Message
			} else {
				m.Status = "mutation failed"
			}
		} else {
			m.Status = msg.Operation + " applied for " + msg.ItemID
			if reloader, ok := m.Executor.(interface{ Reload() tea.Cmd }); ok {
				return m, reloader.Reload()
			}
		}
	case ReloadResultMsg:
		if msg.Err != nil {
			m.Status = "reload failed: " + msg.Err.Error()
			break
		}
		if !msg.Response.OK {
			m.Status = "reload failed"
			break
		}
		var result struct {
			Data struct {
				Intent struct {
					RevisionID string `json:"revision_id"`
					Lifecycle  string `json:"lifecycle_state"`
					Payload    struct {
						Items []struct {
							ID        string `json:"item_id"`
							Kind      string `json:"kind"`
							Statement string `json:"statement"`
							Status    string `json:"review_status"`
						} `json:"items"`
					} `json:"revision_payload"`
				} `json:"intent"`
			} `json:"data"`
		}
		if err := json.Unmarshal(msg.Response.Result, &result); err != nil {
			m.Status = "reload failed: " + err.Error()
			break
		}
		selectedID := ""
		if len(m.Items) > 0 && m.Selected < len(m.Items) {
			selectedID = m.Items[m.Selected].ID
		}
		m.Revision, m.Lifecycle = result.Data.Intent.RevisionID, result.Data.Intent.Lifecycle
		m.Items = make([]Item, 0, len(result.Data.Intent.Payload.Items))
		m.Selected = 0
		for _, item := range result.Data.Intent.Payload.Items {
			m.Items = append(m.Items, Item{ID: item.ID, Kind: item.Kind, Statement: item.Statement, Status: item.Status})
			if item.ID == selectedID {
				m.Selected = len(m.Items) - 1
			}
		}
		m.ExpectedRevision = m.Revision
		if setter, ok := m.Executor.(interface{ SetExpected(string) }); ok {
			setter.SetExpected(m.Revision)
		}
		m.Status = "reloaded " + m.Revision
	case tea.WindowSizeMsg:
		m.Width, m.Height = msg.Width, msg.Height
	}
	return m, nil
}

func (m Model) View() tea.View {
	return renderView(m)
}
