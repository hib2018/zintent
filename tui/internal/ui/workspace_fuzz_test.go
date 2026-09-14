package ui

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
)

func FuzzWorkspaceArbitraryKeys(f *testing.F) {
	f.Add("n/path/to/draft\n\x1bq")
	f.Add("/intent\n\x1bq")
	f.Fuzz(func(t *testing.T, keys string) {
		m := NewWorkspace()
		for _, r := range []rune(keys) {
			next, _ := m.Update(tea.KeyPressMsg(tea.Key{Text: string(r), Code: r}))
			m = next.(WorkspaceModel)
			if m.Modal > ModalError {
				t.Fatalf("invalid modal %d", m.Modal)
			}
		}
		_ = m.View()
	})
}

func FuzzWorkspaceInvalidResponses(f *testing.F) {
	f.Add("bad response", true)
	f.Add("", false)
	f.Fuzz(func(t *testing.T, message string, preview bool) {
		m := NewWorkspace()
		if preview {
			next, _ := m.Update(DraftPreviewMsg{Err: fuzzError(message)})
			m = next.(WorkspaceModel)
		} else {
			next, _ := m.Update(WorkspaceListMsg{Entries: []IntentEntry{{ID: message}}})
			m = next.(WorkspaceModel)
		}
		_, _ = m.Update(tea.KeyReleaseMsg(tea.Key{Text: "q", Code: 'q'}))
		_ = m.Import.CanSubmit(time.Now())
	})
}

type fuzzError string

func (e fuzzError) Error() string { return string(e) }
