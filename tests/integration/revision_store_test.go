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

func TestDirectoryMutationPublishesVerifiedHEADAndIdenticalRetry(t *testing.T) {
	root, _ := filepath.Abs(filepath.Join("..", ".."))
	core := filepath.Join(root, "zig-out", "bin", "zintent-core")
	if _, err := os.Stat(core); err != nil {
		t.Skip("build zintent-core before integration tests")
	}
	source, err := os.ReadFile(filepath.Join(root, "tests", "fixtures", "valid-draft.json"))
	if err != nil {
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
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "revisions"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "revisions", draft.RevisionID+".json"), source, 0o600); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(source)
	head := map[string]any{"schema_version": "1.0.0", "intent_id": "01900000-0000-7000-8000-000000000003", "current_revision_id": draft.RevisionID, "current_revision_hash": hex.EncodeToString(digest[:]), "lifecycle_state": "draft", "approved_snapshot_ref": nil}
	headBytes, _ := json.Marshal(head)
	if err := os.WriteFile(filepath.Join(dir, "HEAD.json"), headBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "intent.json"), source, 0o600); err != nil {
		t.Fatal(err)
	}
	run := func(args ...string) []byte {
		cmd := exec.Command("go", append([]string{"run", "./tui/cmd/zintent"}, args...)...)
		cmd.Dir = root
		var out, errOut bytes.Buffer
		cmd.Stdout, cmd.Stderr = &out, &errOut
		if err := cmd.Run(); err != nil {
			t.Fatalf("CLI failed: %v (%s)", err, errOut.String())
		}
		return out.Bytes()
	}
	item := draft.Payload.Items[0].ID
	args := []string{"item", "accept", dir, item, "--expected-revision", draft.RevisionID, "--operation-id", "directory-op", "--actor-id", "alice", "--core", core, "--output", "json"}
	run(args...)
	run(args...)
	var result struct {
		Result struct {
			Data struct {
				Intent struct {
					RevisionID string `json:"revision_id"`
				} `json:"intent"`
			} `json:"data"`
		} `json:"result"`
	}
	if err := json.Unmarshal(run("show", dir, "--output", "json", "--core", core), &result); err != nil {
		t.Fatal(err)
	}
	if result.Result.Data.Intent.RevisionID != "directory-op" {
		t.Fatalf("unexpected HEAD revision: %+v", result)
	}
	if _, err := os.Stat(filepath.Join(dir, "revisions", "directory-op.json")); err != nil {
		t.Fatal(err)
	}
}
