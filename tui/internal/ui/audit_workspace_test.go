package ui

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestHistoryDiffAndProvenance(t *testing.T) {
	s := HistoryScreen{Revisions: []RevisionRecord{{ID: "r2", ParentID: "r1", ActorID: "alice", Reachable: true}}, Orphans: []RevisionRecord{{ID: "orphan"}}}.SelectPair("r1", "r2")
	if s.SelectedRevision("r2").ActorID != "alice" || s.SelectedRevision("orphan") == nil {
		t.Fatal()
	}
	change := ItemChange{ItemID: "i1", Before: "a", After: "b", Provenance: "human:alice"}
	if !strings.Contains(change.Provenance, "alice") {
		t.Fatal()
	}
}
func TestHistoryScrollsSelectedRevisionIntoView(t *testing.T) {
	s := HistoryScreen{Height: 7, Revisions: []RevisionRecord{{ID: "r1"}, {ID: "r2"}, {ID: "r3"}, {ID: "r4"}}}
	s.Move(3)
	view := s.View()
	if !strings.Contains(view, "[4/4]") || !strings.Contains(view, "r4") || strings.Contains(view, "r1") {
		t.Fatalf("selected revision is not visible: %s", view)
	}
}

func TestValidationFiltersAndJumps(t *testing.T) {
	s := ValidationScreen{Severity: "blocking", SelectedID: "i1", Findings: []ValidationFinding{{Code: "open", Severity: "blocking", RecordID: "i1", Message: "full message"}, {Code: "warn", Severity: "warning"}}}
	if len(s.Visible()) != 1 || s.JumpRecord() != "i1" {
		t.Fatal()
	}
}
func TestSnapshotLinkageAndRecoverySafety(t *testing.T) {
	snapshot := SnapshotScreen{SnapshotID: "s", ApprovalID: "a", RevisionID: "r", ActorID: "alice", ApprovedContentHash: "h", Verified: true}
	if !snapshot.SafeToDisplay() {
		t.Fatal()
	}
	recovery := RecoveryScreen{Token: "t", ExpiresAt: time.Now().Add(time.Minute), Temporary: []RecoveryCandidate{{ID: "tmp", Selected: true}, {ID: "protected", Selected: true, Protected: true}}, Orphans: []RecoveryCandidate{{ID: "orphan", Protected: true}}}
	ids := recovery.SelectedIDs()
	if len(ids) != 1 || ids[0] != "tmp" || !recovery.CanCleanup(time.Now()) {
		t.Fatal()
	}
	recovery.Invalidate("stale")
	if recovery.Token != "" {
		t.Fatal()
	}
}

func TestAuditScreenGoldens(t *testing.T) {
	screens := map[string]string{"history": (HistoryScreen{Revisions: []RevisionRecord{{ID: "r2", ParentID: "r1", OperationType: "reject_item", ActorID: "alice", CreatedAt: "now", Lifecycle: "in_review"}}}).View(), "validation": (ValidationScreen{Findings: []ValidationFinding{{Code: "open", Severity: "blocking", Message: "full"}}}).View(), "snapshot": (SnapshotScreen{SnapshotID: "s", ApprovalID: "a", Verified: true}).View(), "recovery": (RecoveryScreen{RevisionID: "r1", HeadHash: "hash"}).View()}
	for name, view := range screens {
		golden, err := os.ReadFile("testdata/audit-" + name + ".golden")
		if err != nil {
			t.Fatal(err)
		}
		for _, fragment := range strings.Split(strings.TrimSpace(string(golden)), "\n") {
			if !strings.Contains(view, fragment) {
				t.Fatalf("%s missing %q: %s", name, fragment, view)
			}
		}
	}
}
