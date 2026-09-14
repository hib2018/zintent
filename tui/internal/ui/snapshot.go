package ui

type SnapshotScreen struct {
	SnapshotID, ApprovalID, RevisionID, ActorID, ApprovedContentHash string
	Verified                                                         bool
	Findings                                                         []string
}

func (s SnapshotScreen) SafeToDisplay() bool {
	return s.Verified && s.SnapshotID != "" && s.ApprovalID != ""
}
