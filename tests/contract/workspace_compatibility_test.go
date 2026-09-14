package contract

import (
	"encoding/json"
	"testing"
)

type workspaceCommand struct {
	Operation     string `json:"operation"`
	WorkspacePath string `json:"workspace_path"`
}

func TestWorkspaceCommandFixtureCompatibility(t *testing.T) {
	data := []byte(`{"operation":"list_intents","workspace_path":"/tmp/workspace"}`)
	var command workspaceCommand
	if err := json.Unmarshal(data, &command); err != nil {
		t.Fatal(err)
	}
	if command.Operation != "list_intents" || command.WorkspacePath == "" {
		t.Fatalf("unexpected command: %+v", command)
	}
}
