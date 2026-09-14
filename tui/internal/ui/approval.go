package ui

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
)

// ApprovalModel is deliberately separate from review navigation. It displays
// the exact candidate and challenge returned by the core; it never invents or
// submits a challenge response.
type ApprovalModel struct {
	IntentID, RevisionID, RevisionHash, ApprovedContentHash string
	Challenge, Response, Status                             string
	Blockers                                                []string
	Focused                                                 bool
	TokenID                                                 string
	ExpiresAt                                               time.Time
	IncludedCount, ExcludedCount                            int
	Cancelled                                               bool
}

func NewApproval(intentID, revisionID, revisionHash, approvedContentHash, challenge string, blockers []string) ApprovalModel {
	return ApprovalModel{IntentID: intentID, RevisionID: revisionID, RevisionHash: revisionHash, ApprovedContentHash: approvedContentHash, Challenge: challenge, Blockers: blockers}
}

func (m ApprovalModel) Eligible() bool { return len(m.Blockers) == 0 }
func (m ApprovalModel) Init() tea.Cmd  { return nil }

func (m ApprovalModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyPressMsg); ok {
		if m.Focused {
			switch key.String() {
			case "esc":
				m.Response = ""
				m.TokenID = ""
				m.Focused = false
				m.Cancelled = true
				m.Status = "approval cancelled"
			case "backspace":
				if len(m.Response) > 0 {
					m.Response = m.Response[:len(m.Response)-1]
				}
			case "enter":
				if m.Ready(time.Now()) {
					m.Status = "approval confirmation ready"
				} else {
					m.Status = "challenge mismatch or expired"
				}
			default:
				if key.Text != "" {
					m.Response += key.Text
				}
			}
			return m, nil
		}
		switch key.String() {
		case "esc", "q":
			m.Status = "approval cancelled"
			m.Response = ""
			m.TokenID = ""
			m.Cancelled = true
		case "enter":
			if m.Eligible() {
				m.Focused = true
				m.Response = ""
				m.Status = "type the fresh challenge exactly"
			} else {
				m.Status = "approval blocked"
			}
		}
	}
	return m, nil
}

func (m ApprovalModel) Ready(now time.Time) bool {
	return m.Eligible() && m.TokenID != "" && m.Response == m.Challenge && (m.ExpiresAt.IsZero() || now.Before(m.ExpiresAt))
}
func (m *ApprovalModel) Invalidate() {
	m.Response = ""
	m.TokenID = ""
	m.Focused = false
	m.Status = "approval token discarded"
}

func (m ApprovalModel) View() tea.View {
	var b strings.Builder
	b.WriteString("zintent approval\n\n")
	b.WriteString("Intent: ")
	b.WriteString(m.IntentID)
	b.WriteString("\n")
	b.WriteString("Revision: ")
	b.WriteString(m.RevisionID)
	b.WriteString("\n")
	b.WriteString("Revision hash: ")
	b.WriteString(m.RevisionHash)
	b.WriteString("\n")
	b.WriteString("Approved content hash: ")
	b.WriteString(m.ApprovedContentHash)
	b.WriteString("\n")
	b.WriteString("Challenge: ")
	b.WriteString(m.Challenge)
	b.WriteString("\n\n")
	b.WriteString(fmt.Sprintf("Included / excluded: %d / %d\n", m.IncludedCount, m.ExcludedCount))
	if len(m.Blockers) > 0 {
		b.WriteString("Blockers:\n")
		for _, blocker := range m.Blockers {
			b.WriteString("- ")
			b.WriteString(blocker)
			b.WriteByte('\n')
		}
	} else {
		b.WriteString("Eligible for approval\n")
	}
	if m.Status != "" {
		b.WriteString("\n")
		b.WriteString(m.Status)
		b.WriteByte('\n')
	}
	b.WriteString("\nEnter approve  Esc cancel\n")
	return tea.NewView(b.String())
}
