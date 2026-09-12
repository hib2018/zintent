package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"time"

	"github.com/hib2018/zintent/tui/internal/output"
	"github.com/hib2018/zintent/tui/internal/protocol"
	"github.com/hib2018/zintent/tui/internal/runner"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(exitCode(err))
	}
}

func run(args []string) error {
	parsed, err := parseArgs(args)
	if err != nil {
		return exitError{2, err}
	}
	payload, err := json.Marshal(parsed.payload)
	if err != nil {
		return err
	}
	request := protocol.Request{ProtocolVersion: protocol.Version, RequestID: fmt.Sprintf("cli-%d", time.Now().UnixNano()), Operation: parsed.operation, PayloadSchema: "zintent.command/1", Payload: payload}
	response, err := (runner.Core{Executable: parsed.corePath}).Run(context.Background(), request)
	if err != nil {
		return err
	}
	if err := output.Write(os.Stdout, response, parsed.jsonMode); err != nil {
		return err
	}
	if !response.OK {
		return exitError{errorExit(response.Error.Code), errors.New(response.Error.Message)}
	}
	return nil
}

type options struct {
	operation, corePath string
	jsonMode            bool
	payload             map[string]any
}

func parseArgs(args []string) (options, error) {
	o := options{operation: "protocol_info", corePath: findCore(), payload: map[string]any{}}
	positionals := make([]string, 0, 3)
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			o.jsonMode = true
		case "--output":
			if i+1 >= len(args) {
				return o, errors.New("--output requires human or json")
			}
			i++
			if args[i] != "human" && args[i] != "json" {
				return o, errors.New("--output requires human or json")
			}
			o.jsonMode = args[i] == "json"
		case "--core":
			if i+1 >= len(args) {
				return o, errors.New("--core requires a path")
			}
			i++
			o.corePath = args[i]
		default:
			positionals = append(positionals, args[i])
		}
	}
	if len(positionals) > 0 {
		o.operation = positionals[0]
	}
	if alias, ok := map[string]string{"show": "show_intent", "validate": "validate_intent", "diff": "diff_revisions"}[o.operation]; ok {
		o.operation = alias
	}
	switch o.operation {
	case "protocol_info":
		if len(positionals) > 1 {
			return o, errors.New("protocol_info accepts no Intent")
		}
	case "show_intent", "validate_intent", "diff_revisions":
		if len(positionals) != 2 {
			return o, fmt.Errorf("%s requires one Intent path", positionals[0])
		}
		o.payload["intent_path"] = positionals[1]
	default:
		return o, fmt.Errorf("unknown command %q", positionals[0])
	}
	o.payload["operation"] = o.operation
	return o, nil
}

func localActor(fallback string) (map[string]any, error) {
	id, source := fallback, "explicit_fallback"
	if id == "" {
		if current, err := user.Current(); err == nil {
			id, source = current.Username, "os_user"
		}
	}
	if id == "" {
		return nil, errors.New("missing local actor; pass --actor-id")
	}
	return map[string]any{"actor_type": "human", "actor_id": id, "identity_source": source, "authenticated": false}, nil
}

func findCore() string {
	if value := os.Getenv("ZINTENT_CORE"); value != "" {
		return value
	}
	if exe, err := os.Executable(); err == nil {
		candidate := filepath.Join(filepath.Dir(exe), "zintent-core")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return "../zig-out/bin/zintent-core"
}

type exitError struct {
	code int
	err  error
}

func (e exitError) Error() string { return e.err.Error() }
func (e exitError) Unwrap() error { return e.err }
func exitCode(err error) int {
	var value exitError
	if errors.As(err, &value) {
		return value.code
	}
	return 70
}
func errorExit(code string) int {
	switch code {
	case "invalid_artifact", "unsupported_schema", "duplicate_id", "broken_reference":
		return 3
	case "stale_revision", "operation_id_conflict":
		return 4
	case "approval_ineligible", "invalid_transition", "tty_required":
		return 5
	case "persistence_failure":
		return 6
	default:
		return 70
	}
}
