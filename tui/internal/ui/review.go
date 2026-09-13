package ui

import tea "charm.land/bubbletea/v2"

type Item struct {
	ID, Kind, Statement, Status string
}

type Model struct {
	IntentID, Revision, Lifecycle, Actor string
	Items                                []Item
	Selected                             int
	Width, Height                        int
	Findings                             []string
	Quitting                             bool
	Modal                                string
	PendingAction                        string
	Status                               string
}

func New(items []Item) Model { return Model{Items: items, Lifecycle: "draft"} }

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.Quitting = true
			return m, tea.Quit
		case "j", "down":
			if m.Selected+1 < len(m.Items) {
				m.Selected++
			}
		case "k", "up":
			if m.Selected > 0 {
				m.Selected--
			}
		case "a", "e", "c", "x":
			if len(m.Items) > 0 {
				m.PendingAction = map[string]string{"a": "accept", "e": "edit-preview", "c": "comment", "x": "reject"}[msg.String()]
				m.Modal = m.PendingAction
			}
		case "enter":
			if m.Modal != "" {
				m.Status = m.PendingAction + " confirmed for " + m.Items[m.Selected].ID
				m.Modal, m.PendingAction = "", ""
			}
		case "esc":
			m.Modal, m.PendingAction = "", ""
		}
	case tea.WindowSizeMsg:
		m.Width, m.Height = msg.Width, msg.Height
	}
	return m, nil
}

func (m Model) View() tea.View {
	if m.Quitting {
		return tea.NewView("Review closed.\n")
	}
	s := "zintent  " + m.Lifecycle + "  rev:" + m.Revision + "  actor:" + m.Actor + "\n\n"
	for i, item := range m.Items {
		cursor := "  "
		if i == m.Selected {
			cursor = "> "
		}
		s += cursor + item.ID + " [" + item.Status + "] " + item.Statement + "\n"
	}
	if m.Modal != "" {
		s += "\nConfirm " + m.Modal + " for " + m.Items[m.Selected].ID + "? Enter=confirm Esc=cancel\n"
	}
	if m.Status != "" {
		s += "\n" + m.Status + "\n"
	}
	s += "\n↑/↓ j/k navigate  a accept  e edit  c comment  x reject  q quit\n"
	return tea.NewView(s)
}
