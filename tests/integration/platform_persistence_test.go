package integration

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestPlatformAtomicFilePrimitives(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "artifact.json")
	if err := os.WriteFile(target, []byte("first"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(target, target+".next"); err != nil {
		t.Fatal(err)
	}
	if got, err := os.ReadFile(target + ".next"); err != nil || !bytes.Equal(got, []byte("first")) {
		t.Fatalf("atomic replacement verification failed: %v %q", err, got)
	}
}
