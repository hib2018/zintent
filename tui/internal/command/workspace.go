package command

import (
	"encoding/json"
	"errors"
	"github.com/hib2018/zintent/tui/internal/protocol"
)

func ListIntentsRequest(requestID, workspacePath string) (protocol.Request, error) {
	if workspacePath == "" {
		return protocol.Request{}, errors.New("workspace path required")
	}
	body, err := json.Marshal(map[string]any{"operation": "list_intents", "workspace_path": workspacePath})
	return protocol.Request{ProtocolVersion: protocol.Version, RequestID: requestID, Operation: "list_intents", PayloadSchema: "zintent.command/1", Payload: body}, err
}
func InspectDraftRequest(requestID, workspacePath, sourcePath string, actor protocol.Actor) (protocol.Request, error) {
	if sourcePath == "" {
		return protocol.Request{}, errors.New("source path required")
	}
	body, err := json.Marshal(map[string]any{"operation": "inspect_draft", "workspace_path": workspacePath, "source_path": sourcePath, "actor": actor})
	return protocol.Request{ProtocolVersion: protocol.Version, RequestID: requestID, Operation: "inspect_draft", PayloadSchema: "zintent.command/1", Payload: body}, err
}
func ImportDraftRequest(requestID, workspacePath, sourcePath, destination, token, operationID string, actor protocol.Actor) (protocol.Request, error) {
	if destination == "" || token == "" || operationID == "" {
		return protocol.Request{}, errors.New("exact import preview required")
	}
	body, err := json.Marshal(map[string]any{"operation": "import_draft", "workspace_path": workspacePath, "source_path": sourcePath, "destination": destination, "import_token": token, "operation_id": operationID, "actor": actor})
	return protocol.Request{ProtocolVersion: protocol.Version, RequestID: requestID, Operation: "import_draft", PayloadSchema: "zintent.command/1", Payload: body}, err
}
