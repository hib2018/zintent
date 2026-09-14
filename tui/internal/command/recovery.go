package command

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/hib2018/zintent/tui/internal/protocol"
)

type RecoveryCandidate struct {
	CandidateID  string `json:"candidate_id"`
	RelativePath string `json:"relative_path"`
	Kind         string `json:"kind"`
	Size         int64  `json:"size"`
	ContentHash  string `json:"content_hash"`
}
type RecoveryStatus struct {
	ObservedRevisionID  string              `json:"observed_revision_id"`
	ObservedHeadHash    string              `json:"observed_head_hash"`
	TemporaryCandidates []RecoveryCandidate `json:"temporary_candidates"`
	RecoveryToken       string              `json:"recovery_token"`
	ExpiresInSeconds    int                 `json:"expires_in_seconds"`
}

func FormatRecoveryHuman(status RecoveryStatus) string {
	return fmt.Sprintf("revision %s: %d temporary candidate(s)", status.ObservedRevisionID, len(status.TemporaryCandidates))
}

func RecoveryStatusRequest(requestID, intentPath string) (protocol.Request, error) {
	return AuditRequest("recovery_status", requestID, intentPath, "")
}
func CleanupRequest(requestID, intentPath, revision, headHash, operationID, token string, candidates []string, actor protocol.Actor) (protocol.Request, error) {
	if token == "" || len(candidates) == 0 {
		return protocol.Request{}, errors.New("fresh recovery token and explicit candidates required")
	}
	payload := map[string]any{"operation": "cleanup_temporary_files", "intent_path": intentPath, "expected_revision_id": revision, "expected_head_hash": headHash, "operation_id": operationID, "recovery_token": token, "candidate_ids": candidates, "actor": actor}
	body, err := json.Marshal(payload)
	if err != nil {
		return protocol.Request{}, err
	}
	return protocol.Request{ProtocolVersion: protocol.Version, RequestID: requestID, Operation: "cleanup_temporary_files", PayloadSchema: "zintent.command/1", Payload: body}, nil
}
