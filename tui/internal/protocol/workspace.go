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
	Code       string `json:"code"`
	Severity   string `json:"severity"`
	RecordType string `json:"record_type,omitempty"`
	RecordID   string `json:"record_id,omitempty"`
	Path       string `json:"path,omitempty"`
	Message    string `json:"message"`
}

type IntentView struct {
	IntentID       string           `json:"intent_id"`
	RevisionID     string           `json:"revision_id"`
	LifecycleState string           `json:"lifecycle_state"`
	Items          []IntentItemView `json:"items"`
	Comments       []CommentView    `json:"comments"`
}

type IntentItemView struct {
	ID           string `json:"item_id"`
	Kind         string `json:"kind"`
	Statement    string `json:"statement"`
	ReviewStatus string `json:"review_status"`
}
type CommentView struct {
	ID           string `json:"comment_id"`
	TargetItemID string `json:"target_item_id"`
	Body         string `json:"body"`
	Status       string `json:"status"`
}

type CorrelatedResult struct {
	RequestID string          `json:"request_id"`
	Operation string          `json:"operation"`
	Result    json.RawMessage `json:"result,omitempty"`
	Err       error           `json:"-"`
}
