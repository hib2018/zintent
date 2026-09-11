package runner

import (
	"testing"

	"github.com/hib2018/zintent/tui/internal/protocol"
)

func TestLimitedBufferRejectsOverflow(t *testing.T) {
	b := limitedBuffer{limit: 3}
	if _, err := b.Write([]byte("four")); err == nil {
		t.Fatal("expected bounded output failure")
	}
}

func TestCoreRequiresExecutable(t *testing.T) {
	_, err := (Core{}).Run(t.Context(), protocol.Request{})
	if err == nil {
		t.Fatal("expected missing executable error")
	}
}
