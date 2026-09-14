package ui

import "fmt"

type SnapshotScreen struct {
	SnapshotID, ApprovalID, RevisionID, ActorID, ApprovedContentHash string
	Verified                                                         bool
	Findings                                                         []string
}

func (s SnapshotScreen) View() string {
	return fmt.Sprintf("Verified snapshot\nID: %s\nApproval: %s\nRevision: %s\nActor: %s\nApproved content hash: %s\nVerified: %t\n", s.SnapshotID, s.ApprovalID, s.RevisionID, s.ActorID, s.ApprovedContentHash, s.Verified)
}

func (s SnapshotScreen) SafeToDisplay() bool {
	return s.Verified && s.SnapshotID != "" && s.ApprovalID != ""
}
