package ui

import (
	"fmt"
	"strings"
)

type Dashboard struct {
	IntentID, Lifecycle, RevisionID string
	BlockerCount                    int
	SnapshotID                      string
}

func (d Dashboard) Routes() []Screen {
	routes := []Screen{ScreenReview, ScreenComments, ScreenCompletion, ScreenHistory, ScreenValidation}
	if d.Lifecycle == "review_complete" {
		routes = append(routes, ScreenApproval)
	}
	if d.SnapshotID != "" {
		routes = append(routes, ScreenSnapshot)
	}
	return append(routes, ScreenRecovery)
}

func (d Dashboard) View() string {
	var b strings.Builder
	b.WriteString("DASHBOARD\n")
	fmt.Fprintf(&b, "  Intent    : %s\n  Lifecycle : %s\n  Revision  : %s\n  Blockers  : %d\n", shortRef(d.IntentID), fallback(d.Lifecycle, "-"), shortRef(d.RevisionID), d.BlockerCount)
	if d.SnapshotID != "" {
		fmt.Fprintf(&b, "  Snapshot  : %s\n", shortRef(d.SnapshotID))
	}
	b.WriteString("\nWORKFLOWS\n")
	labels := map[Screen]string{ScreenReview: "r Review", ScreenComments: "c Comments", ScreenCompletion: "f Completion", ScreenApproval: "p Approval", ScreenHistory: "h History", ScreenValidation: "v Validation", ScreenSnapshot: "s Snapshot", ScreenRecovery: "R Recovery"}
	for _, route := range d.Routes() {
		fmt.Fprintf(&b, "  %s\n", labels[route])
	}
	return b.String()
}
