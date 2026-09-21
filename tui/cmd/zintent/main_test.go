package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDecodeReviewIntentReadsLifecycleFromRevisionPayload(t *testing.T) {
	raw := json.RawMessage(`{"data":{"intent":{"revision_id":"revision-1","revision_payload":{"intent_id":"intent-1","lifecycle_state":"review_complete","items":[{"item_id":"item-1","kind":"goal","statement":"確認済み","review_status":"accepted"}]}}}}`)
	intentID, revisionID, lifecycle, items, err := decodeReviewIntent(raw)
	if err != nil {
		t.Fatal(err)
	}
	if intentID != "intent-1" || revisionID != "revision-1" || lifecycle != "review_complete" || len(items) != 1 {
		t.Fatalf("decoded wrong canonical level: intent=%q revision=%q lifecycle=%q items=%#v", intentID, revisionID, lifecycle, items)
	}
}

func TestParseReadOnlyCommands(t *testing.T) {
	o, err := parseArgs([]string{"show", "intent", "--output", "json", "--core", "core"})
	if err != nil {
		t.Fatal(err)
	}
	if o.operation != "show_intent" || o.payload["intent_path"] != "intent" || !o.jsonMode {
		t.Fatalf("unexpected options: %#v", o)
	}
}

func TestActorAttribution(t *testing.T) {
	a, err := localActor("reviewer")
	if err != nil {
		t.Fatal(err)
	}
	if a["actor_type"] != "human" || a["authenticated"] != false {
		t.Fatalf("unexpected actor: %#v", a)
	}
}

func TestStableExitClasses(t *testing.T) {
	if errorExit("stale_revision") != 4 || errorExit("approval_ineligible") != 5 || errorExit("persistence_failure") != 6 {
		t.Fatal("wrong exit mapping")
	}
}

func TestParseMutationCommandSurface(t *testing.T) {
	o, err := parseArgs([]string{"item", "accept", "intent", "i-1", "--expected-revision", "r-1", "--operation-id", "op-1", "--actor-id", "alice", "--json"})
	if err != nil {
		t.Fatal(err)
	}
	if o.operation != "accept_item" || o.payload["intent_path"] != "intent" || o.payload["item_id"] != "i-1" {
		t.Fatalf("unexpected parsed mutation: %#v", o)
	}
	if o.payload["expected_revision_id"] != "r-1" || o.payload["operation_id"] != "op-1" {
		t.Fatalf("missing mutation guards: %#v", o.payload)
	}
}

func TestParseCommentCommandSurface(t *testing.T) {
	bodyPath := filepath.Join(t.TempDir(), "comment.txt")
	if err := os.WriteFile(bodyPath, []byte("comment"), 0o600); err != nil {
		t.Fatal(err)
	}
	o, err := parseArgs([]string{"comment", "add", "intent", "i-1", "--body-file", bodyPath, "--expected-revision", "r-1", "--operation-id", "op-1", "--actor-id", "alice"})
	if err != nil {
		t.Fatal(err)
	}
	if o.operation != "add_comment" || o.payload["comment_id"] != nil || o.payload["body"] != "comment" {
		t.Fatalf("unexpected comment command: %#v", o)
	}
}

func TestParseApprovalRequiresExactRevisionAndOperation(t *testing.T) {
	o, err := parseArgs([]string{"approve", "intent", "--revision", "r-1", "--operation-id", "op-1", "--actor-id", "alice"})
	if err != nil {
		t.Fatal(err)
	}
	if o.operation != "approve_intent" || !o.interactive || o.payload["expected_revision_id"] != "r-1" || o.payload["operation_id"] != "op-1" {
		t.Fatalf("unexpected approval options: %#v", o)
	}
}

func TestApprovalRefusesNonTTY(t *testing.T) {
	err := run([]string{"approve", "intent", "--revision", "r-1", "--operation-id", "op-1", "--actor-id", "alice"})
	if err == nil || err.Error() == "" {
		t.Fatal("expected tty refusal")
	}
}

func TestWorkspaceCommandAndNonTTYRefusal(t *testing.T) {
	o, err := parseArgs([]string{"workspace", "/tmp/intents"})
	if err != nil {
		t.Fatal(err)
	}
	if !o.interactive || o.operation != "workspace" || o.payload["workspace_path"] != "/tmp/intents" {
		t.Fatalf("unexpected workspace options: %#v", o)
	}
	if err := run([]string{"workspace", "/tmp/intents"}); err == nil || !strings.Contains(err.Error(), "tty_required") {
		t.Fatalf("expected TTY refusal, got %v", err)
	}
}

func TestFindCoreForPrefersExplicitOverride(t *testing.T) {
	if got := findCoreFor("/custom/zintent-core", "/prefix/bin/zintent"); got != "/custom/zintent-core" {
		t.Fatalf("got %q", got)
	}
}

func TestFindCoreForPrefersSiblingBinary(t *testing.T) {
	prefix := t.TempDir()
	frontend := filepath.Join(prefix, "bin", "zintent")
	sibling := filepath.Join(prefix, "bin", "zintent-core")
	libexec := filepath.Join(prefix, "libexec", "zintent", "zintent-core")
	writeExecutable(t, sibling)
	writeExecutable(t, libexec)

	if got := findCoreFor("", frontend); got != sibling {
		t.Fatalf("got %q, want sibling %q", got, sibling)
	}
}

func TestFindCoreForUsesLibexecLayout(t *testing.T) {
	prefix := t.TempDir()
	frontend := filepath.Join(prefix, "bin", "zintent")
	libexec := filepath.Join(prefix, "libexec", "zintent", "zintent-core")
	writeExecutable(t, libexec)

	if got := findCoreFor("", frontend); got != libexec {
		t.Fatalf("got %q, want libexec %q", got, libexec)
	}
}

func TestFindCoreForFallsBackForDevelopment(t *testing.T) {
	if got := findCoreFor("", filepath.Join(t.TempDir(), "bin", "zintent")); got != "../zig-out/bin/zintent-core" {
		t.Fatalf("got %q", got)
	}
}

func writeExecutable(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("test"), 0o755); err != nil {
		t.Fatal(err)
	}
}
