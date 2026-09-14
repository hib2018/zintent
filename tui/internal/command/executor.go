package command

import (
	"context"
	"errors"
	"sync"

	"github.com/hib2018/zintent/tui/internal/protocol"
)

var ErrMutationInFlight = errors.New("a mutation is already in flight")
var ErrLateResponse = errors.New("late or mismatched core response")

type Runner interface {
	Run(context.Context, protocol.Request) (protocol.Response, error)
}

type Executor struct {
	core           Runner
	ctx            context.Context
	cancel         context.CancelFunc
	mu             sync.Mutex
	activeMutation string
}

func NewExecutor(parent context.Context, core Runner) *Executor {
	ctx, cancel := context.WithCancel(parent)
	return &Executor{core: core, ctx: ctx, cancel: cancel}
}
func (e *Executor) Close() { e.cancel() }

func (e *Executor) Execute(request protocol.Request, mutation bool) (protocol.Response, error) {
	if mutation {
		e.mu.Lock()
		if e.activeMutation != "" {
			e.mu.Unlock()
			return protocol.Response{}, ErrMutationInFlight
		}
		e.activeMutation = request.RequestID
		e.mu.Unlock()
		defer func() { e.mu.Lock(); e.activeMutation = ""; e.mu.Unlock() }()
	}
	response, err := e.core.Run(e.ctx, request)
	if err != nil {
		return response, err
	}
	if response.RequestID != request.RequestID {
		return protocol.Response{}, ErrLateResponse
	}
	return response, nil
}
