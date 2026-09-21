package ui

import (
	"encoding/json"
	"testing"

	"github.com/hib2018/zintent/tui/internal/protocol"
)

const canonicalIntentResult = `{
  "data": {
    "intent": {
      "revision_id": "revision-1",
      "revision_hash": "hash-1",
      "revision_payload": {
        "intent_id": "intent-1",
        "lifecycle_state": "review_complete",
        "items": [
          {"item_id":"item-1","kind":"goal","statement":"確認済み","review_status":"accepted","provenance":{"content_origin":"human","operation_type":"accept_item"},"rationale":null}
        ],
        "comments": []
      }
    }
  }
}`

func TestDecodeWorkspaceIntentReadsCanonicalPayloadLifecycle(t *testing.T) {
	msg := decodeWorkspaceIntent(json.RawMessage(canonicalIntentResult), "/tmp/intents/intent-1")
	if msg.Err != nil {
		t.Fatal(msg.Err)
	}
	if msg.IntentID != "intent-1" || msg.RevisionID != "revision-1" || msg.Lifecycle != "review_complete" {
		t.Fatalf("canonical identity decoded from wrong level: %#v", msg)
	}
	if len(msg.Items) != 1 || msg.Items[0].Statement != "確認済み" {
		t.Fatalf("items=%#v", msg.Items)
	}
}

func TestReviewReloadReadsLifecycleFromRevisionPayload(t *testing.T) {
	m := New([]Item{{ID: "item-1", Status: "accepted"}})
	m.Lifecycle = "in_review"
	next, _ := m.Update(ReloadResultMsg{Response: protocol.Response{OK: true, Result: json.RawMessage(canonicalIntentResult)}})
	got := next.(Model)
	if got.Lifecycle != "review_complete" || got.Revision != "revision-1" {
		t.Fatalf("reload used noncanonical lifecycle: lifecycle=%q revision=%q", got.Lifecycle, got.Revision)
	}
}
