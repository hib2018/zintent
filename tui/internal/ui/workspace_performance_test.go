package ui

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestIntentListThousandEntriesRendersBoundedRowsUnder100ms(t *testing.T) {
	entries := make([]IntentEntry, 1000)
	for i := range entries {
		entries[i] = IntentEntry{ID: fmt.Sprintf("intent-%04d", i), Lifecycle: "in_review"}
	}
	s := IntentListScreen{Height: 24}.Reload(entries)
	started := time.Now()
	view := s.View()
	if elapsed := time.Since(started); elapsed >= 100*time.Millisecond {
		t.Fatalf("render took %s", elapsed)
	}
	if rows := strings.Count(view, "intent-"); rows != 24 {
		t.Fatalf("rendered %d rows, want 24", rows)
	}
}

func TestReviewThousandItemsUsesVisibleWindow(t *testing.T) {
	items := make([]Item, 1000)
	for i := range items {
		items[i] = Item{ID: fmt.Sprintf("I-%04d", i), Status: "unreviewed"}
	}
	s := ReviewScreen{Height: 24}.Reload(items)
	started := time.Now()
	if got := len(s.Visible()); got != 24 {
		t.Fatalf("visible=%d", got)
	}
	if elapsed := time.Since(started); elapsed >= 100*time.Millisecond {
		t.Fatalf("reducer took %s", elapsed)
	}
}
