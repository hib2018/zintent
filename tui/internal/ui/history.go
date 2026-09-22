package ui

import (
	"fmt"
	"strings"
)

type RevisionRecord struct {
	ID, ParentID, Hash, OperationID, OperationType, ActorID, CreatedAt, Lifecycle string
	Reachable                                                                     bool
}
type HistoryScreen struct {
	Revisions, Orphans                 []RevisionRecord
	Changes                            []ItemChange
	BaseID, TargetID, SelectedChangeID string
	Selected                           int
	Inspected                          *RevisionRecord
}

func (s *HistoryScreen) Move(delta int) {
	if len(s.Revisions) == 0 {
		return
	}
	s.Selected = max(0, min(s.Selected+delta, len(s.Revisions)-1))
	s.Inspected = nil
}

func (s HistoryScreen) SelectedRevisionRecord() *RevisionRecord {
	if s.Selected < 0 || s.Selected >= len(s.Revisions) {
		return nil
	}
	return &s.Revisions[s.Selected]
}

func (s HistoryScreen) SelectPair(base, target string) HistoryScreen {
	s.BaseID, s.TargetID = base, target
	return s
}
func (s HistoryScreen) SelectedRevision(id string) *RevisionRecord {
	for i := range s.Revisions {
		if s.Revisions[i].ID == id {
			return &s.Revisions[i]
		}
	}
	for i := range s.Orphans {
		if s.Orphans[i].ID == id {
			return &s.Orphans[i]
		}
	}
	return nil
}

type ItemChange struct{ ItemID, Kind, Before, After, Provenance string }

func (s HistoryScreen) View() string {
	var b strings.Builder
	b.WriteString("Verified history\n")
	for index, r := range s.Revisions {
		line := fmt.Sprintf("  %s <- %s  %s  actor:%s  %s", shortRef(r.ID), shortRef(r.ParentID), r.OperationType, r.ActorID, r.CreatedAt)
		if index == s.Selected {
			line = highlightTopLine("→ " + strings.TrimPrefix(line, "  "))
		}
		b.WriteString(line + "\n")
	}
	if len(s.Orphans) > 0 {
		b.WriteString("Orphans (not canonical)\n")
		for _, r := range s.Orphans {
			b.WriteString(r.ID + "\n")
		}
	}
	if s.BaseID != "" || s.TargetID != "" {
		fmt.Fprintf(&b, "Diff %s → %s\n", s.BaseID, s.TargetID)
		for _, c := range s.Changes {
			fmt.Fprintf(&b, "%s [%s]\n- %s\n+ %s\nprovenance: %s\n", c.ItemID, c.Kind, c.Before, c.After, c.Provenance)
		}
	}
	if s.Inspected != nil {
		r := s.Inspected
		fmt.Fprintf(&b, "\nVERIFIED REVISION\n  ID        : %s\n  Hash      : %s\n  Parent    : %s\n  Operation : %s\n  Actor     : %s\n  Lifecycle : %s\n", r.ID, r.Hash, r.ParentID, r.OperationType, r.ActorID, r.Lifecycle)
	}
	if len(s.Revisions) > 0 && s.BaseID == "" && s.TargetID == "" {
		b.WriteString("\nj/k select  Enter inspect  d diff with parent  Esc back")
	}
	return b.String()
}
