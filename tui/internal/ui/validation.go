package ui

import (
	"fmt"
	"strings"
)

type ValidationFinding struct{ Code, Severity, RecordType, RecordID, Path, Message string }
type ValidationScreen struct {
	Findings   []ValidationFinding
	Severity   string
	SelectedID string
}

func (s ValidationScreen) View() string {
	var b strings.Builder
	b.WriteString("Validation findings\n")
	for _, f := range s.Visible() {
		fmt.Fprintf(&b, "[%s] %s %s/%s\n%s\n", f.Severity, f.Code, f.RecordType, f.RecordID, f.Message)
	}
	return b.String()
}

func (s ValidationScreen) Visible() []ValidationFinding {
	out := make([]ValidationFinding, 0, len(s.Findings))
	for _, f := range s.Findings {
		if s.Severity == "" || f.Severity == s.Severity {
			out = append(out, f)
		}
	}
	return out
}
func (s ValidationScreen) JumpRecord() string {
	for _, f := range s.Visible() {
		if f.RecordID == s.SelectedID {
			return f.RecordID
		}
	}
	return ""
}
