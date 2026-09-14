package contract

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type workspaceCommand struct {
	Operation          string          `json:"operation"`
	WorkspacePath      string          `json:"workspace_path,omitempty"`
	IntentPath         string          `json:"intent_path,omitempty"`
	SourcePath         string          `json:"source_path,omitempty"`
	Destination        string          `json:"destination,omitempty"`
	ImportToken        string          `json:"import_token,omitempty"`
	OperationID        string          `json:"operation_id,omitempty"`
	RevisionID         string          `json:"revision_id,omitempty"`
	SnapshotID         string          `json:"snapshot_id,omitempty"`
	ExpectedRevisionID string          `json:"expected_revision_id,omitempty"`
	ExpectedHeadHash   string          `json:"expected_head_hash,omitempty"`
	RecoveryToken      string          `json:"recovery_token,omitempty"`
	IncludeOrphan      bool            `json:"include_orphan,omitempty"`
	CandidateIDs       []string        `json:"candidate_ids,omitempty"`
	Actor              json.RawMessage `json:"actor,omitempty"`
}

func decodeWorkspaceStrict(data []byte) (workspaceCommand, error) {
	var command workspaceCommand
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	err := decoder.Decode(&command)
	return command, err
}

func validWorkspaceCommand(c workspaceCommand) bool {
	switch c.Operation {
	case "list_intents":
		return c.WorkspacePath != ""
	case "inspect_draft":
		return c.WorkspacePath != "" && c.SourcePath != "" && len(c.Actor) > 0
	case "import_draft":
		return c.WorkspacePath != "" && c.SourcePath != "" && c.Destination != "" && c.ImportToken != "" && c.OperationID != "" && len(c.Actor) > 0
	case "list_revisions", "recovery_status":
		return c.IntentPath != ""
	case "inspect_revision":
		return c.IntentPath != "" && c.RevisionID != ""
	case "inspect_snapshot":
		return c.IntentPath != "" && c.SnapshotID != ""
	case "cleanup_temporary_files":
		return c.IntentPath != "" && c.ExpectedRevisionID != "" && len(c.ExpectedHeadHash) == 64 && c.RecoveryToken != "" && len(c.CandidateIDs) > 0 && c.OperationID != "" && len(c.Actor) > 0
	}
	return false
}

func TestWorkspaceCommandFixturesMatchGoContract(t *testing.T) {
	entries, err := os.ReadDir("fixtures/workspace")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		t.Run(entry.Name(), func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join("fixtures/workspace", entry.Name()))
			if err != nil {
				t.Fatal(err)
			}
			command, decodeErr := decodeWorkspaceStrict(data)
			valid := decodeErr == nil && validWorkspaceCommand(command)
			if strings.Contains(entry.Name(), ".valid.") && !valid {
				t.Fatalf("valid fixture rejected: %v %+v", decodeErr, command)
			}
			if strings.Contains(entry.Name(), ".invalid.") && valid {
				t.Fatal("invalid fixture accepted")
			}
		})
	}
}

func TestWorkspaceResultEnvelopeCorrelation(t *testing.T) {
	var result struct {
		ProtocolVersion string          `json:"protocol_version"`
		RequestID       string          `json:"request_id"`
		OK              bool            `json:"ok"`
		ResultSchema    string          `json:"result_schema"`
		Result          json.RawMessage `json:"result"`
	}
	data := []byte(`{"protocol_version":"1.0","request_id":"r-1","ok":true,"result_schema":"zintent.result/1","result":{"operation":"list_intents","data":{"entries":[]}}}`)
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&result); err != nil {
		t.Fatal(err)
	}
	if result.ProtocolVersion != "1.0" || result.RequestID != "r-1" || !result.OK || len(result.Result) == 0 {
		t.Fatalf("bad result envelope: %+v", result)
	}
}
