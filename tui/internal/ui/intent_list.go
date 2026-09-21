package ui

import (
	"sort"
	"strconv"
	"strings"
)

type IntentEntry struct {
	ID, DisplayName, Path, Revision, Lifecycle, ApprovalState, SnapshotID string
	BlockerCount                                                          int
	Corrupt                                                               bool
	Finding                                                               string
}
type IntentListScreen struct {
	Entries            []IntentEntry
	Filter, SelectedID string
	FilterEditing      bool
	Offset, Height     int
}

func (s IntentListScreen) Visible() []IntentEntry {
	result := []IntentEntry{}
	query := strings.ToLower(s.Filter)
	for _, e := range s.Entries {
		if query == "" || strings.Contains(strings.ToLower(e.ID+" "+e.DisplayName+" "+e.Lifecycle), query) {
			result = append(result, e)
		}
	}
	return result
}
func (s IntentListScreen) Reload(entries []IntentEntry) IntentListScreen {
	selected := s.SelectedID
	s.Entries = append(s.Entries[:0], entries...)
	sort.SliceStable(s.Entries, func(i, j int) bool { return s.Entries[i].ID < s.Entries[j].ID })
	s.SelectedID = ""
	for _, e := range s.Visible() {
		if e.ID == selected {
			s.SelectedID = selected
			return s
		}
	}
	if visible := s.Visible(); len(visible) > 0 {
		s.SelectedID = visible[0].ID
	}
	return s
}
func (s IntentListScreen) Selected() *IntentEntry {
	for i := range s.Entries {
		if s.Entries[i].ID == s.SelectedID {
			return &s.Entries[i]
		}
	}
	return nil
}
func (s *IntentListScreen) Move(delta int) {
	visible := s.Visible()
	if len(visible) == 0 {
		return
	}
	index := 0
	for i, e := range visible {
		if e.ID == s.SelectedID {
			index = i
			break
		}
	}
	index = max(0, min(index+delta, len(visible)-1))
	s.SelectedID = visible[index].ID
}
func (s IntentListScreen) View() string {
	var b strings.Builder
	b.WriteString("Intent list\nFilter: " + s.Filter + "\n")
	if len(s.Visible()) == 0 {
		b.WriteString("No Intent selected.\n")
	}
	visible := s.Visible()
	start := min(max(s.Offset, 0), len(visible))
	height := s.Height
	if height <= 0 {
		height = 20
	}
	end := min(start+height, len(visible))
	for _, e := range visible[start:end] {
		if e.ID == s.SelectedID {
			b.WriteString("> ")
		} else {
			b.WriteString("  ")
		}
		b.WriteString(shortRef(e.ID) + "  [" + e.Lifecycle + "]  blockers:")
		b.WriteString(strconv.Itoa(e.BlockerCount))
		if e.Corrupt {
			b.WriteString(" CORRUPT " + e.Finding)
		}
		b.WriteByte('\n')
	}
	return b.String()
}
