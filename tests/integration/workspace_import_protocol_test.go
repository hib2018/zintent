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

func workspaceCall(t *testing.T, core, op string, payload map[string]any) (bool, map[string]any, []byte) {
	t.Helper()
	payload["operation"] = op
	body, _ := json.Marshal(map[string]any{"protocol_version": "1.0", "request_id": "workspace-" + op, "operation": op, "payload_schema": "zintent.command/1", "payload": payload})
	cmd := exec.Command(core)
	cmd.Stdin = bytes.NewReader(body)
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	var envelope struct {
		OK     bool `json:"ok"`
		Result struct {
			Data map[string]any `json:"data"`
		} `json:"result"`
	}
	if err := json.Unmarshal(out, &envelope); err != nil {
		t.Fatal(err)
	}
	return envelope.OK, envelope.Result.Data, out
}
func TestWorkspaceListInspectImportProtocol(t *testing.T) {
	root, _ := filepath.Abs(filepath.Join("..", ".."))
	core := filepath.Join(root, "zig-out", "bin", "zintent-core")
	if _, err := os.Stat(core); err != nil {
		t.Skip("build core")
	}
	workspace := t.TempDir()
	sourcePath := filepath.Join(t.TempDir(), "draft.json")
	source, err := os.ReadFile(filepath.Join(root, "tests", "fixtures", "valid-draft.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sourcePath, source, 0600); err != nil {
		t.Fatal(err)
	}
	actor := map[string]any{"actor_type": "human", "actor_id": "alice", "identity_source": "explicit_fallback", "authenticated": false}
	ok, preview, out := workspaceCall(t, core, "inspect_draft", map[string]any{"workspace_path": workspace, "source_path": sourcePath, "actor": actor})
	if !ok {
		t.Fatalf("inspect: %s", out)
	}
	destination := preview["proposed_destination"].(string)
	token := preview["import_token"].(string)
	payload := map[string]any{"workspace_path": workspace, "source_path": sourcePath, "destination": destination, "import_token": token, "operation_id": "import-1", "actor": actor}
	ok, imported, out := workspaceCall(t, core, "import_draft", payload)
	if !ok || imported["source_preserved"] != true {
		t.Fatalf("import: %s", out)
	}
	unchanged, _ := os.ReadFile(sourcePath)
	if !bytes.Equal(source, unchanged) {
		t.Fatal("source was modified")
	}
	ok, retry, out := workspaceCall(t, core, "import_draft", payload)
	if !ok || retry["identical_retry"] != true {
		t.Fatalf("retry: %s", out)
	}
	ok, list, out := workspaceCall(t, core, "list_intents", map[string]any{"workspace_path": workspace})
	if !ok || len(list["entries"].([]any)) != 1 {
		t.Fatalf("list: %s", out)
	}
	entry := list["entries"].([]any)[0].(map[string]any)
	if entry["blocker_count"] != float64(1) {
		t.Fatalf("blocker_count=%v, want 1: %s", entry["blocker_count"], out)
	}
	if _, err := os.Stat(filepath.Join(workspace, destination, "HEAD.json")); err != nil {
		t.Fatal(err)
	}
}
func TestImportRejectsChangedSourceAndCollision(t *testing.T) {
	root, _ := filepath.Abs(filepath.Join("..", ".."))
	core := filepath.Join(root, "zig-out", "bin", "zintent-core")
	workspace := t.TempDir()
	sourcePath := filepath.Join(t.TempDir(), "draft.json")
	source, _ := os.ReadFile(filepath.Join(root, "tests", "fixtures", "valid-draft.json"))
	_ = os.WriteFile(sourcePath, source, 0600)
	actor := map[string]any{"actor_type": "human", "actor_id": "alice", "identity_source": "explicit_fallback", "authenticated": false}
	_, preview, _ := workspaceCall(t, core, "inspect_draft", map[string]any{"workspace_path": workspace, "source_path": sourcePath, "actor": actor})
	_ = os.WriteFile(sourcePath, append(source, ' '), 0600)
	ok, _, out := workspaceCall(t, core, "import_draft", map[string]any{"workspace_path": workspace, "source_path": sourcePath, "destination": preview["proposed_destination"], "import_token": preview["import_token"], "operation_id": "changed-source", "actor": actor})
	if ok || !bytes.Contains(out, []byte(`"code":"draft_invalid"`)) {
		t.Fatalf("changed source accepted: %s", out)
	}
}

func TestWorkspaceDiscoveryIsDirectChildOnlyAndReportsCorruption(t *testing.T) {
	root, _ := filepath.Abs(filepath.Join("..", ".."))
	core := filepath.Join(root, "zig-out", "bin", "zintent-core")
	workspace := t.TempDir()
	if err := os.Mkdir(filepath.Join(workspace, "corrupt"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(workspace, "nested", "intent"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "tests", "fixtures"), filepath.Join(workspace, "linked")); err != nil {
		t.Fatal(err)
	}
	ok, data, out := workspaceCall(t, core, "list_intents", map[string]any{"workspace_path": workspace})
	if !ok {
		t.Fatalf("list: %s", out)
	}
	if len(data["entries"].([]any)) != 0 {
		t.Fatalf("unexpected entry: %s", out)
	}
	findings := data["findings"].([]any)
	if len(findings) != 2 {
		t.Fatalf("findings=%d, want corrupt direct directories only: %s", len(findings), out)
	}
	if bytes.Contains(out, []byte(`"record_id":"linked"`)) || bytes.Contains(out, []byte(`"record_id":"intent"`)) {
		t.Fatalf("followed symlink or nested child: %s", out)
	}
}

func TestImportCollisionDoesNotPublishTemporaryDirectory(t *testing.T) {
	root, _ := filepath.Abs(filepath.Join("..", ".."))
	core := filepath.Join(root, "zig-out", "bin", "zintent-core")
	workspace := t.TempDir()
	sourcePath := filepath.Join(t.TempDir(), "draft.json")
	source, _ := os.ReadFile(filepath.Join(root, "tests", "fixtures", "valid-draft.json"))
	_ = os.WriteFile(sourcePath, source, 0600)
	actor := map[string]any{"actor_type": "human", "actor_id": "alice", "identity_source": "explicit_fallback", "authenticated": false}
	_, preview, _ := workspaceCall(t, core, "inspect_draft", map[string]any{"workspace_path": workspace, "source_path": sourcePath, "actor": actor})
	destination := preview["proposed_destination"].(string)
	if err := os.Mkdir(filepath.Join(workspace, destination), 0700); err != nil {
		t.Fatal(err)
	}
	ok, _, out := workspaceCall(t, core, "import_draft", map[string]any{"workspace_path": workspace, "source_path": sourcePath, "destination": destination, "import_token": preview["import_token"], "operation_id": "collision", "actor": actor})
	if ok {
		t.Fatalf("collision accepted: %s", out)
	}
	entries, _ := os.ReadDir(workspace)
	for _, entry := range entries {
		if strings.Contains(entry.Name(), ".tmp-") {
			t.Fatalf("partial import left behind: %s", entry.Name())
		}
	}
}
