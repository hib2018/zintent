package ui

import (
	"os"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestWorkspaceRootFrameGoldenScenarios(t *testing.T) {
	cases := []struct {
		name  string
		model WorkspaceModel
	}{
		{"wide", func() WorkspaceModel {
			m := NewWorkspace()
			m.Width = 120
			m.IntentID = "intent-1"
			m.Revision = "rev-1"
			return m
		}()},
		{"narrow", func() WorkspaceModel { m := NewWorkspace(); m.Width = 20; return m }()},
		{"loading", func() WorkspaceModel { m := NewWorkspace(); m.Status = "Loading canonical Intent…"; return m }()},
		{"empty", NewWorkspace()},
		{"long-text", func() WorkspaceModel { m := NewWorkspace(); m.Status = strings.Repeat("long status ", 20); return m }()},
		{"error", func() WorkspaceModel { m := NewWorkspace(); m.Status = "error: core process unavailable"; return m }()},
		{"blocker-heavy", func() WorkspaceModel {
			m := NewWorkspace()
			m.Status = "17 blocking findings; open Validation for details"
			return m
		}()},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			golden, err := os.ReadFile("testdata/" + tc.name + ".golden")
			if err != nil {
				t.Fatal(err)
			}
			view := tc.model.View()
			for _, required := range strings.Split(strings.TrimSpace(string(golden)), "\n") {
				if !strings.Contains(view.Content, required) {
					t.Fatalf("missing golden fragment %q in:\n%s", required, view.Content)
				}
			}
			if !view.AltScreen {
				t.Fatal("root frame must retain alternate screen")
			}
		})
	}
}

func TestWorkspaceResizeDoesNotLoseStableSelection(t *testing.T) {
	m := NewWorkspace().ReloadRecords([]string{"i-1", "i-2"})
	m.SelectedID = "i-2"
	next, _ := m.Update(tea.WindowSizeMsg{Width: 38, Height: 10})
	if next.(WorkspaceModel).SelectedID != "i-2" {
		t.Fatal("resize lost stable selection")
	}
}
