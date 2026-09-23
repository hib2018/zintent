package integration

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestApproveRefusesNonTTYBeforeCoreInvocation(t *testing.T) {
	root, _ := filepath.Abs(filepath.Join("..", ".."))
	cmd := exec.Command("go", "run", "./tui/cmd/zintent", "approve", "missing", "--revision", "r-1", "--operation-id", "op-1", "--actor-id", "alice")
	cmd.Dir = root
	cmd.Stdin = bytes.NewReader(nil)
	out, err := cmd.CombinedOutput()
	if err == nil || !strings.Contains(string(out), "tty_required") {
		t.Fatalf("expected tty_required, got err=%v output=%s", err, out)
	}
}

func TestPrepareAndApproveEligibleRevision(t *testing.T) {
	root, _ := filepath.Abs(filepath.Join("..", ".."))
	core := filepath.Join(root, "zig-out", "bin", "zintent-core")
	if _, err := os.Stat(core); err != nil {
		t.Skip("build zintent-core before integration tests")
	}
	source, err := os.ReadFile(filepath.Join(root, "tests", "fixtures", "valid-draft.json"))
	if err != nil {
		t.Fatal(err)
	}
	intent := filepath.Join(t.TempDir(), "intent.json")
	if err := os.WriteFile(intent, source, 0o600); err != nil {
		t.Fatal(err)
	}
	var draft struct {
		RevisionID string `json:"revision_id"`
		Payload    struct {
			Items []struct {
				ID string `json:"item_id"`
			} `json:"items"`
		} `json:"revision_payload"`
	}
	if err := json.Unmarshal(source, &draft); err != nil {
		t.Fatal(err)
	}
	runCLI := func(args ...string) {
		cmd := exec.Command("go", append([]string{"run", "./tui/cmd/zintent"}, args...)...)
		cmd.Dir = root
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("CLI failed: %v %s", err, out)
		}
	}
	runCLI("start-review", intent, "--expected-revision", draft.RevisionID, "--operation-id", "approval-start", "--actor-id", "alice", "--core", core)
	runCLI("item", "accept", intent, draft.Payload.Items[0].ID, "--expected-revision", "approval-start", "--operation-id", "approval-accept", "--actor-id", "alice", "--core", core)
	runCLI("complete-review", intent, "--expected-revision", "approval-accept", "--operation-id", "approval-complete", "--actor-id", "alice", "--core", core)
	request := func(operation string, payload map[string]any) map[string]any {
		payload["operation"] = operation
		body, _ := json.Marshal(map[string]any{"protocol_version": "1.0", "request_id": "approval-test", "operation": operation, "payload_schema": "zintent.command/1", "payload": payload})
		cmd := exec.Command(core)
		cmd.Stdin = bytes.NewReader(body)
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("core failed: %v", err)
		}
		var response struct {
			OK     bool            `json:"ok"`
			Result json.RawMessage `json:"result"`
			Error  any             `json:"error"`
		}
		if err := json.Unmarshal(out, &response); err != nil {
			t.Fatal(err)
		}
		if !response.OK {
			t.Fatalf("approval operation failed: %s", out)
		}
		var result struct {
			Data map[string]any `json:"data"`
		}
		if err := json.Unmarshal(response.Result, &result); err != nil {
			t.Fatal(err)
		}
		return result.Data
	}
	prepared := request("prepare_approval", map[string]any{"intent_path": intent, "expected_revision_id": "approval-complete", "actor": map[string]any{"actor_type": "human", "actor_id": "alice", "identity_source": "explicit_fallback", "authenticated": false}, "interactive_tty": true})
	confirmation := prepared["confirmation"].(map[string]any)
	approved := request("approve_intent", map[string]any{"intent_path": intent, "expected_revision_id": "approval-complete", "operation_id": "approval-approved", "actor": map[string]any{"actor_type": "human", "actor_id": "alice", "identity_source": "explicit_fallback", "authenticated": false}, "interactive_tty": true, "confirmation_token": confirmation["token_id"], "challenge_response": confirmation["challenge"]})
	snapshotPath, ok := approved["snapshot_path"].(string)
	if !ok || snapshotPath == "" {
		t.Fatalf("approval did not return snapshot path: %#v", approved)
	}
	if _, err := os.Stat(snapshotPath); err != nil {
		t.Fatalf("snapshot was not published: %v", err)
	}
}
