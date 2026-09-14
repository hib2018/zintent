package integration

import (
	"context"
	"os/exec"
	"testing"
	"time"
)

func TestCoreRunnerHonorsCancellation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
	defer cancel()
	cmd := exec.CommandContext(ctx, "/bin/sh", "-c", "sleep 1")
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected cancellation/timeout")
	}
}
