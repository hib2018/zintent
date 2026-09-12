package main

import "testing"

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
