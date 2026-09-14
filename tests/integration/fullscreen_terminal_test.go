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

func TestWorkspaceRefusesNonTTY(t *testing.T) {
	root, _ := filepath.Abs(filepath.Join("..", ".."))
	cmd := exec.Command("go", "run", "./tui/cmd/zintent", "workspace", t.TempDir())
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "GOCACHE="+filepath.Join(t.TempDir(), "go-cache"))
	out, err := cmd.CombinedOutput()
	if err == nil || !strings.Contains(string(out), "tty_required") {
		t.Fatalf("err=%v out=%s", err, out)
	}
}

func TestWorkspaceCtrlCAndCoreCrashRestoreTerminal(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("BSD script PTY test")
	}
	root, _ := filepath.Abs(filepath.Join("..", ".."))
	binary := filepath.Join(t.TempDir(), "zintent")
	build := exec.Command("go", "build", "-o", binary, "./tui/cmd/zintent")
	build.Dir = root
	build.Env = append(os.Environ(), "GOCACHE="+filepath.Join(t.TempDir(), "go-cache"))
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v %s", err, out)
	}
	for _, tc := range []struct{ name, core, key string }{{"ctrl-c", filepath.Join(root, "zig-out/bin/zintent-core"), "\x03"}, {"core-crash", "/definitely/missing/zintent-core", "q"}} {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.Command("/usr/bin/script", "-q", "/dev/null", binary, "workspace", t.TempDir(), "--core", tc.core)
			input, _ := cmd.StdinPipe()
			var out bytes.Buffer
			cmd.Stdout = &out
			cmd.Stderr = &out
			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}
			time.Sleep(time.Second)
			_, _ = input.Write([]byte(tc.key))
			_ = input.Close()
			if err := cmd.Wait(); err != nil && tc.name != "ctrl-c" {
				t.Fatalf("PTY: %v %s", err, out.String())
			}
			if !strings.Contains(out.String(), "\x1b[?1049h") || !strings.Contains(out.String(), "\x1b[?1049l") || !strings.Contains(out.String(), "\x1b[?25h") {
				t.Fatalf("terminal not restored: %q", out.String())
			}
		})
	}
}
