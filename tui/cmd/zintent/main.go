package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/term"
	"github.com/hib2018/zintent/tui/internal/output"
	"github.com/hib2018/zintent/tui/internal/protocol"
	"github.com/hib2018/zintent/tui/internal/runner"
	"github.com/hib2018/zintent/tui/internal/ui"
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
	if parsed.interactive && !stdinIsTerminal() {
		return exitError{5, errors.New("tty_required: review requires an interactive terminal")}
	}
	if parsed.interactive {
		if parsed.operation == "approve_intent" {
			return runApproval(context.Background(), parsed)
		}
		if parsed.operation == "workspace" {
			return runWorkspace(parsed)
		}
		return runReview(context.Background(), parsed)
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

func runWorkspace(parsed options) error {
	if !term.IsTerminal(os.Stdin.Fd()) || !term.IsTerminal(os.Stdout.Fd()) {
		return exitError{5, errors.New("tty_required: workspace requires an interactive terminal")}
	}
	actor, err := localActor(parsed.actorID)
	if err != nil {
		return err
	}
	workspacePath, _ := parsed.payload["workspace_path"].(string)
	workspacePath, err = filepath.Abs(workspacePath)
	if err != nil {
		return fmt.Errorf("resolve workspace path: %w", err)
	}
	draftRoot := filepath.Join(filepath.Dir(filepath.Clean(workspacePath)), "draft")
	model := ui.NewWorkspace()
	model.WorkspacePath = workspacePath
	model.WorkspaceExecutor = ui.WorkspaceCoreCommands{Core: runner.Core{Executable: parsed.corePath}, WorkspacePath: workspacePath, DraftRoot: draftRoot, Actor: actor}
	_, err = tea.NewProgram(model, tea.WithInput(os.Stdin), tea.WithOutput(os.Stdout)).Run()
	return err
}

func runApproval(ctx context.Context, parsed options) error {
	if !stdinIsTerminal() {
		return exitError{5, errors.New("tty_required: approval requires an interactive terminal")}
	}
	core := runner.Core{Executable: parsed.corePath}
	actor := parsed.payload["actor"]
	preparePayload := map[string]any{"operation": "prepare_approval", "intent_path": parsed.payload["intent_path"], "expected_revision_id": parsed.payload["expected_revision_id"], "actor": actor, "interactive_tty": true}
	prepareBody, err := json.Marshal(preparePayload)
	if err != nil {
		return err
	}
	prepare, err := core.Run(ctx, protocol.Request{ProtocolVersion: protocol.Version, RequestID: fmt.Sprintf("prepare-%d", time.Now().UnixNano()), Operation: "prepare_approval", PayloadSchema: "zintent.command/1", Payload: prepareBody})
	if err != nil {
		return err
	}
	if !prepare.OK {
		return responseError(prepare)
	}
	var prepared struct {
		Data struct {
			Confirmation struct {
				TokenID   string `json:"token_id"`
				Challenge string `json:"challenge"`
			} `json:"confirmation"`
		} `json:"data"`
	}
	if err := json.Unmarshal(prepare.Result, &prepared); err != nil {
		return err
	}
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return exitError{5, errors.New("tty_required: approval challenge TTY unavailable")}
	}
	defer tty.Close()
	fmt.Fprintf(tty, "Approval challenge: %s\nType the challenge exactly to approve: ", prepared.Data.Confirmation.Challenge)
	answer, err := bufio.NewReader(tty).ReadString('\n')
	if err != nil {
		return err
	}
	answer = strings.TrimSpace(answer)
	approvePayload := map[string]any{"operation": "approve_intent", "intent_path": parsed.payload["intent_path"], "expected_revision_id": parsed.payload["expected_revision_id"], "operation_id": parsed.payload["operation_id"], "actor": actor, "confirmation_token": prepared.Data.Confirmation.TokenID, "challenge_response": answer, "interactive_tty": true}
	approveBody, err := json.Marshal(approvePayload)
	if err != nil {
		return err
	}
	approved, err := core.Run(ctx, protocol.Request{ProtocolVersion: protocol.Version, RequestID: fmt.Sprintf("approve-%d", time.Now().UnixNano()), Operation: "approve_intent", PayloadSchema: "zintent.command/1", Payload: approveBody})
	if err != nil {
		return err
	}
	if err := output.Write(os.Stdout, approved, parsed.jsonMode); err != nil {
		return err
	}
	if !approved.OK {
		return responseError(approved)
	}
	return nil
}

func responseError(response protocol.Response) error {
	if response.Error == nil {
		return errors.New("operation failed")
	}
	return exitError{errorExit(response.Error.Code), errors.New(response.Error.Message)}
}

func runReview(ctx context.Context, parsed options) error {
	core := runner.Core{Executable: parsed.corePath}
	payload, err := json.Marshal(map[string]any{"operation": "show_intent", "intent_path": parsed.payload["intent_path"]})
	if err != nil {
		return err
	}
	response, err := core.Run(ctx, protocol.Request{ProtocolVersion: protocol.Version, RequestID: fmt.Sprintf("review-%d", time.Now().UnixNano()), Operation: "show_intent", PayloadSchema: "zintent.command/1", Payload: payload})
	if err != nil {
		return err
	}
	if !response.OK {
		if response.Error != nil {
			return exitError{errorExit(response.Error.Code), errors.New(response.Error.Message)}
		}
		return errors.New("show_intent failed")
	}
	intentID, revisionID, lifecycle, items, err := decodeReviewIntent(response.Result)
	if err != nil {
		return err
	}
	actor, _ := parsed.payload["actor"].(map[string]any)
	actorID, _ := actor["actor_id"].(string)
	model := ui.New(items)
	model.IntentID = intentID
	model.Revision = revisionID
	model.Lifecycle = lifecycle
	model.Actor = actorID
	model.IntentPath, _ = parsed.payload["intent_path"].(string)
	model.ExpectedRevision = model.Revision
	model.Executor = &ui.CoreCommands{Core: core, IntentPath: model.IntentPath, Expected: model.ExpectedRevision, Actor: actor}
	_, err = tea.NewProgram(model).Run()
	return err
}

func decodeReviewIntent(raw json.RawMessage) (string, string, string, []ui.Item, error) {
	var result struct {
		Data struct {
			Intent struct {
				RevisionID      string `json:"revision_id"`
				RevisionPayload struct {
					IntentID  string `json:"intent_id"`
					Lifecycle string `json:"lifecycle_state"`
					Items     []struct {
						ID        string `json:"item_id"`
						Kind      string `json:"kind"`
						Statement string `json:"statement"`
						Status    string `json:"review_status"`
					} `json:"items"`
				} `json:"revision_payload"`
			} `json:"intent"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return "", "", "", nil, err
	}
	payload := result.Data.Intent.RevisionPayload
	items := make([]ui.Item, 0, len(payload.Items))
	for _, item := range payload.Items {
		items = append(items, ui.Item{ID: item.ID, Kind: item.Kind, Statement: item.Statement, Status: item.Status})
	}
	return payload.IntentID, result.Data.Intent.RevisionID, payload.Lifecycle, items, nil
}

func stdinIsTerminal() bool {
	info, err := os.Stdin.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

type options struct {
	operation, corePath string
	jsonMode            bool
	actorID             string
	interactive         bool
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
		case "--actor-id":
			if i+1 >= len(args) {
				return o, errors.New("--actor-id requires a value")
			}
			i++
			o.actorID = args[i]
		case "--expected-revision", "--revision", "--operation-id", "--preview-token", "--statement-file", "--reason-file", "--body-file", "--resolution-revision", "--from", "--to":
			if i+1 >= len(args) {
				return o, fmt.Errorf("%s requires a value", args[i])
			}
			key := map[string]string{
				"--expected-revision":   "expected_revision_id",
				"--revision":            "expected_revision_id",
				"--operation-id":        "operation_id",
				"--preview-token":       "preview_token",
				"--statement-file":      "statement_file",
				"--reason-file":         "reason_file",
				"--body-file":           "body_file",
				"--resolution-revision": "resolution_revision_id",
				"--from":                "from_revision_id",
				"--to":                  "to_revision_id",
			}[args[i]]
			if key == "" {
				key = strings.TrimPrefix(args[i], "--")
			}
			i++
			o.payload[key] = args[i]
		default:
			positionals = append(positionals, args[i])
		}
	}
	if len(positionals) > 0 {
		o.operation = positionals[0]
	}
	if len(positionals) > 1 && positionals[0] == "item" {
		o.operation = positionals[1] + "_item"
		positionals = append([]string{positionals[1]}, positionals[2:]...)
	} else if len(positionals) > 1 && positionals[0] == "comment" {
		o.operation = positionals[1] + "_comment"
		positionals = append([]string{positionals[1]}, positionals[2:]...)
	}
	if alias, ok := map[string]string{"show": "show_intent", "validate": "validate_intent", "diff": "diff_revisions"}[o.operation]; ok {
		o.operation = alias
	}
	if alias, ok := map[string]string{"edit-preview_item": "preview_edit", "accept_item": "accept_item", "reject_item": "reject_item"}[o.operation]; ok {
		o.operation = alias
	}
	if alias, ok := map[string]string{"complete-review": "complete_review", "start-review": "start_review", "edit-preview": "preview_edit", "accept": "accept_item", "reject": "reject_item", "add": "add_comment", "resolve": "resolve_comment", "withdraw": "withdraw_comment", "approve": "approve_intent"}[o.operation]; ok {
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
	case "start_review", "complete_review":
		if len(positionals) != 2 {
			return o, fmt.Errorf("%s requires one Intent path", positionals[0])
		}
		o.payload["intent_path"] = positionals[1]
	case "review":
		if len(positionals) != 2 {
			return o, errors.New("review requires one Intent path")
		}
		o.interactive = true
		o.payload["intent_path"] = positionals[1]
	case "workspace":
		if len(positionals) != 2 {
			return o, errors.New("workspace requires one workspace path")
		}
		o.interactive = true
		o.payload["workspace_path"] = positionals[1]
	case "approve_intent":
		if len(positionals) != 2 {
			return o, errors.New("approve requires one Intent path")
		}
		o.interactive = true
		o.payload["intent_path"] = positionals[1]
	case "accept_item", "reject_item", "preview_edit", "edit_item":
		if len(positionals) != 3 {
			return o, fmt.Errorf("%s requires Intent path and item ID", positionals[0])
		}
		o.payload["intent_path"], o.payload["item_id"] = positionals[1], positionals[2]
	case "add_comment":
		if len(positionals) != 3 {
			return o, errors.New("comment add requires Intent path and item ID")
		}
		o.payload["intent_path"], o.payload["item_id"] = positionals[1], positionals[2]
	case "resolve_comment", "withdraw_comment":
		if len(positionals) != 3 {
			return o, fmt.Errorf("%s requires Intent path and comment ID", positionals[0])
		}
		o.payload["intent_path"], o.payload["comment_id"] = positionals[1], positionals[2]
	default:
		return o, fmt.Errorf("unknown command %q", o.operation)
	}
	if modelNeedsActor(o.operation) {
		actor, err := localActor(o.actorID)
		if err != nil {
			return o, err
		}
		o.payload["actor"] = actor
	}
	for source, target := range map[string]string{"statement_file": "statement", "body_file": "body"} {
		if value, ok := o.payload[source]; ok {
			path, ok := value.(string)
			if !ok {
				return o, fmt.Errorf("%s must be a path", source)
			}
			contents, err := os.ReadFile(path)
			if err != nil {
				return o, fmt.Errorf("read %s: %w", source, err)
			}
			o.payload[target] = string(contents)
			delete(o.payload, source)
		}
	}
	if value, ok := o.payload["reason_file"]; ok {
		path, ok := value.(string)
		if !ok {
			return o, errors.New("reason_file must be a path")
		}
		contents, err := os.ReadFile(path)
		if err != nil {
			return o, fmt.Errorf("read reason_file: %w", err)
		}
		target := "rationale"
		if o.operation == "resolve_comment" || o.operation == "withdraw_comment" {
			target = "reason"
		}
		o.payload[target] = string(contents)
		delete(o.payload, "reason_file")
	}
	o.payload["operation"] = o.operation
	return o, nil
}

func modelNeedsActor(operation string) bool {
	switch operation {
	case "protocol_info", "show_intent", "validate_intent", "diff_revisions", "prepare_approval", "workspace", "list_intents", "list_revisions", "inspect_revision", "inspect_snapshot", "recovery_status":
		return false
	default:
		return true
	}
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
	executable, _ := os.Executable()
	return findCoreFor(os.Getenv("ZINTENT_CORE"), executable)
}

func findCoreFor(explicit, executable string) string {
	if explicit != "" {
		return explicit
	}
	if executable != "" {
		dir := filepath.Dir(executable)
		candidates := []string{
			filepath.Join(dir, "zintent-core"),
			filepath.Clean(filepath.Join(dir, "..", "libexec", "zintent", "zintent-core")),
		}
		for _, candidate := range candidates {
			if info, err := os.Stat(candidate); err == nil && info.Mode().IsRegular() && info.Mode()&0o111 != 0 {
				return candidate
			}
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
	case "invalid_artifact", "integrity_failure", "unsupported_schema", "duplicate_id", "broken_reference":
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
