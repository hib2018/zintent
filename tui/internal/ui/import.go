package ui

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

type DraftEntry struct{ Name, Path string }

type DraftPicker struct {
	Root, SelectedPath string
	Entries            []DraftEntry
	Index              int
}

func (p DraftPicker) Reload(entries []DraftEntry) DraftPicker {
	selected := p.SelectedPath
	p.Entries = append(p.Entries[:0], entries...)
	sort.SliceStable(p.Entries, func(i, j int) bool { return p.Entries[i].Name < p.Entries[j].Name })
	p.Index, p.SelectedPath = 0, ""
	for i, entry := range p.Entries {
		if entry.Path == selected {
			p.Index, p.SelectedPath = i, entry.Path
			return p
		}
	}
	if len(p.Entries) > 0 {
		p.SelectedPath = p.Entries[0].Path
	}
	return p
}

func (p *DraftPicker) Move(delta int) {
	if len(p.Entries) == 0 {
		return
	}
	p.Index = min(max(p.Index+delta, 0), len(p.Entries)-1)
	p.SelectedPath = p.Entries[p.Index].Path
}

func (p DraftPicker) View() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Root: %s\n", p.Root)
	if len(p.Entries) == 0 {
		b.WriteString("No Draft JSON files found.\n")
		return b.String()
	}
	start := max(0, p.Index-4)
	end := min(len(p.Entries), start+9)
	if end-start < 9 {
		start = max(0, end-9)
	}
	for i, entry := range p.Entries[start:end] {
		absoluteIndex := start + i
		marker := "  "
		if absoluteIndex == p.Index {
			marker = "→ "
		}
		b.WriteString(marker + entry.Name + "\n")
	}
	if len(p.Entries) > end {
		fmt.Fprintf(&b, "  … %d more\n", len(p.Entries)-end)
	}
	return b.String()
}

type ImportModal struct {
	Phase                                                        ModalPhase
	SourcePath, SourceHash, IntentID, Destination, Token, Status string
	Findings                                                     []string
	ExpiresAt                                                    time.Time
}

func (m ImportModal) CanSubmit(now time.Time) bool {
	return m.Phase == ModalConfirming && m.Token != "" && now.Before(m.ExpiresAt) && len(m.Findings) == 0
}
func (m *ImportModal) ResetFailure(status string) {
	*m = ImportModal{Phase: ModalError, Status: status}
}
func (m *ImportModal) Clear() { *m = ImportModal{Phase: ModalClosed} }
