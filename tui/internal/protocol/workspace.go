package protocol

import "encoding/json"

type Actor struct {
	Type           string `json:"actor_type"`
	ID             string `json:"actor_id"`
	IdentitySource string `json:"identity_source"`
	Authenticated  bool   `json:"authenticated"`
}

type WorkspaceCommand struct {
	Operation          string   `json:"operation"`
	WorkspacePath      string   `json:"workspace_path,omitempty"`
	IntentPath         string   `json:"intent_path,omitempty"`
	SourcePath         string   `json:"source_path,omitempty"`
	Destination        string   `json:"destination,omitempty"`
	ImportToken        string   `json:"import_token,omitempty"`
	RevisionID         string   `json:"revision_id,omitempty"`
	SnapshotID         string   `json:"snapshot_id,omitempty"`
	IncludeOrphan      bool     `json:"include_orphan,omitempty"`
	ExpectedRevisionID string   `json:"expected_revision_id,omitempty"`
	ExpectedHeadHash   string   `json:"expected_head_hash,omitempty"`
	RecoveryToken      string   `json:"recovery_token,omitempty"`
	CandidateIDs       []string `json:"candidate_ids,omitempty"`
	OperationID        string   `json:"operation_id,omitempty"`
	Actor              *Actor   `json:"actor,omitempty"`
}

type Finding struct {
	Code, Severity, RecordType, RecordID, Path, Message string
}

type IntentView struct {
	IntentID, RevisionID, LifecycleState string
	Items                                []IntentItemView
	Comments                             []CommentView
}

type IntentItemView struct{ ID, Kind, Statement, ReviewStatus string }
type CommentView struct{ ID, TargetItemID, Body, Status string }

type CorrelatedResult struct {
	RequestID string
	Operation string
	Result    json.RawMessage
	Err       error
}
