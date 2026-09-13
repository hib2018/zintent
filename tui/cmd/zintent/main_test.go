package main

import (
	"os"
	"path/filepath"
	"testing"
)

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
