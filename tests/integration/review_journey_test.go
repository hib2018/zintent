package integration

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestReviewJourneyAcceptAndReopenFromCurrentArtifact(t *testing.T) {
	root, _ := filepath.Abs(filepath.Join("..", ".."))
	core := filepath.Join(root, "zig-out", "bin", "zintent-core")
	if _, err := os.Stat(core); err != nil {
		t.Skip("build zintent-core before integration tests")
	}
	source, err := os.ReadFile(filepath.Join(root, "tests", "fixtures", "valid-draft.json"))
	if err != nil {
		t.Fatal(err)
	}
	intent := filepath.Join(t.TempDir(), "intent.json")
	if err := os.WriteFile(intent, source, 0o600); err != nil {
		t.Fatal(err)
	}
	var initial struct {
		Result struct {
			Data struct {
				Intent struct {
					RevisionID string `json:"revision_id"`
					Payload    struct {
						Items []struct {
							ID string `json:"item_id"`
						} `json:"items"`
					} `json:"revision_payload"`
				} `json:"intent"`
			} `json:"data"`
		} `json:"result"`
	}
	run := func(args ...string) []byte {
		cmd := exec.Command("go", append([]string{"run", "./tui/cmd/zintent"}, args...)...)
		cmd.Dir = root
		var out, errOut bytes.Buffer
		cmd.Stdout, cmd.Stderr = &out, &errOut
		if err := cmd.Run(); err != nil {
			t.Fatalf("CLI %v failed: %v (%s)", args, err, errOut.String())
		}
		return out.Bytes()
	}
	if err := json.Unmarshal(run("show", intent, "--output", "json", "--core", core), &initial); err != nil {
		t.Fatal(err)
	}
	if len(initial.Result.Data.Intent.Payload.Items) == 0 {
		t.Fatal("fixture has no item")
	}
	item := initial.Result.Data.Intent.Payload.Items[0].ID
	if err := json.Unmarshal(run("start-review", intent, "--expected-revision", initial.Result.Data.Intent.RevisionID, "--operation-id", "journey-start", "--actor-id", "journey", "--core", core, "--output", "json"), &initial); err != nil {
		t.Fatal(err)
	}
	work := t.TempDir()
	statement := filepath.Join(work, "statement.txt")
	reason := filepath.Join(work, "reason.txt")
	body := filepath.Join(work, "body.txt")
	for path, content := range map[string]string{statement: "Edited by journey", reason: "out of scope", body: "Please clarify"} {
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	previewBytes := run("item", "edit-preview", intent, item, "--statement-file", statement, "--expected-revision", initial.Result.Data.Intent.RevisionID, "--actor-id", "journey", "--core", core, "--output", "json")
	var preview struct {
		Result struct {
			Data struct {
				Preview struct {
					Token string `json:"preview_token"`
				} `json:"preview"`
			} `json:"data"`
		} `json:"result"`
	}
	if err := json.Unmarshal(previewBytes, &preview); err != nil || preview.Result.Data.Preview.Token == "" {
		t.Fatalf("invalid edit preview: %s", previewBytes)
	}
	editBytes := run("item", "edit", intent, item, "--statement-file", statement, "--expected-revision", initial.Result.Data.Intent.RevisionID, "--preview-token", preview.Result.Data.Preview.Token, "--operation-id", "journey-edit", "--actor-id", "journey", "--core", core, "--output", "json")
	var edited struct {
		Result struct {
			Data struct {
				Intent struct {
					RevisionID string `json:"revision_id"`
				} `json:"intent"`
			} `json:"data"`
		} `json:"result"`
	}
	if err := json.Unmarshal(editBytes, &edited); err != nil {
		t.Fatal(err)
	}
	commentBytes := run("comment", "add", intent, item, "--body-file", body, "--expected-revision", edited.Result.Data.Intent.RevisionID, "--operation-id", "journey-comment", "--actor-id", "journey", "--core", core, "--output", "json")
	var commented struct {
		Result struct {
			Data struct {
				Intent struct {
					RevisionID string `json:"revision_id"`
				} `json:"intent"`
			} `json:"data"`
		} `json:"result"`
	}
	if err := json.Unmarshal(commentBytes, &commented); err != nil {
		t.Fatal(err)
	}
	run("comment", "resolve", intent, "journey-comment", "--reason-file", reason, "--expected-revision", commented.Result.Data.Intent.RevisionID, "--operation-id", "journey-resolve", "--actor-id", "journey", "--core", core)
	var current struct {
		Result struct {
			Data struct {
				Intent struct {
					RevisionID string `json:"revision_id"`
				} `json:"intent"`
			} `json:"data"`
		} `json:"result"`
	}
	if err := json.Unmarshal(run("show", intent, "--output", "json", "--core", core), &current); err != nil {
		t.Fatal(err)
	}
	run("item", "reject", intent, item, "--reason-file", reason, "--expected-revision", current.Result.Data.Intent.RevisionID, "--operation-id", "journey-reject", "--actor-id", "journey", "--core", core)
	var reopened struct {
		Result struct {
			Data struct {
				Intent struct {
					RevisionID string `json:"revision_id"`
				} `json:"intent"`
			} `json:"data"`
		} `json:"result"`
	}
	if err := json.Unmarshal(run("show", intent, "--output", "json", "--core", core), &reopened); err != nil {
		t.Fatal(err)
	}
	if reopened.Result.Data.Intent.RevisionID != "journey-reject" {
		t.Fatalf("reopen did not expose published revision: %+v", reopened)
	}
	acceptedBytes := run("item", "accept", intent, item, "--expected-revision", "journey-reject", "--operation-id", "journey-restore", "--actor-id", "journey", "--core", core, "--output", "json")
	var restored struct {
		Result struct {
			Data struct {
				Intent struct {
					Payload struct {
						Items []struct {
							ID        string  `json:"item_id"`
							Status    string  `json:"review_status"`
							Included  bool    `json:"included_in_approval"`
							Rationale *string `json:"rationale"`
						} `json:"items"`
					} `json:"revision_payload"`
				} `json:"intent"`
			} `json:"data"`
		} `json:"result"`
	}
	if err := json.Unmarshal(acceptedBytes, &restored); err != nil {
		t.Fatal(err)
	}
	got := restored.Result.Data.Intent.Payload.Items[0]
	if got.ID != item || got.Status != "accepted" || !got.Included || got.Rationale != nil {
		t.Fatalf("reject was not reversed: %+v", got)
	}
}
