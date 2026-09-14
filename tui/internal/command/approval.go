package command

import (
	"errors"
	"github.com/hib2018/zintent/tui/internal/protocol"
)

func PrepareApprovalRequest(requestID, intentPath, revision string, actor protocol.Actor, interactiveTTY bool) (protocol.Request, error) {
	if !interactiveTTY {
		return protocol.Request{}, errors.New("terminal_required")
	}
	return ReviewRequest("prepare_approval", requestID, intentPath, revision, "", "", "", actor, map[string]any{"interactive_tty": true})
}
func ApproveRequest(requestID, intentPath, revision, operationID, token, response string, actor protocol.Actor, interactiveTTY bool) (protocol.Request, error) {
	if !interactiveTTY {
		return protocol.Request{}, errors.New("terminal_required")
	}
	if token == "" || response == "" {
		return protocol.Request{}, errors.New("fresh challenge response required")
	}
	return ReviewRequest("approve_intent", requestID, intentPath, revision, operationID, "", "", actor, map[string]any{"interactive_tty": true, "confirmation_token": token, "challenge_response": response})
}
