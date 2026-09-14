package protocol

import "testing"

func FuzzDecodeStrictNeverPanics(f *testing.F) {
	f.Add([]byte(`{"protocol_version":"1.0","request_id":"r"}`))
	f.Add([]byte{0xff, 0xfe, 0x00})
	f.Fuzz(func(t *testing.T, data []byte) {
		var response Response
		_ = DecodeStrict(data, &response)
	})
}
