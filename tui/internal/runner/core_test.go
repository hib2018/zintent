package runner

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/hib2018/zintent/tui/internal/protocol"
)

func TestCoreTimeoutTerminatesHungProcess(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture")
	}
	fixture := filepath.Join(t.TempDir(), "hung-core")
	if err := os.WriteFile(fixture, []byte("#!/bin/sh\nexec sleep 60\n"), 0700); err != nil {
		t.Fatal(err)
	}
	_, err := (Core{Executable: fixture, Timeout: 20 * time.Millisecond}).Run(context.Background(), protocol.Request{ProtocolVersion: protocol.Version, RequestID: "timeout", Operation: "protocol_info", PayloadSchema: "zintent.command/1", Payload: json.RawMessage(`{"operation":"protocol_info"}`)})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error=%v", err)
	}
}

func TestLimitedBufferRejectsOverflow(t *testing.T) {
	b := limitedBuffer{limit: 3}
	if _, err := b.Write([]byte("four")); err == nil {
		t.Fatal("expected bounded output failure")
	}
}

func TestCoreRequiresExecutable(t *testing.T) {
	_, err := (Core{}).Run(t.Context(), protocol.Request{})
	if err == nil {
		t.Fatal("expected missing executable error")
	}
}
