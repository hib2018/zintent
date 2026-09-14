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
	for _, r := range s.Revisions {
		fmt.Fprintf(&b, "%s <- %s  %s  actor:%s  %s\n", r.ID, r.ParentID, r.OperationType, r.ActorID, r.CreatedAt)
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
	return b.String()
}
