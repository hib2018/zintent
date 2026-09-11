package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"time"

	"github.com/hib2018/zintent/tui/internal/protocol"
)

type Core struct {
	Executable string
	Timeout    time.Duration
}

func (c Core) Run(ctx context.Context, request protocol.Request) (protocol.Response, error) {
	if c.Executable == "" {
		return protocol.Response{}, errors.New("core executable is required")
	}
	timeout := c.Timeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	body, err := json.Marshal(request)
	if err != nil {
		return protocol.Response{}, err
	}
	cmd := exec.CommandContext(ctx, c.Executable)
	cmd.Stdin = bytes.NewReader(body)
	var stdout, stderr limitedBuffer
	stdout.limit, stderr.limit = protocol.MaxMessageBytes, protocol.MaxMessageBytes
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return protocol.Response{}, ctx.Err()
		}
		return protocol.Response{}, fmt.Errorf("core failed: %w: %s", err, stderr.String())
	}
	var response protocol.Response
	if err := protocol.DecodeStrict(stdout.Bytes(), &response); err != nil {
		return protocol.Response{}, err
	}
	if err := protocol.ValidateResponse(request, response); err != nil {
		return protocol.Response{}, err
	}
	return response, nil
}

type limitedBuffer struct {
	bytes.Buffer
	limit int
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	if b.Len()+len(p) > b.limit {
		return 0, io.ErrShortBuffer
	}
	return b.Buffer.Write(p)
}
