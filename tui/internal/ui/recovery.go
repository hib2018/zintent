package ui

import (
	"fmt"
	"strings"
	"time"
)

type RecoveryCandidate struct {
	ID, RelativePath, Kind, Hash string
	Size                         int64
	Selected                     bool
	Protected                    bool
}

func (s RecoveryScreen) View() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Recovery status\nRevision: %s\nHEAD hash: %s\n", s.RevisionID, s.HeadHash)
	for index, c := range s.Temporary {
		line := fmt.Sprintf("  [%t] %s %s %d bytes", c.Selected, c.ID, c.Kind, c.Size)
		if index == s.Cursor {
			line = highlightTopLine("→ " + strings.TrimPrefix(line, "  "))
		}
		b.WriteString(line + "\n")
	}
	if len(s.Orphans) > 0 {
		b.WriteString("Protected orphans\n")
		for _, c := range s.Orphans {
			b.WriteString(c.ID + "\n")
		}
	}
	if s.Status != "" {
		b.WriteString(s.Status + "\n")
	}
	b.WriteString("\nj/k select  Space toggle  x cleanup  Esc back\n")
	return b.String()
}

type RecoveryScreen struct {
	RevisionID, HeadHash, Token string
	ExpiresAt                   time.Time
	Temporary, Orphans          []RecoveryCandidate
	Status                      string
	Cursor                      int
}

func (s RecoveryScreen) SelectedIDs() []string {
	ids := []string{}
	for _, c := range s.Temporary {
		if c.Selected && !c.Protected {
			ids = append(ids, c.ID)
		}
	}
	return ids
}
func (s RecoveryScreen) CanCleanup(now time.Time) bool {
	return s.Token != "" && now.Before(s.ExpiresAt) && len(s.SelectedIDs()) > 0
}
func (s *RecoveryScreen) Invalidate(reason string) {
	s.Token = ""
	s.Status = reason
	for i := range s.Temporary {
		s.Temporary[i].Selected = false
	}
}
