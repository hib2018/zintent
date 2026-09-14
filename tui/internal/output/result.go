package output

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/hib2018/zintent/tui/internal/protocol"
)

func Write(w io.Writer, response protocol.Response, jsonMode bool) error {
	if jsonMode {
		enc := json.NewEncoder(w)
		enc.SetEscapeHTML(false)
		return enc.Encode(response)
	}
	if response.OK {
		var result struct {
			Operation string          `json:"operation"`
			Data      json.RawMessage `json:"data"`
		}
		_ = json.Unmarshal(response.Result, &result)
		label := result.Operation
		if label == "" {
			label = response.ResultSchema
		}
		_, err := fmt.Fprintf(w, "%s succeeded\n", label)
		if err != nil {
			return err
		}
		if len(result.Data) > 0 && label == "diff_revisions" {
			var diff struct {
				Changes []struct {
					RecordID string `json:"record_id"`
				} `json:"changes"`
			}
			if json.Unmarshal(result.Data, &diff) == nil {
				_, err = fmt.Fprintf(w, "changes: %d\n", len(diff.Changes))
			}
		}
		return err
	}
	_, err := fmt.Fprintf(w, "%s: %s\n", response.Error.Code, response.Error.Message)
	return err
}
