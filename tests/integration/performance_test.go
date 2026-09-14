package integration

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLargeIntentValidationCompletesWithinBudget(t *testing.T) {
	root, _ := filepath.Abs(filepath.Join("..", ".."))
	core := filepath.Join(root, "zig-out", "bin", "zintent-core")
	items := make([]map[string]any, 1000)
	for i := range items {
		items[i] = map[string]any{"item_id": "item-" + string(rune('a'+i%26)) + string(rune('0'+i%10)), "kind": "goal", "statement": "generated intent statement", "review_status": "unreviewed"}
	}
	items[0]["statement"] = strings.Repeat("x", 10*1024*1024)
	artifact := map[string]any{"schema_version": "1.0.0", "revision_id": "large", "revision_hash": "0000000000000000000000000000000000000000000000000000000000000000", "revision_payload": map[string]any{"intent_id": "large-intent", "lifecycle_state": "draft", "source_references": []any{}, "items": items, "comments": []any{}, "approval_refs": []any{}}}
	data, _ := json.Marshal(artifact)
	if len(data) < 10*1024*1024 {
		t.Fatal("generated fixture did not reach 10 MiB target")
	}
	path := filepath.Join(t.TempDir(), "large.json")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	body, _ := json.Marshal(map[string]any{"protocol_version": "1.0", "request_id": "perf", "operation": "validate_intent", "payload_schema": "zintent.command/1", "payload": map[string]any{"operation": "validate_intent", "intent_path": path}})
	start := time.Now()
	cmd := exec.Command(core)
	cmd.Stdin = bytes.NewReader(body)
	out, err := cmd.Output()
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("large validation failed: %v %s", err, out)
	}
	if elapsed > 2*time.Second {
		t.Fatalf("large validation exceeded budget: %s", elapsed)
	}
}
