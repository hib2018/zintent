package ui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
)

func renderView(m Model) tea.View {
	if m.Quitting {
		return tea.NewView("Review closed.\n")
	}
	var list, detail strings.Builder

	start, end := 0, len(m.Items)
	if m.Height > 8 && len(m.Items) > m.Height-8 {
		visible := m.Height - 8
		start = max(0, min(m.Selected-visible/2, len(m.Items)-visible))
		end = start + visible
	}
	for i := start; i < end; i++ {
		item := m.Items[i]
		if i == m.Selected {
			list.WriteString("→ ")
		} else {
			list.WriteString("  ")
		}
		list.WriteString(item.ID + " [" + item.Status + "] " + item.Statement + "\n")
	}

	if len(m.Items) > 0 {
		selected := m.Items[m.Selected]
		detail.WriteString(selected.ID + " / " + selected.Kind + "\n\n" + selected.Statement + "\n")
		if selected.Provenance != "" {
			detail.WriteString("provenance: " + selected.Provenance + "\n")
		}
		if selected.Rationale != "" {
			detail.WriteString("rationale: " + selected.Rationale + "\n")
		}
	}
	width := max(40, m.Width)
	height := max(12, m.Height-7)
	var b strings.Builder
	header := "zintent  " + m.Lifecycle + "  rev:" + m.Revision + "  actor:" + m.Actor
	b.WriteString(strings.Join(renderPane("Review", header, width, 3, false), "\n"))
	b.WriteByte('\n')
	if width < 72 {
		b.WriteString(strings.Join(renderPane("Items", list.String(), width, max(6, height/2), true), "\n"))
		b.WriteByte('\n')
		b.WriteString(strings.Join(renderPane("Item Detail", detail.String(), width, max(6, height/2), false), "\n"))
	} else {
		left := max(28, width*2/5)
		right := width - left - 1
		b.WriteString(joinPanes(renderPane("Items", list.String(), left, height, true), renderPane("Item Detail", detail.String(), right, height, false)))
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
