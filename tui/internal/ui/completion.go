package ui

import "strings"

type CompletionScreen struct {
	Blockers   []string
	Selected   int
	RevisionID string
	Lifecycle  string
}

func (s CompletionScreen) JumpTarget() string {
	blocker := s.SelectedBlocker()
	for _, prefix := range []string{"unreviewed item: ", "open comment: "} {
		if strings.HasPrefix(blocker, prefix) {
			return strings.TrimPrefix(blocker, prefix)
		}
	}
	return ""
}

func (s CompletionScreen) View() string {
	var b strings.Builder
	b.WriteString("COMPLETION\n")
	b.WriteString("  Revision : " + shortRef(s.RevisionID) + "\n")
	if s.Lifecycle == "review_complete" || s.Lifecycle == "approved" {
		b.WriteString("  Status   : already complete (" + s.Lifecycle + ")\n")
		return b.String()
	}
	if s.Eligible() {
		b.WriteString("  Status   : eligible\n\nPress Enter to complete review.")
		return b.String()
	}
	b.WriteString("  Status   : blocked\n\nBLOCKERS\n")
	for index, blocker := range s.Blockers {
		marker := "  "
		if index == s.Selected {
			marker = "→ "
		}
		b.WriteString(marker + blocker + "\n")
	}
	b.WriteString("\nEnter=open blocker  j/k select  Esc back")
	return b.String()
}

func (s CompletionScreen) Eligible() bool {
	return len(s.Blockers) == 0 && s.RevisionID != "" && s.Lifecycle != "review_complete" && s.Lifecycle != "approved"
}
func (s CompletionScreen) SelectedBlocker() string {
	if s.Selected < 0 || s.Selected >= len(s.Blockers) {
		return ""
	}
	return s.Blockers[s.Selected]
}
