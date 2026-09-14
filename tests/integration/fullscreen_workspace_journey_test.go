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

func TestFullscreenWorkspaceListImportResumeJourney(t *testing.T) {
	root, _ := filepath.Abs(filepath.Join("..", ".."))
	core := filepath.Join(root, "zig-out", "bin", "zintent-core")
	if _, err := os.Stat(core); err != nil {
		t.Skip("build zintent-core")
	}
	workspace := t.TempDir()
	sourcePath := filepath.Join(t.TempDir(), "draft.json")
	source, _ := os.ReadFile(filepath.Join(root, "tests", "fixtures", "valid-draft.json"))
	if err := os.WriteFile(sourcePath, source, 0600); err != nil {
		t.Fatal(err)
	}
	actor := map[string]any{"actor_type": "human", "actor_id": "alice", "identity_source": "explicit_fallback", "authenticated": false}
	_, preview, _ := workspaceCall(t, core, "inspect_draft", map[string]any{"workspace_path": workspace, "source_path": sourcePath, "actor": actor})
	ok, _, out := workspaceCall(t, core, "import_draft", map[string]any{"workspace_path": workspace, "source_path": sourcePath, "destination": preview["proposed_destination"], "import_token": preview["import_token"], "operation_id": "pty-import", "actor": actor})
	if !ok {
		t.Fatalf("import: %s", out)
	}
	if runtime.GOOS != "darwin" {
		return
	}
	binary := filepath.Join(t.TempDir(), "zintent")
	build := exec.Command("go", "build", "-o", binary, "./tui/cmd/zintent")
	build.Dir = root
	build.Env = append(os.Environ(), "GOCACHE="+filepath.Join(t.TempDir(), "go-cache"))
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v %s", err, out)
	}
	cmd := exec.Command("/usr/bin/script", "-q", "/dev/null", binary, "workspace", workspace, "--core", core)
	input, _ := cmd.StdinPipe()
	var transcript bytes.Buffer
	cmd.Stdout = &transcript
	cmd.Stderr = &transcript
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Second)
	_, _ = input.Write([]byte("q"))
	_ = input.Close()
	if err := cmd.Wait(); err != nil {
		t.Fatalf("workspace: %v %s", err, transcript.String())
	}
	if !strings.Contains(transcript.String(), "\x1b[?25h") {
		t.Fatal("cursor not restored")
	}
}

func TestGoRuntimeDoesNotWriteIntentArtifacts(t *testing.T) {
	root, _ := filepath.Abs(filepath.Join("..", ".."))
	err := filepath.Walk(filepath.Join(root, "tui"), func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || strings.HasSuffix(path, "_test.go") {
			return err
		}
		body, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		for _, forbidden := range []string{"os.WriteFile(", "os.Rename(", "os.Remove("} {
			if bytes.Contains(body, []byte(forbidden)) {
				t.Errorf("direct artifact write in %s: %s", path, forbidden)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
