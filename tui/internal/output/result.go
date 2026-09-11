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
		_, err := fmt.Fprintf(w, "%s succeeded\n", response.ResultSchema)
		return err
	}
	_, err := fmt.Fprintf(w, "%s: %s\n", response.Error.Code, response.Error.Message)
	return err
}
