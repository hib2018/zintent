package protocol

import "testing"

func TestDecodeStrictRejectsUnknownField(t *testing.T) {
	var r Request
	err := DecodeStrict([]byte(`{"protocol_version":"1.0","request_id":"r","operation":"protocol_info","payload_schema":"zintent.command/1","payload":{},"extra":true}`), &r)
	if err == nil {
		t.Fatal("expected unknown field rejection")
	}
}

func TestValidateResponseCorrelatesRequest(t *testing.T) {
	req := Request{RequestID: "a"}
	res := Response{ProtocolVersion: Version, RequestID: "b", OK: true}
	if ValidateResponse(req, res) == nil {
		t.Fatal("expected request ID mismatch")
	}
}

func TestDecodeStrictRejectsDuplicateNestedField(t *testing.T) {
	var r Request
	err := DecodeStrict([]byte(`{"protocol_version":"1.0","request_id":"r","operation":"protocol_info","payload_schema":"zintent.command/1","payload":{"operation":"protocol_info","operation":"protocol_info"}}`), &r)
	if err == nil {
		t.Fatal("expected duplicate field rejection")
	}
}
