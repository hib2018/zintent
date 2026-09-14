package command

import (
	"context"
	"encoding/json"
	"github.com/hib2018/zintent/tui/internal/protocol"
	"testing"
)

type sequenceRunner struct{ requests []protocol.Request }

func (s *sequenceRunner) Run(_ context.Context, r protocol.Request) (protocol.Response, error) {
	s.requests = append(s.requests, r)
	return protocol.Response{ProtocolVersion: protocol.Version, RequestID: r.RequestID, OK: true}, nil
}

func actorFixture() protocol.Actor {
	return protocol.Actor{Type: "human", ID: "alice", IdentitySource: "explicit_fallback"}
}
func payloadOf(t *testing.T, r protocol.Request) map[string]any {
	t.Helper()
	var p map[string]any
	if err := json.Unmarshal(r.Payload, &p); err != nil {
		t.Fatal(err)
	}
	return p
}
func TestReviewFlowExactDispatch(t *testing.T) {
	a := actorFixture()
	preview, _ := PreviewEditRequest("p", "intent", "r1", "i1", "after", a)
	p := payloadOf(t, preview)
	if p["statement"] != "after" {
		t.Fatal(p)
	}
	apply, _ := ApplyEditRequest("a", "intent", "r1", "op", "i1", "after", "exact-token", a)
	p = payloadOf(t, apply)
	if p["preview_token"] != "exact-token" {
		t.Fatal(p)
	}
	reject, _ := ReviewRequest("reject_item", "r", "intent", "r1", "op", "item_id", "i1", a, map[string]any{"rationale": "because"})
	if payloadOf(t, reject)["rationale"] != "because" {
		t.Fatal()
	}
}

func TestReviewMutationOperationsPreserveExactTargetsAndReasons(t *testing.T) {
	actor := actorFixture()
	cases := []struct {
		op, targetKey, targetID, field string
		value                          any
	}{{"accept_item", "item_id", "i1", "", nil}, {"reject_item", "item_id", "i2", "rationale", "out of scope"}, {"add_comment", "item_id", "i3", "body", "clarify"}, {"resolve_comment", "comment_id", "c1", "reason", "fixed"}, {"withdraw_comment", "comment_id", "c2", "reason", "obsolete"}}
	for _, tc := range cases {
		t.Run(tc.op, func(t *testing.T) {
			fields := map[string]any{}
			if tc.field != "" {
				fields[tc.field] = tc.value
			}
			r, err := ReviewRequest(tc.op, "request", "intent", "rev", "operation", tc.targetKey, tc.targetID, actor, fields)
			if err != nil {
				t.Fatal(err)
			}
			p := payloadOf(t, r)
			if p[tc.targetKey] != tc.targetID {
				t.Fatal(p)
			}
			if tc.field != "" && p[tc.field] != tc.value {
				t.Fatal(p)
			}
		})
	}
}
func TestApprovalCannotUseGenericMessage(t *testing.T) {
	a := actorFixture()
	if _, err := ApproveRequest("a", "i", "r", "o", "token", "", a, true); err == nil {
		t.Fatal("generic/empty response accepted")
	}
	r, _ := ApproveRequest("a", "i", "r", "o", "token", "challenge", a, true)
	if payloadOf(t, r)["challenge_response"] != "challenge" {
		t.Fatal()
	}
}

func TestEveryMutationIsFollowedByCanonicalReload(t *testing.T) {
	runner := &sequenceRunner{}
	executor := NewExecutor(context.Background(), runner)
	defer executor.Close()
	request, _ := ReviewRequest("accept_item", "r1", "intent", "rev", "op", "item_id", "i1", actorFixture(), nil)
	if _, err := (ReviewFlow{Executor: executor, IntentPath: "intent"}).MutateAndReload(request); err != nil {
		t.Fatal(err)
	}
	if len(runner.requests) != 2 || runner.requests[1].Operation != "show_intent" {
		t.Fatalf("requests=%+v", runner.requests)
	}
}
