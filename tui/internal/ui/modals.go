package ui

import "time"

type ModalKind uint8

const (
	ModalEdit ModalKind = iota + 1
	ModalComment
	ModalReject
	ModalClosure
	ModalCompletion
	ModalApproval
)

type ModalState struct {
	Kind                               ModalKind
	Phase                              ModalPhase
	TargetID, StartingRevisionID, Text string
	CapabilityToken                    string
	ExpiresAt                          time.Time
	RequestID                          string
	Before, After                      string
}

func NewModal(kind ModalKind, targetID, revisionID string) ModalState {
	return ModalState{Kind: kind, Phase: ModalEditing, TargetID: targetID, StartingRevisionID: revisionID}
}
func (m *ModalState) Preview(requestID string) bool {
	if m.Phase != ModalEditing || requestID == "" {
		return false
	}
	m.Phase = ModalPreviewLoading
	m.RequestID = requestID
	return true
}
func (m *ModalState) Confirm(token string, expires time.Time) bool {
	if m.Phase != ModalPreviewLoading || token == "" {
		return false
	}
	m.CapabilityToken = token
	m.ExpiresAt = expires
	m.Phase = ModalConfirming
	return true
}
func (m *ModalState) Submit(requestID string, now time.Time) bool {
	if m.Phase != ModalConfirming || requestID == "" || (!m.ExpiresAt.IsZero() && !now.Before(m.ExpiresAt)) {
		return false
	}
	m.RequestID = requestID
	m.Phase = ModalSubmitting
	return true
}
func (m *ModalState) Reload() bool {
	if m.Phase != ModalSubmitting {
		return false
	}
	m.Phase = ModalReloadLoading
	return true
}
func (m *ModalState) Complete() bool {
	if m.Phase != ModalReloadLoading {
		return false
	}
	m.Clear()
	return true
}
func (m *ModalState) Fail()  { m.Phase = ModalError; m.CapabilityToken = "" }
func (m *ModalState) Clear() { *m = ModalState{Phase: ModalClosed} }
