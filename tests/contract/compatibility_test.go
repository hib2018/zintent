package contract

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

type response struct {
	ProtocolVersion string          `json:"protocol_version"`
	RequestID       string          `json:"request_id"`
	OK              bool            `json:"ok"`
	ResultSchema    string          `json:"result_schema,omitempty"`
	Result          json.RawMessage `json:"result,omitempty"`
	Error           *struct {
		Code      string `json:"code"`
		Message   string `json:"message"`
		Retryable bool   `json:"retryable"`
	} `json:"error,omitempty"`
}

func TestCoreAcceptsAndRejectsSharedMessages(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	core := filepath.Join(root, "zig-out", "bin", "zintent-core")
	if _, err := os.Stat(core); err != nil {
		t.Fatalf("build core first: %v", err)
	}
	cases := []struct {
		file  string
		valid bool
	}{
		{"protocol-info.valid.json", true},
		{"edit-command.valid.json", true},
		{"unknown-field.invalid.json", false},
		{"operation-mismatch.invalid.json", false},
	}
	for _, tc := range cases {
		t.Run(tc.file, func(t *testing.T) {
			body, err := os.ReadFile(filepath.Join("fixtures", "messages", tc.file))
			if err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command(core)
			cmd.Stdin = bytes.NewReader(body)
			out, err := cmd.Output()
			if err != nil {
				t.Fatal(err)
			}
			if strings.Count(strings.TrimSpace(string(out)), "\n") != 0 {
				t.Fatal("core emitted extra stdout")
			}
			var got response
			dec := json.NewDecoder(bytes.NewReader(out))
			dec.DisallowUnknownFields()
			if err := dec.Decode(&got); err != nil {
				t.Fatal(err)
			}
			accepted := got.RequestID != ""
			if accepted != tc.valid {
				t.Fatalf("accepted=%v, want %v: %s", accepted, tc.valid, out)
			}
		})
	}
}
