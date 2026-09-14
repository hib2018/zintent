package ui

import (
	"encoding/json"

	tea "charm.land/bubbletea/v2"
)

type Item struct {
	ID, Kind, Statement, Status string
	Provenance, Rationale       string
}

type ReviewScreen struct {
	Items          []Item
	SelectedID     string
	Offset, Height int
	FocusDetail    bool
}

func (s ReviewScreen) Reload(items []Item) ReviewScreen {
	selected := s.SelectedID
	s.Items = append(s.Items[:0], items...)
	s.SelectedID = ""
	for _, item := range s.Items {
		if item.ID == selected {
			s.SelectedID = selected
			return s
		}
	}
	for _, item := range s.Items {
		if item.Status == "unreviewed" || item.Status == "" {
			s.SelectedID = item.ID
			return s
		}
	}
	if len(s.Items) > 0 {
		s.SelectedID = s.Items[0].ID
	}
	return s
}
func (s ReviewScreen) Selected() *Item {
	for i := range s.Items {
		if s.Items[i].ID == s.SelectedID {
			return &s.Items[i]
		}
	}
	return nil
}
func (s ReviewScreen) JumpBlocker() ReviewScreen {
	for _, item := range s.Items {
		if item.Status == "unreviewed" || item.Status == "" {
			s.SelectedID = item.ID
			break
		}
	}
	return s
}
func (s ReviewScreen) Visible() []Item {
	start := min(max(s.Offset, 0), len(s.Items))
	height := s.Height
	if height <= 0 {
		height = len(s.Items)
	}
	end := min(start+height, len(s.Items))
	return s.Items[start:end]
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
	IntentPath, ExpectedRevision                                 string
	ResumeNotice                                                 string
	Input, PreviewToken, Before, After, Challenge, ApprovalToken string
	RevisionHash, ApprovedContentHash, SnapshotPath              string
}

func New(items []Item) Model { return Model{Items: items, Lifecycle: "draft"} }

// ResumeBlockers reports the actionable work still preventing completion.
func (m Model) ResumeBlockers() []string {
	blockers := make([]string, 0)
	for _, item := range m.Items {
		if item.Status == "unreviewed" || item.Status == "" {
			blockers = append(blockers, "unreviewed item: "+item.ID)
		}
	}
	if m.Lifecycle == "review_complete" && len(blockers) == 0 {
		return blockers
	}
	return blockers
}

// Restore selects the same stable item ID after a process restart.
func (m Model) Restore(revision, lifecycle, selectedID string, items []Item) Model {
	m.Revision, m.Lifecycle, m.Items = revision, lifecycle, items
	m.Selected = 0
	for i, item := range items {
		if item.ID == selectedID {
			m.Selected = i
			break
		}
	}
	m.ResumeNotice = "resumed from canonical artifact " + revision
	return m
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c":
			m.Quitting = true
			return m, tea.Quit
		case "q":
			if m.Modal != "" {
				m.Input += "q"
				break
			}
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
		case "f":
			if m.Executor != nil {
				m.Status = "completing review"
				return m, m.Executor.Execute("complete_review", "", nil)
			}
		case "p":
			if m.Executor != nil && m.Lifecycle == "review_complete" {
				m.Modal = "approval-loading"
				return m, m.Executor.Execute("prepare_approval", "", map[string]any{"interactive_tty": true})
			}
			m.Status = "approval requires a review-complete Intent"
		case "enter":
			if m.Modal != "" {
				itemID := ""
				if len(m.Items) > 0 && m.Selected < len(m.Items) {
					itemID = m.Items[m.Selected].ID
				}
				m.Status = m.PendingAction + " requested for " + itemID
				if m.Executor != nil {
					operation := map[string]string{"accept": "accept_item", "comment": "add_comment", "reject": "reject_item"}[m.PendingAction]
					extra := map[string]any{}
					if m.PendingAction == "comment" {
						extra["body"] = m.Input
					}
					if m.PendingAction == "reject" {
						extra["rationale"] = m.Input
					}
					if m.PendingAction == "edit-preview" {
						m.Modal = "preview-loading"
						return m, m.Executor.Execute("preview_edit", itemID, map[string]any{"statement": m.Input})
					}
					if m.PendingAction == "edit-confirm" {
						m.Modal = "submitting"
						return m, m.Executor.Execute("edit_item", itemID, map[string]any{"statement": m.After, "preview_token": m.PreviewToken})
					}
					if m.PendingAction == "approval-confirm" {
						if m.Input != m.Challenge {
							m.Status = "challenge mismatch"
							break
						}
						m.Modal = "submitting"
						return m, m.Executor.Execute("approve_intent", "", map[string]any{"interactive_tty": true, "confirmation_token": m.ApprovalToken, "challenge_response": m.Input})
					}
					if operation != "" && (m.PendingAction == "accept" || m.Input != "") {
						m.Modal, m.PendingAction = "", ""
						return m, m.Executor.Execute(operation, itemID, extra)
					}
				}
				m.Status = m.PendingAction + " confirmed for " + itemID
				m.Modal, m.PendingAction = "", ""
			}
		case "backspace":
			if m.Modal != "" && len(m.Input) > 0 {
				m.Input = m.Input[:len(m.Input)-1]
			}
		case "esc":
			m.Modal, m.PendingAction, m.Input, m.PreviewToken, m.ApprovalToken = "", "", "", "", ""
		default:
			if m.Modal != "" && msg.Text != "" {
				m.Input += msg.Text
			}
		}
	case ActionResultMsg:
		if msg.Err != nil {
			m.Modal = "error"
			m.Status = "mutation failed: " + msg.Err.Error()
		} else if !msg.Response.OK {
			if msg.Response.Error != nil {
				m.Status = msg.Response.Error.Code + ": " + msg.Response.Error.Message
			} else {
				m.Status = "mutation failed"
			}
		} else if msg.Operation == "preview_edit" {
			var result struct {
				Data struct {
					Preview struct {
						Before, After, Token string `json:"-"`
						BeforeStatement      string `json:"before_statement"`
						AfterStatement       string `json:"after_statement"`
						PreviewToken         string `json:"preview_token"`
					} `json:"preview"`
				} `json:"data"`
			}
			if err := json.Unmarshal(msg.Response.Result, &result); err != nil {
				m.Status = "preview decode failed"
				break
			}
			m.Before = result.Data.Preview.BeforeStatement
			m.After = result.Data.Preview.AfterStatement
			m.PreviewToken = result.Data.Preview.PreviewToken
			m.PendingAction = "edit-confirm"
			m.Modal = "edit-confirm"
			m.Status = "review exact before/after, then confirm"
		} else if msg.Operation == "prepare_approval" {
			var result struct {
				Data struct {
					Confirmation struct {
						TokenID               string `json:"token_id"`
						Challenge             string `json:"challenge"`
						ConfirmedRevisionID   string `json:"confirmed_revision_id"`
						ConfirmedRevisionHash string `json:"confirmed_revision_hash"`
						ApprovedContentHash   string `json:"approved_content_hash"`
					} `json:"confirmation"`
				} `json:"data"`
			}
			if err := json.Unmarshal(msg.Response.Result, &result); err != nil {
				m.Status = "approval decode failed"
				break
			}
			c := result.Data.Confirmation
			m.ApprovalToken = c.TokenID
			m.Challenge = c.Challenge
			m.RevisionHash = c.ConfirmedRevisionHash
			m.ApprovedContentHash = c.ApprovedContentHash
			m.Input = ""
			m.PendingAction = "approval-confirm"
			m.Modal = "approval-confirm"
			m.Status = "type the displayed challenge exactly"
		} else {
			m.Modal, m.PendingAction, m.Input = "", "", ""
			if msg.Operation == "approve_intent" {
				var result struct {
					Data struct {
						SnapshotPath string `json:"snapshot_path"`
					} `json:"data"`
				}
				if json.Unmarshal(msg.Response.Result, &result) == nil {
					m.SnapshotPath = result.Data.SnapshotPath
				}
			}
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
