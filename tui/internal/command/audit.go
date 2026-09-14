// Package command contains presentation helpers for read-only audit commands.
package command

import (
	"encoding/json"
	"fmt"
)

// AuditResult is the stable subset consumed by human and JSON frontends.
type AuditResult struct {
	Operation string          `json:"operation"`
	Data      json.RawMessage `json:"data"`
	Findings  []string        `json:"findings"`
}

// FormatHuman keeps audit output useful without hiding the machine-readable
// response returned by the core process.
func FormatHuman(result AuditResult) string {
	if len(result.Findings) == 0 {
		return fmt.Sprintf("%s succeeded", result.Operation)
	}
	return fmt.Sprintf("%s succeeded (%d findings)", result.Operation, len(result.Findings))
}
