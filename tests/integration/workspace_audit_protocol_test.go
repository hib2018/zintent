package integration

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func auditFixture(t *testing.T) (string, string, string) {
	t.Helper()
	dir := t.TempDir()
	for _, name := range []string{"revisions", "snapshots", "capabilities"} {
		if err := os.Mkdir(filepath.Join(dir, name), 0700); err != nil {
			t.Fatal(err)
		}
	}
	revision := []byte(`{"revision_id":"r1","parent_revision_id":null,"revision_payload":{"intent_id":"intent-1","lifecycle_state":"draft","items":[],"comments":[]}}`)
	if err := os.WriteFile(filepath.Join(dir, "revisions", "r1.json"), revision, 0600); err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256(revision)
	headHash := hex.EncodeToString(hash[:])
	head, _ := json.Marshal(map[string]any{"schema_version": "1.0.0", "intent_id": "intent-1", "current_revision_id": "r1", "current_revision_hash": headHash, "lifecycle_state": "draft"})
	if err := os.WriteFile(filepath.Join(dir, "HEAD.json"), head, 0600); err != nil {
		t.Fatal(err)
	}
	approvedHash := sha256.Sum256([]byte("{}"))
	snapshot, _ := json.Marshal(map[string]any{"snapshot_id": "snap", "approved_content": map[string]any{}, "approval": map[string]any{"approval_id": "a", "approved_revision_id": "r1", "approved_content_hash": hex.EncodeToString(approvedHash[:])}})
	if err := os.WriteFile(filepath.Join(dir, "snapshots", "snap.json"), snapshot, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "revisions", "interrupted.tmp"), []byte("partial"), 0600); err != nil {
		t.Fatal(err)
	}
	root, _ := filepath.Abs(filepath.Join("..", ".."))
	return dir, headHash, filepath.Join(root, "zig-out", "bin", "zintent-core")
}
func auditCall(t *testing.T, core, op string, payload map[string]any) map[string]any {
	t.Helper()
	payload["operation"] = op
	body, _ := json.Marshal(map[string]any{"protocol_version": "1.0", "request_id": "audit-" + op, "operation": op, "payload_schema": "zintent.command/1", "payload": payload})
	cmd := exec.Command(core)
	cmd.Stdin = bytes.NewReader(body)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("core %s: %v %s", op, err, out)
	}
	var envelope struct {
		OK     bool `json:"ok"`
		Result struct {
			Data map[string]any `json:"data"`
		} `json:"result"`
		Error any `json:"error"`
	}
	if err := json.Unmarshal(out, &envelope); err != nil {
		t.Fatal(err)
	}
	if !envelope.OK {
		t.Fatalf("%s failed: %s", op, out)
	}
	return envelope.Result.Data
}

func auditRaw(t *testing.T, core, op string, payload map[string]any) []byte {
	t.Helper()
	payload["operation"] = op
	body, _ := json.Marshal(map[string]any{"protocol_version": "1.0", "request_id": "raw-" + op, "operation": op, "payload_schema": "zintent.command/1", "payload": payload})
	cmd := exec.Command(core)
	cmd.Stdin = bytes.NewReader(body)
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	return out
}
func TestWorkspaceAuditAndRecoveryProtocol(t *testing.T) {
	dir, headHash, core := auditFixture(t)
	if _, err := os.Stat(core); err != nil {
		t.Skip("build core first")
	}
	auditCall(t, core, "list_revisions", map[string]any{"intent_path": dir})
	auditCall(t, core, "inspect_revision", map[string]any{"intent_path": dir, "revision_id": "r1"})
	snapshot := auditCall(t, core, "inspect_snapshot", map[string]any{"intent_path": dir, "snapshot_id": "snap"})
	if snapshot["verified"] != true {
		t.Fatal(snapshot)
	}
	status := auditCall(t, core, "recovery_status", map[string]any{"intent_path": dir})
	token, _ := status["recovery_token"].(string)
	if token == "" {
		t.Fatal(status)
	}
	cleanup := auditCall(t, core, "cleanup_temporary_files", map[string]any{"intent_path": dir, "expected_revision_id": "r1", "expected_head_hash": headHash, "recovery_token": token, "candidate_ids": []string{"interrupted.tmp"}, "operation_id": "cleanup-1", "actor": map[string]any{"actor_type": "human", "actor_id": "alice", "identity_source": "explicit_fallback", "authenticated": false}})
	if cleanup["removed_count"].(float64) != 1 {
		t.Fatal(cleanup)
	}
	if _, err := os.Stat(filepath.Join(dir, "revisions", "interrupted.tmp")); !os.IsNotExist(err) {
		t.Fatal("temporary survived cleanup")
	}
}
func TestAuditRejectsPathTraversal(t *testing.T) {
	dir, _, core := auditFixture(t)
	payload := map[string]any{"operation": "inspect_revision", "intent_path": dir, "revision_id": "../HEAD"}
	body, _ := json.Marshal(map[string]any{"protocol_version": "1.0", "request_id": "escape", "operation": "inspect_revision", "payload_schema": "zintent.command/1", "payload": payload})
	cmd := exec.Command(core)
	cmd.Stdin = bytes.NewReader(body)
	out, _ := cmd.Output()
	if !bytes.Contains(out, []byte(`"code":"path_escape"`)) {
		t.Fatalf("unexpected: %s", out)
	}
}

func TestRecoveryRejectsChangedCandidateStaleHeadAndConsumedToken(t *testing.T) {
	actor := map[string]any{"actor_type": "human", "actor_id": "alice", "identity_source": "explicit_fallback", "authenticated": false}
	dir, headHash, core := auditFixture(t)
	status := auditCall(t, core, "recovery_status", map[string]any{"intent_path": dir})
	token := status["recovery_token"].(string)
	if err := os.WriteFile(filepath.Join(dir, "revisions", "interrupted.tmp"), []byte("changed"), 0600); err != nil {
		t.Fatal(err)
	}
	base := map[string]any{"intent_path": dir, "expected_revision_id": "r1", "expected_head_hash": headHash, "recovery_token": token, "candidate_ids": []string{"interrupted.tmp"}, "operation_id": "cleanup-change", "actor": actor}
	if out := auditRaw(t, core, "cleanup_temporary_files", base); !bytes.Contains(out, []byte(`"code":"cleanup_target_changed"`)) {
		t.Fatalf("candidate change accepted: %s", out)
	}
	dir, headHash, core = auditFixture(t)
	status = auditCall(t, core, "recovery_status", map[string]any{"intent_path": dir})
	token = status["recovery_token"].(string)
	base = map[string]any{"intent_path": dir, "expected_revision_id": "stale", "expected_head_hash": headHash, "recovery_token": token, "candidate_ids": []string{"interrupted.tmp"}, "operation_id": "cleanup-stale", "actor": actor}
	if out := auditRaw(t, core, "cleanup_temporary_files", base); !bytes.Contains(out, []byte(`"code":"recovery_observation_stale"`)) {
		t.Fatalf("stale HEAD accepted: %s", out)
	}
	dir, headHash, core = auditFixture(t)
	status = auditCall(t, core, "recovery_status", map[string]any{"intent_path": dir})
	token = status["recovery_token"].(string)
	base = map[string]any{"intent_path": dir, "expected_revision_id": "r1", "expected_head_hash": headHash, "recovery_token": token, "candidate_ids": []string{"interrupted.tmp"}, "operation_id": "cleanup-once", "actor": actor}
	auditCall(t, core, "cleanup_temporary_files", base)
	if out := auditRaw(t, core, "cleanup_temporary_files", base); !bytes.Contains(out, []byte(`"code":"recovery_observation_stale"`)) {
		t.Fatalf("token reused: %s", out)
	}
}
