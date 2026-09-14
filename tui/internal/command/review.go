package command

import (
	"encoding/json"
	"errors"

	"github.com/hib2018/zintent/tui/internal/protocol"
)

type ReviewFlow struct {
	Executor   *Executor
	IntentPath string
}

func (f ReviewFlow) MutateAndReload(mutation protocol.Request) (protocol.Response, error) {
	if f.Executor == nil {
		return protocol.Response{}, errors.New("executor required")
	}
	response, err := f.Executor.Execute(mutation, true)
	if err != nil || !response.OK {
		return response, err
	}
	body, _ := json.Marshal(map[string]any{"operation": "show_intent", "intent_path": f.IntentPath})
	reload := protocol.Request{ProtocolVersion: protocol.Version, RequestID: mutation.RequestID + "-reload", Operation: "show_intent", PayloadSchema: "zintent.command/1", Payload: body}
	canonical, err := f.Executor.Execute(reload, false)
	if err == nil && canonical.OK {
		f.Executor.MarkReloaded()
	}
	return canonical, err
}

func ReviewRequest(operation, requestID, intentPath, expectedRevision, operationID, targetKey, targetID string, actor protocol.Actor, fields map[string]any) (protocol.Request, error) {
	payload := map[string]any{"operation": operation, "intent_path": intentPath, "expected_revision_id": expectedRevision, "operation_id": operationID, "actor": actor}
	if targetKey != "" {
		payload[targetKey] = targetID
	}
	for k, v := range fields {
		payload[k] = v
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return protocol.Request{}, err
	}
	return protocol.Request{ProtocolVersion: protocol.Version, RequestID: requestID, Operation: operation, PayloadSchema: "zintent.command/1", Payload: body}, nil
}

func PreviewEditRequest(requestID, intentPath, revision, itemID, statement string, actor protocol.Actor) (protocol.Request, error) {
	return ReviewRequest("preview_edit", requestID, intentPath, revision, "", "item_id", itemID, actor, map[string]any{"statement": statement})
}
func ApplyEditRequest(requestID, intentPath, revision, operationID, itemID, statement, token string, actor protocol.Actor) (protocol.Request, error) {
	if token == "" {
		return protocol.Request{}, errors.New("preview token required")
	}
	return ReviewRequest("edit_item", requestID, intentPath, revision, operationID, "item_id", itemID, actor, map[string]any{"statement": statement, "preview_token": token})
}
func CompletionRequest(requestID, intentPath, revision, operationID string, actor protocol.Actor) (protocol.Request, error) {
	return ReviewRequest("complete_review", requestID, intentPath, revision, operationID, "", "", actor, nil)
}
