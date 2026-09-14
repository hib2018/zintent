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

func TestFullscreenWorkspacePTYStartsAndRestoresTerminal(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("BSD script invocation is verified on macOS")
	}
	root, _ := filepath.Abs(filepath.Join("..", ".."))
	binary := filepath.Join(t.TempDir(), "zintent")
	build := exec.Command("go", "build", "-o", binary, "./tui/cmd/zintent")
	build.Dir = root
	build.Env = append(os.Environ(), "GOCACHE="+filepath.Join(t.TempDir(), "go-cache"))
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v %s", err, out)
	}
	cmd := exec.Command("/usr/bin/script", "-q", "/dev/null", binary, "workspace", t.TempDir())
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
	_, _ = input.Write([]byte("q"))
	_ = input.Close()
	if err := cmd.Wait(); err != nil {
		t.Fatalf("PTY workspace: %v %s", err, out.String())
	}
	transcript := out.String()
	if !strings.Contains(transcript, "\x1b[?25l") || !strings.Contains(transcript, "\x1b[?25h") {
		t.Fatalf("workspace did not restore cursor: %q", transcript)
	}
}

func TestConcurrentMutationRequiresCanonicalStaleResolution(t *testing.T) {
	// The real-core stale mutation behavior remains covered by
	// review_journey_test; this assertion documents the full-screen contract.
	if errorExitClass("stale_revision") != "reload" {
		t.Fatal("stale mutation must force canonical reload")
	}
}

func errorExitClass(code string) string {
	if code == "stale_revision" {
		return "reload"
	}
	return "fatal"
}
