package ui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
)

type ModalPhase uint8

const (
	ModalClosed ModalPhase = iota
	ModalEditing
	ModalPreviewLoading
	ModalConfirming
	ModalSubmitting
	ModalReloadLoading
	ModalError
)

type WorkspaceModel struct {
	nav                        navigation
	Modal                      ModalPhase
	Width, Height              int
	IntentID, Revision, Status string
	SelectedID                 string
	Quitting                   bool
}

func NewWorkspace() WorkspaceModel {
	return WorkspaceModel{nav: navigation{stack: []Screen{ScreenIntentList}}}
}
func (m WorkspaceModel) Screen() Screen { return m.nav.current() }
func (m WorkspaceModel) Init() tea.Cmd  { return nil }

func (m WorkspaceModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyReleaseMsg:
		return m, nil
	case tea.KeyPressMsg:
		key := msg.String()
		if key == "ctrl+c" || key == "q" && m.Modal == ModalClosed {
			m.Quitting = true
			return m, tea.Quit
		}
		if key == "esc" {
			if m.Modal != ModalClosed {
				m.Modal = ModalClosed
			} else {
				m.nav.back()
			}
			return m, nil
		}
		if m.Modal != ModalClosed {
			return m, nil
		}
		switch key {
		case "enter":
			if m.Screen() == ScreenIntentList {
				m.nav.push(ScreenDashboard)
			}
		case "r":
			m.nav.push(ScreenReview)
		case "c":
			m.nav.push(ScreenComments)
		case "f":
			m.nav.push(ScreenCompletion)
		case "p":
			m.nav.push(ScreenApproval)
		case "h":
			m.nav.push(ScreenHistory)
		case "v":
			m.nav.push(ScreenValidation)
		case "s":
			m.nav.push(ScreenSnapshot)
		case "R":
			m.nav.push(ScreenRecovery)
		}
	case tea.WindowSizeMsg:
		m.Width, m.Height = msg.Width, msg.Height
	}
	return m, nil
}

func (m WorkspaceModel) View() tea.View {
	if m.Quitting {
		return tea.NewView("Workspace closed.\n")
	}
	width := m.Width
	if width < 40 {
		width = 40
	}
	var b strings.Builder
	fmt.Fprintf(&b, "zintent workspace  %s  intent:%s  rev:%s\n", m.Screen(), fallback(m.IntentID, "-"), fallback(m.Revision, "-"))
	b.WriteString(strings.Repeat("─", min(width, 100)))
	b.WriteByte('\n')
	fmt.Fprintf(&b, "\n%s\n", workspaceBody(m.Screen()))
	if m.Status != "" {
		fmt.Fprintf(&b, "\n%s\n", m.Status)
	}
	b.WriteString("\nenter open  esc back  r review  c comments  f complete  p approve  h history  v validate  q quit\n")
	v := tea.NewView(b.String())
	v.AltScreen = true
	return v
}

func fallback(value, otherwise string) string {
	if value == "" {
		return otherwise
	}
	return value
}
func workspaceBody(screen Screen) string {
	return map[Screen]string{
		ScreenIntentList: "Intent list\nNo Intent selected.", ScreenDashboard: "Dashboard\nChoose a workflow action.",
		ScreenReview: "Review\nSelect an item to inspect and review.", ScreenComments: "Comments\nOpen and closed review comments.",
		ScreenCompletion: "Completion\nReview blockers before completion.", ScreenApproval: "Approval\nExact revision and challenge confirmation.",
		ScreenHistory: "History\nVerified reachable revisions.", ScreenDiff: "Diff\nItem-aware revision comparison.",
		ScreenValidation: "Validation\nCore findings.", ScreenSnapshot: "Snapshot\nVerified approved snapshot.",
		ScreenRecovery: "Recovery\nObserved temporary candidates.",
	}[screen]
}
