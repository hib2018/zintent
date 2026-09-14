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
	for _, c := range s.Temporary {
		fmt.Fprintf(&b, "[%t] %s %s %d bytes\n", c.Selected, c.ID, c.Kind, c.Size)
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
	return b.String()
}

type RecoveryScreen struct {
	RevisionID, HeadHash, Token string
	ExpiresAt                   time.Time
	Temporary, Orphans          []RecoveryCandidate
	Status                      string
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
