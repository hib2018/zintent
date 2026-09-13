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

	for i, item := range m.Items {
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
	}
	if m.Modal != "" && len(m.Items) > 0 {
		b.WriteString("\nConfirm ")
		b.WriteString(m.Modal)
		b.WriteString(" for ")
		b.WriteString(m.Items[m.Selected].ID)
		b.WriteString("? Enter=confirm Esc=cancel\n")
	}
	if m.Status != "" {
		b.WriteString("\n")
		b.WriteString(m.Status)
		b.WriteByte('\n')
	}
	b.WriteString("\n↑/↓ j/k navigate  a accept  e edit  c comment  x reject  q quit\n")
	return tea.NewView(b.String())
}
