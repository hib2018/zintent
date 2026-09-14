package ui

import (
	"strings"

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
}

func NewApproval(intentID, revisionID, revisionHash, approvedContentHash, challenge string, blockers []string) ApprovalModel {
	return ApprovalModel{IntentID: intentID, RevisionID: revisionID, RevisionHash: revisionHash, ApprovedContentHash: approvedContentHash, Challenge: challenge, Blockers: blockers}
}

func (m ApprovalModel) Eligible() bool { return len(m.Blockers) == 0 }
func (m ApprovalModel) Init() tea.Cmd  { return nil }

func (m ApprovalModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyPressMsg); ok {
		switch key.String() {
		case "esc", "q":
			m.Status = "approval cancelled"
		case "enter":
			if m.Eligible() {
				m.Focused = true
				m.Status = "challenge response required in the approval TTY"
			} else {
				m.Status = "approval blocked"
			}
		}
	}
	return m, nil
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
