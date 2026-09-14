package ui

import "strings"

type CompletionScreen struct {
	Blockers   []string
	Selected   int
	RevisionID string
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

func (s CompletionScreen) Eligible() bool { return len(s.Blockers) == 0 && s.RevisionID != "" }
func (s CompletionScreen) SelectedBlocker() string {
	if s.Selected < 0 || s.Selected >= len(s.Blockers) {
		return ""
	}
	return s.Blockers[s.Selected]
}
