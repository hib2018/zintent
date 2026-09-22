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
		line := reviewItemText("  ", item)
		if i == m.Selected {
			line = highlightTopLine(reviewItemText("→ ", item))
		}
		list.WriteString(line)
	}

	if len(m.Items) > 0 {
		selected := m.Items[m.Selected]
		detail.WriteString("ITEM\n")
		detail.WriteString("  ID         : " + shortRef(selected.ID) + "\n")
		detail.WriteString("  Kind       : " + selected.Kind + "\n")
		detail.WriteString("  Status     : " + fallback(selected.Status, "unreviewed") + "\n\n")
		detail.WriteString("STATEMENT\n  " + selected.Statement + "\n")
		if selected.Provenance != "" {
			detail.WriteString("\nPROVENANCE\n  " + selected.Provenance + "\n")
		}
		if selected.Rationale != "" {
			detail.WriteString("\nRATIONALE\n  " + selected.Rationale + "\n")
		}
	}
	width := max(40, m.Width)
	height := max(12, m.Height-7)
	var b strings.Builder
	header := "REVIEW  lifecycle:" + m.Lifecycle + "  revision:" + shortRef(m.Revision) + "  actor:" + fallback(m.Actor, "-")
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
	inputLabel := ""
	if m.Modal != "" && len(m.Items) > 0 {
		b.WriteString("\nConfirm ")
		b.WriteString(m.Modal)
		b.WriteString(" for ")
		b.WriteString(shortRef(m.Items[m.Selected].ID))
		b.WriteString("? Enter=confirm Esc=cancel\n")
		if m.PendingAction == "edit-confirm" {
			b.WriteString("before: ")
			b.WriteString(m.Before)
			b.WriteString("\nafter: ")
			b.WriteString(m.After)
			b.WriteByte('\n')
		}
		switch m.PendingAction {
		case "edit-preview":
			inputLabel = "New statement: "
		case "comment":
			inputLabel = "Comment: "
		case "reject":
			inputLabel = "Rejection reason: "
		}
		if inputLabel != "" {
			b.WriteString(inputLabel)
			b.WriteString(m.Input)
			b.WriteByte('\n')
		}
	}
	if m.PendingAction == "approval-confirm" {
		inputLabel = "Response: "
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
	view := tea.NewView(b.String())
	if inputLabel != "" {
		view.Cursor = cursorAfterLabel(view.Content, inputLabel, m.Input)
	}
	return view
}

func reviewItemText(marker string, item Item) string {
	return marker + "[" + fallback(item.Status, "unreviewed") + "] " + item.Kind + "  " + item.Statement + "\n\n"
}

func shortRef(value string) string {
	const head, tail = 10, 6
	if len(value) <= head+tail+1 {
		return value
	}
	return value[:head] + "…" + value[len(value)-tail:]
}
