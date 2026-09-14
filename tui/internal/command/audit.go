// Package command contains presentation helpers for read-only audit commands.
package command

import (
	"encoding/json"
	"fmt"

	"github.com/hib2018/zintent/tui/internal/protocol"
)

// AuditResult is the stable subset consumed by human and JSON frontends.
type AuditResult struct {
	Operation string          `json:"operation"`
	Data      json.RawMessage `json:"data"`
	Findings  []string        `json:"findings"`
}

func InspectSnapshotRequest(requestID, intentPath, snapshotID string) (protocol.Request, error) {
	body, err := json.Marshal(map[string]any{"operation": "inspect_snapshot", "intent_path": intentPath, "snapshot_id": snapshotID})
	if err != nil {
		return protocol.Request{}, err
	}
	return protocol.Request{ProtocolVersion: protocol.Version, RequestID: requestID, Operation: "inspect_snapshot", PayloadSchema: "zintent.command/1", Payload: body}, nil
}

// FormatHuman keeps audit output useful without hiding the machine-readable
// response returned by the core process.
func FormatHuman(result AuditResult) string {
	if len(result.Findings) == 0 {
		return fmt.Sprintf("%s succeeded", result.Operation)
	}
	return fmt.Sprintf("%s succeeded (%d findings)", result.Operation, len(result.Findings))
}
