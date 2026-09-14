package ui

import "time"

type ImportModal struct {
	Phase                                                        ModalPhase
	SourcePath, SourceHash, IntentID, Destination, Token, Status string
	Findings                                                     []string
	ExpiresAt                                                    time.Time
}

func (m ImportModal) CanSubmit(now time.Time) bool {
	return m.Phase == ModalConfirming && m.Token != "" && now.Before(m.ExpiresAt) && len(m.Findings) == 0
}
func (m *ImportModal) ResetFailure(status string) {
	*m = ImportModal{Phase: ModalError, Status: status}
}
func (m *ImportModal) Clear() { *m = ImportModal{Phase: ModalClosed} }
