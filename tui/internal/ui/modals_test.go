package ui

import (
	"testing"
	"time"
)

func TestModalStateMachines(t *testing.T) {
	for _, kind := range []ModalKind{ModalEdit, ModalComment, ModalReject, ModalClosure, ModalCompletion, ModalApproval} {
		t.Run(string(rune('0'+kind)), func(t *testing.T) {
			m := NewModal(kind, "target", "rev-1")
			if !m.Preview("preview") || !m.Confirm("token", time.Now().Add(time.Minute)) || !m.Submit("submit", time.Now()) || !m.Reload() || !m.Complete() {
				t.Fatalf("invalid transition: %+v", m)
			}
			if m.CapabilityToken != "" {
				t.Fatal("token not disposed")
			}
		})
	}
}
func TestModalCancelAndExpiry(t *testing.T) {
	m := NewModal(ModalEdit, "i", "r")
	m.Clear()
	if m.Phase != ModalClosed {
		t.Fatal()
	}
	m = NewModal(ModalApproval, "", "r")
	m.Preview("p")
	m.Confirm("t", time.Now().Add(-time.Second))
	if m.Submit("s", time.Now()) {
		t.Fatal("expired capability submitted")
	}
}
