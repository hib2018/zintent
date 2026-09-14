package integration

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestProcessRestartCanReadCanonicalArtifact(t *testing.T) {
	root, _ := filepath.Abs(filepath.Join("..", ".."))
	core := filepath.Join(root, "zig-out", "bin", "zintent-core")
	source, err := os.ReadFile(filepath.Join(root, "tests", "fixtures", "valid-draft.json"))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "intent.json")
	if err := os.WriteFile(path, source, 0o600); err != nil {
		t.Fatal(err)
	}
	body, _ := json.Marshal(map[string]any{"protocol_version": "1.0", "request_id": "restart", "operation": "show_intent", "payload_schema": "zintent.command/1", "payload": map[string]any{"operation": "show_intent", "intent_path": path}})
	cmd := exec.Command(core)
	cmd.Stdin = bytes.NewReader(body)
	if out, err := cmd.Output(); err != nil || !bytes.Contains(out, []byte(`"ok":true`)) {
		t.Fatalf("first process failed: %v %s", err, out)
	}
	cmd = exec.Command(core)
	cmd.Stdin = bytes.NewReader(body)
	if out, err := cmd.Output(); err != nil || !bytes.Contains(out, []byte(`"ok":true`)) {
		t.Fatalf("restart process failed: %v %s", err, out)
	}
}
