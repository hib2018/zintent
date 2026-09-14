package command

import (
	"context"
	"errors"
	"testing"

	"github.com/hib2018/zintent/tui/internal/protocol"
)

type fakeRunner struct {
	response protocol.Response
	err      error
}

func (f fakeRunner) Run(context.Context, protocol.Request) (protocol.Response, error) {
	return f.response, f.err
}

type blockingRunner struct {
	entered chan struct{}
	release chan struct{}
}

func (b blockingRunner) Run(context.Context, protocol.Request) (protocol.Response, error) {
	close(b.entered)
	<-b.release
	return protocol.Response{RequestID: "first"}, nil
}

func TestExecutorRejectsLateResponse(t *testing.T) {
	e := NewExecutor(context.Background(), fakeRunner{response: protocol.Response{RequestID: "old"}})
	defer e.Close()
	_, err := e.Execute(protocol.Request{RequestID: "new"}, false)
	if !errors.Is(err, ErrLateResponse) {
		t.Fatalf("error=%v", err)
	}
}

func TestExecutorMarksFailedMutationIndeterminate(t *testing.T) {
	e := NewExecutor(context.Background(), fakeRunner{err: context.DeadlineExceeded})
	defer e.Close()
	_, err := e.Execute(protocol.Request{RequestID: "mutation"}, true)
	if !errors.Is(err, ErrIndeterminate) || !e.NeedsCanonicalReload() {
		t.Fatalf("error=%v reload=%v", err, e.NeedsCanonicalReload())
	}
	e.MarkReloaded()
	if e.NeedsCanonicalReload() {
		t.Fatal("canonical reload must clear state")
	}
}

func TestExecutorAllowsOnlyOneActiveMutation(t *testing.T) {
	runner := blockingRunner{entered: make(chan struct{}), release: make(chan struct{})}
	e := NewExecutor(context.Background(), runner)
	defer e.Close()
	done := make(chan error, 1)
	go func() { _, err := e.Execute(protocol.Request{RequestID: "first"}, true); done <- err }()
	<-runner.entered
	_, err := e.Execute(protocol.Request{RequestID: "second"}, true)
	if !errors.Is(err, ErrMutationInFlight) {
		t.Fatalf("error=%v", err)
	}
	close(runner.release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}
