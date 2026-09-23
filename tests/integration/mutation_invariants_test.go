package integration

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestCoreEnforcesMutationInvariantsAndReversibleRejection(t *testing.T) {
	root, _ := filepath.Abs(filepath.Join("..", ".."))
	core := filepath.Join(root, "zig-out", "bin", "zintent-core")
	if _, err := os.Stat(core); err != nil {
		t.Skip("build zintent-core before integration tests")
	}
	source, err := os.ReadFile(filepath.Join(root, "tests", "fixtures", "valid-draft.json"))
	if err != nil {
		t.Fatal(err)
	}
	var legacy map[string]any
	if err := json.Unmarshal(source, &legacy); err != nil {
		t.Fatal(err)
	}
	legacyItem := legacy["revision_payload"].(map[string]any)["items"].([]any)[0].(map[string]any)
	legacyItem["review_status"], legacyItem["included_in_approval"] = "accepted", false
	source, _ = json.Marshal(legacy)
	intent := filepath.Join(t.TempDir(), "intent.json")
	if err := os.WriteFile(intent, source, 0o600); err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		RevisionID string `json:"revision_id"`
		Payload    struct {
			Items []struct {
				ID string `json:"item_id"`
			} `json:"items"`
		} `json:"revision_payload"`
	}
	if err := json.Unmarshal(source, &fixture); err != nil {
		t.Fatal(err)
	}
	itemID := fixture.Payload.Items[0].ID
	actor := map[string]any{"actor_type": "human", "actor_id": "alice", "identity_source": "explicit_fallback", "authenticated": false}
	type response struct {
		OK    bool `json:"ok"`
		Error *struct {
			Code string `json:"code"`
		} `json:"error"`
		Result struct {
			AffectedIDs []string `json:"affected_ids"`
			Data        struct {
				Intent struct {
					RevisionID string `json:"revision_id"`
					Payload    struct {
						Lifecycle string `json:"lifecycle_state"`
						Items     []struct {
							Status    string  `json:"review_status"`
							Included  bool    `json:"included_in_approval"`
							Rationale *string `json:"rationale"`
						} `json:"items"`
						Comments []struct {
							Status           string  `json:"status"`
							ClosedRevisionID *string `json:"closed_revision_id"`
						} `json:"comments"`
					} `json:"revision_payload"`
				} `json:"intent"`
			} `json:"data"`
		} `json:"result"`
	}
	request := func(operation string, payload map[string]any) response {
		payload["operation"] = operation
		body, _ := json.Marshal(map[string]any{"protocol_version": "1.0", "request_id": operation, "operation": operation, "payload_schema": "zintent.command/1", "payload": payload})
		cmd := exec.Command(core)
		cmd.Stdin = bytes.NewReader(body)
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("core failed: %v", err)
		}
		var got response
		if err := json.Unmarshal(out, &got); err != nil {
			t.Fatalf("invalid core response: %v: %s", err, out)
		}
		return got
	}
	mutation := func(expected, operationID string) map[string]any {
		return map[string]any{"intent_path": intent, "expected_revision_id": expected, "operation_id": operationID, "actor": actor}
	}

	p := mutation(fixture.RevisionID, "invalid-draft-accept")
	p["item_id"] = itemID
	if got := request("accept_item", p); got.OK || got.Error == nil || got.Error.Code != "invalid_transition" {
		t.Fatalf("draft accept was not rejected: %+v", got)
	}

	started := request("start_review", mutation(fixture.RevisionID, "start"))
	if !started.OK || started.Result.Data.Intent.Payload.Lifecycle != "in_review" || !started.Result.Data.Intent.Payload.Items[0].Included {
		t.Fatalf("review did not start or normalize legacy inclusion: %+v", started)
	}

	p = mutation("start", "reject")
	p["item_id"], p["rationale"] = itemID, "out of scope"
	rejected := request("reject_item", p)
	item := rejected.Result.Data.Intent.Payload.Items[0]
	if !rejected.OK || item.Status != "rejected" || item.Included || item.Rationale == nil || *item.Rationale != "out of scope" {
		t.Fatalf("invalid rejection result: %+v", rejected)
	}
	if len(rejected.Result.AffectedIDs) != 1 || rejected.Result.AffectedIDs[0] != itemID {
		t.Fatalf("affected item was not reported: %+v", rejected.Result.AffectedIDs)
	}

	p = mutation("reject", "restore")
	p["item_id"] = itemID
	restored := request("accept_item", p)
	item = restored.Result.Data.Intent.Payload.Items[0]
	if !restored.OK || item.Status != "accepted" || !item.Included || item.Rationale != nil {
		t.Fatalf("rejection was not reversed: %+v", restored)
	}

	p = mutation("restore", "bad-comment")
	p["item_id"], p["body"] = "missing-item", "question"
	if got := request("add_comment", p); got.OK || got.Error == nil || got.Error.Code != "invalid_artifact" {
		t.Fatalf("broken comment target was accepted: %+v", got)
	}

	p = mutation("restore", "comment")
	p["item_id"], p["body"] = itemID, "question"
	commented := request("add_comment", p)
	if !commented.OK || len(commented.Result.Data.Intent.Payload.Comments) != 1 {
		t.Fatalf("comment was not added: %+v", commented)
	}

	p = mutation("comment", "missing-reason")
	p["comment_id"] = "comment"
	if got := request("resolve_comment", p); got.OK || got.Error == nil || got.Error.Code != "invalid_request" {
		t.Fatalf("comment closure without a reason was accepted: %+v", got)
	}
	p = mutation("comment", "resolve")
	p["comment_id"], p["reason"] = "comment", "answered"
	resolved := request("resolve_comment", p)
	comment := resolved.Result.Data.Intent.Payload.Comments[0]
	if !resolved.OK || comment.Status != "resolved" || comment.ClosedRevisionID == nil || *comment.ClosedRevisionID != "resolve" {
		t.Fatalf("comment closure was not recorded: %+v", resolved)
	}
	p = mutation("resolve", "close-again")
	p["comment_id"], p["reason"] = "comment", "again"
	if got := request("withdraw_comment", p); got.OK || got.Error == nil || got.Error.Code != "invalid_transition" {
		t.Fatalf("closed comment changed state: %+v", got)
	}

	completed := request("complete_review", mutation("resolve", "complete"))
	if !completed.OK || completed.Result.Data.Intent.Payload.Lifecycle != "review_complete" {
		t.Fatalf("review did not complete: %+v", completed)
	}
	if got := request("start_review", mutation("complete", "restart")); got.OK || got.Error == nil || got.Error.Code != "invalid_transition" {
		t.Fatalf("completed review restarted directly: %+v", got)
	}
}
