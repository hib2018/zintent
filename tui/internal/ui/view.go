package ui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
)

func renderView(m Model) tea.View {
	if m.Quitting {
		return tea.NewView("Review closed.\n")
	}
	var b strings.Builder
	b.WriteString("zintent  ")
	b.WriteString(m.Lifecycle)
	b.WriteString("  rev:")
	b.WriteString(m.Revision)
	b.WriteString("  actor:")
	b.WriteString(m.Actor)
	b.WriteString("\n\n")

	start, end := 0, len(m.Items)
	if m.Height > 8 && len(m.Items) > m.Height-8 {
		visible := m.Height - 8
		start = max(0, min(m.Selected-visible/2, len(m.Items)-visible))
		end = start + visible
	}
	for i := start; i < end; i++ {
		item := m.Items[i]
		if i == m.Selected {
			b.WriteString("> ")
		} else {
			b.WriteString("  ")
		}
		b.WriteString(item.ID)
		b.WriteString(" [")
		b.WriteString(item.Status)
		b.WriteString("] ")
		b.WriteString(item.Statement)
		b.WriteByte('\n')
	}

	if len(m.Items) > 0 {
		selected := m.Items[m.Selected]
		b.WriteString("\n--- detail ---\n")
		b.WriteString(selected.ID)
		b.WriteString(" / ")
		b.WriteString(selected.Kind)
		b.WriteString("\n")
		b.WriteString(selected.Statement)
		b.WriteByte('\n')
		if selected.Provenance != "" {
			b.WriteString("provenance: ")
			b.WriteString(selected.Provenance)
			b.WriteByte('\n')
		}
		if selected.Rationale != "" {
			b.WriteString("rationale: ")
			b.WriteString(selected.Rationale)
			b.WriteByte('\n')
		}
	}
	if m.Modal != "" && len(m.Items) > 0 {
		b.WriteString("\nConfirm ")
		b.WriteString(m.Modal)
		b.WriteString(" for ")
		b.WriteString(m.Items[m.Selected].ID)
		b.WriteString("? Enter=confirm Esc=cancel\n")
		if m.PendingAction == "edit-confirm" {
			b.WriteString("before: ")
			b.WriteString(m.Before)
			b.WriteString("\nafter: ")
			b.WriteString(m.After)
			b.WriteByte('\n')
		}
		if m.Input != "" {
			b.WriteString("input: ")
			b.WriteString(m.Input)
			b.WriteByte('\n')
		}
	}
	if m.PendingAction == "approval-confirm" {
		b.WriteString("\nRevision hash: ")
		b.WriteString(m.RevisionHash)
		b.WriteString("\nApproved content hash: ")
		b.WriteString(m.ApprovedContentHash)
		b.WriteString("\nChallenge: ")
		b.WriteString(m.Challenge)
		b.WriteString("\nResponse: ")
		b.WriteString(m.Input)
		b.WriteByte('\n')
	}
	if m.Status != "" {
		b.WriteString("\n")
		b.WriteString(m.Status)
		b.WriteByte('\n')
	}
	if m.ResumeNotice != "" {
		b.WriteString("\n")
		b.WriteString(m.ResumeNotice)
		b.WriteByte('\n')
	}
	if m.SnapshotPath != "" {
		b.WriteString("\nApproved snapshot: ")
		b.WriteString(m.SnapshotPath)
		b.WriteByte('\n')
	}
	if blockers := m.ResumeBlockers(); len(blockers) > 0 {
		b.WriteString("\nblockers:\n")
		for _, blocker := range blockers {
			b.WriteString("- ")
			b.WriteString(blocker)
			b.WriteByte('\n')
		}
	}
	b.WriteString("\n↑/↓ j/k navigate  a accept  e edit  c comment  x reject  f complete  p approve  q quit\n")
	return tea.NewView(b.String())
}
