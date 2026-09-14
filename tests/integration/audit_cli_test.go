package integration

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestAuditOperationsReturnMachineReadableResults(t *testing.T) {
	root, _ := filepath.Abs(filepath.Join("..", ".."))
	core := filepath.Join(root, "zig-out", "bin", "zintent-core")
	for _, operation := range []string{"show_intent", "validate_intent", "diff_revisions"} {
		body, _ := json.Marshal(map[string]any{
			"protocol_version": "1.0", "request_id": "audit-test", "operation": operation,
			"payload_schema": "zintent.command/1", "payload": map[string]any{"operation": operation, "intent_path": filepath.Join(root, "tests", "fixtures", "valid-draft.json")},
		})
		cmd := exec.Command(core)
		cmd.Stdin = bytes.NewReader(body)
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("%s failed: %v", operation, err)
		}
		var envelope map[string]any
		if err := json.Unmarshal(out, &envelope); err != nil {
			t.Fatalf("%s returned invalid JSON: %v", operation, err)
		}
		if envelope["request_id"] != "audit-test" {
			t.Fatalf("%s did not preserve request id: %#v", operation, envelope)
		}
	}
}
