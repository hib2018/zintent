package ui

import (
	"fmt"
	"strings"
	"time"

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
	RecordIDs                  []string
	ActiveRequestID            string
	Quitting                   bool
	Lifecycle                  string
	Review                     ReviewScreen
	Comments                   CommentsScreen
	Completion                 CompletionScreen
	Approval                   ApprovalModel
	Snapshot                   SnapshotScreen
	History                    HistoryScreen
	Validation                 ValidationScreen
	Recovery                   RecoveryScreen
	IntentList                 IntentListScreen
	Import                     ImportModal
	WorkspacePath              string
	WorkspaceExecutor          interface {
		List() tea.Cmd
		InspectDraft(string) tea.Cmd
		ImportDraft(ImportModal) tea.Cmd
		OpenIntent(string) tea.Cmd
	}
}

type WorkspaceCanonicalMsg struct {
	IntentID, RevisionID, Lifecycle, SelectedID string
	Items                                       []Item
	Comments                                    []CommentRecord
	Err                                         error
}
type SnapshotIssuedMsg struct{ Snapshot SnapshotScreen }

func NewWorkspace() WorkspaceModel {
	return WorkspaceModel{nav: navigation{stack: []Screen{ScreenIntentList}}}
}
func (m WorkspaceModel) Screen() Screen { return m.nav.current() }
func (m WorkspaceModel) Init() tea.Cmd {
	if m.WorkspaceExecutor != nil {
		return m.WorkspaceExecutor.List()
	}
	return nil
}

func (m WorkspaceModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyReleaseMsg:
		return m, nil
	case tea.KeyPressMsg:
		key := msg.String()
		if m.IntentList.FilterEditing {
			switch key {
			case "esc", "enter":
				m.IntentList.FilterEditing = false
			case "backspace":
				if len(m.IntentList.Filter) > 0 {
					m.IntentList.Filter = m.IntentList.Filter[:len(m.IntentList.Filter)-1]
				}
			default:
				if msg.Text != "" {
					m.IntentList.Filter += msg.Text
					m.IntentList = m.IntentList.Reload(m.IntentList.Entries)
				}
			}
			return m, nil
		}
		if key == "ctrl+c" || key == "q" && m.Modal == ModalClosed {
			m.Quitting = true
			return m, tea.Quit
		}
		if m.Import.Phase == ModalEditing {
			switch key {
			case "esc":
				m.Import.Clear()
				m.Modal = ModalClosed
			case "backspace":
				if len(m.Import.SourcePath) > 0 {
					m.Import.SourcePath = m.Import.SourcePath[:len(m.Import.SourcePath)-1]
				}
			case "enter":
				if m.Import.SourcePath != "" && m.WorkspaceExecutor != nil {
					m.Import.Phase = ModalPreviewLoading
					return m, m.WorkspaceExecutor.InspectDraft(m.Import.SourcePath)
				}
			default:
				if msg.Text != "" {
					m.Import.SourcePath += msg.Text
				}
			}
			return m, nil
		}
		if m.Import.Phase == ModalConfirming {
			if key == "esc" {
				m.Import.Clear()
				m.Modal = ModalClosed
				return m, nil
			}
			if key == "enter" && m.Import.CanSubmit(time.Now()) && m.WorkspaceExecutor != nil {
				m.Import.Phase = ModalSubmitting
				return m, m.WorkspaceExecutor.ImportDraft(m.Import)
			}
			return m, nil
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
		case "/":
			if m.Screen() == ScreenIntentList {
				m.IntentList.FilterEditing = true
			}
		case "n":
			if m.Screen() == ScreenIntentList {
				m.Import = ImportModal{Phase: ModalEditing}
				m.Modal = ModalEditing
			}
		case "enter":
			if m.Screen() == ScreenIntentList {
				if selected := m.IntentList.Selected(); selected != nil && !selected.Corrupt {
					m.IntentID = selected.ID
					m.Revision = selected.Revision
					m.Lifecycle = selected.Lifecycle
					m.SelectedID = selected.ID
					if m.WorkspaceExecutor != nil {
						m.Status = "loading canonical Intent"
						return m, m.WorkspaceExecutor.OpenIntent(selected.Path)
					}
					m.nav.push(ScreenDashboard)
				} else if m.WorkspaceExecutor == nil {
					m.nav.push(ScreenDashboard)
				}
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
	case WorkspaceCanonicalMsg:
		if msg.Err != nil {
			m.Status = "canonical reload failed: " + msg.Err.Error()
			m.Modal = ModalError
			break
		}
		m.IntentID, m.Revision, m.Lifecycle = msg.IntentID, msg.RevisionID, msg.Lifecycle
		m.Review = m.Review.Reload(msg.Items)
		if msg.SelectedID != "" {
			m.Review.SelectedID = msg.SelectedID
		}
		m.Comments = m.Comments.Reload(msg.Comments)
		m.Completion.RevisionID = m.Revision
		if m.Screen() == ScreenIntentList {
			m.nav.push(ScreenDashboard)
		}
		m.ActiveRequestID = ""
		m.Modal = ModalClosed
		m.Status = "canonical Intent reloaded"
	case SnapshotIssuedMsg:
		m.Snapshot = msg.Snapshot
		if msg.Snapshot.SafeToDisplay() {
			m.nav.push(ScreenSnapshot)
			m.Status = "approved snapshot verified"
		}
	case WorkspaceListMsg:
		if msg.Err != nil {
			m.Status = "workspace reload failed: " + msg.Err.Error()
		} else {
			m.IntentList = m.IntentList.Reload(msg.Entries)
			m.Status = "workspace reloaded"
		}
	case DraftPreviewMsg:
		if msg.Err != nil {
			m.Import.ResetFailure(msg.Err.Error())
			m.Modal = ModalError
		} else {
			m.Import.SourceHash = msg.SourceHash
			m.Import.IntentID = msg.IntentID
			m.Import.Destination = msg.Destination
			m.Import.Token = msg.Token
			m.Import.Findings = msg.Findings
			m.Import.ExpiresAt = time.Now().Add(10 * time.Minute)
			m.Import.Phase = ModalConfirming
			m.Modal = ModalConfirming
		}
	case DraftImportedMsg:
		if msg.Err != nil {
			m.Import.ResetFailure(msg.Err.Error())
			m.Modal = ModalError
		} else {
			m.SelectedID = msg.IntentID
			m.IntentList.SelectedID = msg.IntentID
			m.Import.Clear()
			m.Modal = ModalClosed
			m.Status = "Draft imported"
			if m.WorkspaceExecutor != nil {
				return m, m.WorkspaceExecutor.List()
			}
		}
	}
	return m, nil
}

// ReloadRecords preserves selection by stable ID. If it disappeared, the
// first remaining record becomes the deterministic fallback.
func (m WorkspaceModel) ReloadRecords(ids []string) WorkspaceModel {
	selected := m.SelectedID
	m.RecordIDs = append(m.RecordIDs[:0], ids...)
	m.SelectedID = ""
	for _, id := range m.RecordIDs {
		if id == selected {
			m.SelectedID = id
			return m
		}
	}
	if len(m.RecordIDs) > 0 {
		m.SelectedID = m.RecordIDs[0]
	}
	return m
}

// BeginConfirm suppresses repeated confirmation while the same request is in
// flight. The operation itself remains delegated to the command executor.
func (m WorkspaceModel) BeginConfirm(requestID string) (WorkspaceModel, bool) {
	if requestID == "" || m.ActiveRequestID != "" || m.Modal == ModalSubmitting || m.Modal == ModalReloadLoading {
		return m, false
	}
	m.ActiveRequestID, m.Modal = requestID, ModalSubmitting
	return m, true
}

func (m WorkspaceModel) View() tea.View {
	if m.Quitting {
		return tea.NewView("Workspace closed.\n")
	}
	width := m.Width
	if width < 40 {
		width = 40
	}
	height := m.Height
	if height < 18 {
		height = 24
	}
	header := fmt.Sprintf("zintent workspace  %s  intent:%s  rev:%s", m.Screen(), fallback(m.IntentID, "-"), fallback(m.Revision, "-"))
	navigation := m.navigationBody()
	main := m.screenBody()
	bodyHeight := max(8, height-9)
	var body string
	if width < 72 {
		body = strings.Join(renderPane("Navigation", navigation, width, 7, false), "\n") + "\n" +
			strings.Join(renderPane(m.Screen().String(), main, width, bodyHeight, true), "\n")
	} else {
		left := max(24, width/4)
		right := width - left - 1
		body = joinPanes(renderPane("Navigation", navigation, left, bodyHeight, false), renderPane(m.Screen().String(), main, right, bodyHeight, true))
	}
	status := fallback(m.Status, "Ready")
	var b strings.Builder
	b.WriteString(strings.Join(renderPane("Workspace", header, width, 3, false), "\n"))
	b.WriteByte('\n')
	b.WriteString(body)
	if m.Import.Phase != ModalClosed {
		var modal strings.Builder
		fmt.Fprintf(&modal, "Source: %s\nHash: %s\nDestination: %s\n", m.Import.SourcePath, m.Import.SourceHash, m.Import.Destination)
		for _, finding := range m.Import.Findings {
			fmt.Fprintf(&modal, "BLOCKING: %s\n", finding)
		}
		b.WriteByte('\n')
		b.WriteString(strings.Join(renderPane("Import Draft", modal.String(), width, 7, true), "\n"))
	}
	b.WriteByte('\n')
	b.WriteString(strings.Join(renderPane("Status", status, width, 3, false), "\n"))
	b.WriteString("\nenter open  esc back  r review  c comments  f complete  p approve  h history  v validate  q quit\n")
	v := tea.NewView(b.String())
	v.AltScreen = true
	if m.Modal == ModalEditing {
		v.Cursor = tea.NewCursor(0, 0)
	}
	return v
}

func (m WorkspaceModel) navigationBody() string {
	labels := []struct {
		screen Screen
		key    string
	}{
		{ScreenIntentList, "•"}, {ScreenDashboard, "d"}, {ScreenReview, "r"},
		{ScreenComments, "c"}, {ScreenCompletion, "f"}, {ScreenApproval, "p"},
		{ScreenHistory, "h"}, {ScreenValidation, "v"}, {ScreenSnapshot, "s"}, {ScreenRecovery, "R"},
	}
	var lines []string
	for _, item := range labels {
		marker := "  "
		if item.screen == m.Screen() {
			marker = "→ "
		}
		lines = append(lines, fmt.Sprintf("%s%s  %s", marker, item.key, item.screen))
	}
	return strings.Join(lines, "\n")
}

func (m WorkspaceModel) screenBody() string {
	switch m.Screen() {
	case ScreenHistory, ScreenDiff:
		return m.History.View()
	case ScreenValidation:
		return m.Validation.View()
	case ScreenSnapshot:
		return m.Snapshot.View()
	case ScreenRecovery:
		return m.Recovery.View()
	case ScreenIntentList:
		return m.IntentList.View() + "\nn new Draft  / search"
	default:
		return workspaceBody(m.Screen())
	}
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
