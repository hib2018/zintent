package protocol

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

const Version = "1.0"
const MaxMessageBytes = 16 << 20

type Request struct {
	ProtocolVersion string          `json:"protocol_version"`
	RequestID       string          `json:"request_id"`
	Operation       string          `json:"operation"`
	PayloadSchema   string          `json:"payload_schema"`
	Payload         json.RawMessage `json:"payload"`
}

type Error struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Retryable bool   `json:"retryable"`
}

type Response struct {
	ProtocolVersion string          `json:"protocol_version"`
	RequestID       string          `json:"request_id"`
	OK              bool            `json:"ok"`
	ResultSchema    string          `json:"result_schema,omitempty"`
	Result          json.RawMessage `json:"result,omitempty"`
	Error           *Error          `json:"error,omitempty"`
}

func DecodeStrict[T any](data []byte, target *T) error {
	if len(data) > MaxMessageBytes {
		return errors.New("message exceeds 16 MiB")
	}
	if err := rejectDuplicateNames(data); err != nil {
		return err
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(target); err != nil {
		return err
	}
	if dec.More() {
		return errors.New("extra JSON value")
	}
	return nil
}

func rejectDuplicateNames(data []byte) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	var walk func() error
	walk = func() error {
		tok, err := dec.Token()
		if err != nil {
			return err
		}
		delim, ok := tok.(json.Delim)
		if !ok {
			return nil
		}
		switch delim {
		case '{':
			seen := map[string]struct{}{}
			for dec.More() {
				name, err := dec.Token()
				if err != nil {
					return err
				}
				key, ok := name.(string)
				if !ok {
					return errors.New("object key is not a string")
				}
				if _, exists := seen[key]; exists {
					return fmt.Errorf("duplicate JSON field %q", key)
				}
				seen[key] = struct{}{}
				if err := walk(); err != nil {
					return err
				}
			}
			_, err = dec.Token()
			return err
		case '[':
			for dec.More() {
				if err := walk(); err != nil {
					return err
				}
			}
			_, err = dec.Token()
			return err
		default:
			return errors.New("unexpected JSON delimiter")
		}
	}
	if err := walk(); err != nil {
		return err
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return errors.New("extra JSON value")
	}
	return nil
}

func ValidateResponse(request Request, response Response) error {
	if response.ProtocolVersion != Version {
		return fmt.Errorf("unsupported protocol version %q", response.ProtocolVersion)
	}
	if response.RequestID != request.RequestID {
		return errors.New("response request_id mismatch")
	}
	if response.OK && response.Error != nil {
		return errors.New("successful response contains error")
	}
	if !response.OK && response.Error == nil {
		return errors.New("failed response has no error")
	}
	return nil
}
