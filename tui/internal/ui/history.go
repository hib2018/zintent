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
	Selected, Height                   int
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
	if s.BaseID != "" || s.TargetID != "" {
		fmt.Fprintf(&b, "DIFF [%s → %s]\n", shortRef(s.BaseID), shortRef(s.TargetID))
		for _, c := range s.Changes {
			fmt.Fprintf(&b, "[%s] %s\n- %s\n+ %s\nprovenance: %s\n", c.Kind, c.ItemID, c.Before, c.After, c.Provenance)
		}
		return b.String()
	}
	if s.Inspected != nil {
		r := s.Inspected
		fmt.Fprintf(&b, "REVISION [verified]\n  ID        : %s\n  Hash      : %s\n  Parent    : %s\n  Operation : [%s]\n  Actor     : %s\n  Lifecycle : [%s]\n\nj/k return to history  Esc back", r.ID, r.Hash, fallback(r.ParentID, "-"), fallback(r.OperationType, "-"), fallback(r.ActorID, "-"), fallback(r.Lifecycle, "-"))
		return b.String()
	}

	position := 0
	if len(s.Revisions) > 0 {
		position = min(s.Selected, len(s.Revisions)-1) + 1
	}
	fmt.Fprintf(&b, "HISTORY [verified] [%d/%d]\n", position, len(s.Revisions))
	capacity := len(s.Revisions)
	if s.Height > 0 {
		capacity = max(1, (s.Height-3)/2)
	}
	start := max(0, min(s.Selected-capacity+1, len(s.Revisions)-capacity))
	end := min(start+capacity, len(s.Revisions))
	for index := start; index < end; index++ {
		r := s.Revisions[index]
		top := fmt.Sprintf("  [%s] [%s] %s", fallback(r.Lifecycle, "-"), fallback(r.OperationType, "-"), shortRef(r.ID))
		if index == s.Selected {
			top = highlightTopLine("→ " + strings.TrimPrefix(top, "  "))
		}
		b.WriteString(top + "\n")
		fmt.Fprintf(&b, "    parent:%s  actor:%s  time:%s\n", shortRef(fallback(r.ParentID, "-")), fallback(r.ActorID, "-"), fallback(r.CreatedAt, "-"))
	}
	if end == len(s.Revisions) && len(s.Orphans) > 0 {
		fmt.Fprintf(&b, "ORPHANS [not canonical:%d]\n", len(s.Orphans))
		for _, r := range s.Orphans {
			fmt.Fprintf(&b, "  [orphan] %s\n", shortRef(r.ID))
		}
	}
	if len(s.Revisions) > 0 {
		b.WriteString("\nj/k select  Enter inspect  d diff with parent  Esc back")
	}
	return b.String()
}
