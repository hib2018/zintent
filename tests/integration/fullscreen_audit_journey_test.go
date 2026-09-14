package integration

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestAuditRecoveryJourneyLeavesProtectedArtifacts(t *testing.T) {
	dir, headHash, core := auditFixture(t)
	status := auditCall(t, core, "recovery_status", map[string]any{"intent_path": dir})
	token := status["recovery_token"].(string)
	auditCall(t, core, "cleanup_temporary_files", map[string]any{"intent_path": dir, "expected_revision_id": "r1", "expected_head_hash": headHash, "recovery_token": token, "candidate_ids": []string{"interrupted.tmp"}, "operation_id": "cleanup-pty", "actor": map[string]any{"actor_type": "human", "actor_id": "alice", "identity_source": "explicit_fallback", "authenticated": false}})
	if _, err := os.Stat(filepath.Join(dir, "HEAD.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "revisions", "r1.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "snapshots", "snap.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "revisions", "interrupted.tmp")); !os.IsNotExist(err) {
		t.Fatal("confirmed temporary was not removed")
	}
	if runtime.GOOS != "darwin" {
		return
	}
	root, _ := filepath.Abs(filepath.Join("..", ".."))
	binary := filepath.Join(t.TempDir(), "zintent")
	build := exec.Command("go", "build", "-o", binary, "./tui/cmd/zintent")
	build.Dir = root
	build.Env = append(os.Environ(), "GOCACHE="+filepath.Join(t.TempDir(), "go-cache"))
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v %s", err, out)
	}
	cmd := exec.Command("/usr/bin/script", "-q", "/dev/null", binary, "workspace", filepath.Dir(dir))
	input, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	time.Sleep(300 * time.Millisecond)
	_, _ = input.Write([]byte("hRq"))
	_ = input.Close()
	if err := cmd.Wait(); err != nil {
		t.Fatalf("PTY audit: %v %s", err, out.String())
	}
	if !strings.Contains(out.String(), "\x1b[?25h") {
		t.Fatalf("cursor was not restored: %q", out.String())
	}
}
