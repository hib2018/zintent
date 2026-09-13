package integration

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestReviewCLIEmitsOneJSONEnvelope(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	core := filepath.Join(root, "zig-out", "bin", "zintent-core")
	if _, err := os.Stat(core); err != nil {
		t.Skip("build zintent-core before integration tests")
	}
	fixture := filepath.Join(root, "tests", "fixtures", "valid-draft.json")
	cmd := exec.Command("go", "run", "./tui/cmd/zintent", "show", fixture, "--output", "json", "--core", core)
	cmd.Dir = root
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("CLI failed: %v (%s)", err, stderr.String())
	}
	var envelope struct {
		OK           bool            `json:"ok"`
		RequestID    string          `json:"request_id"`
		Result       json.RawMessage `json:"result"`
		ResultSchema string          `json:"result_schema"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("stdout is not one JSON envelope: %v; output=%q", err, stdout.String())
	}
	if !envelope.OK || envelope.RequestID == "" || envelope.ResultSchema == "" || len(envelope.Result) == 0 {
		t.Fatalf("incomplete result envelope: %+v", envelope)
	}
}
