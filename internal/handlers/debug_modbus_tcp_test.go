package handlers

import "testing"

func TestBuildRawModbusTCPRequest(t *testing.T) {
	req := &modbusTCPDebugRequest{RawRequest: "00 01 00 00 00 06 01 03 00 00 00 03"}
	frame, txID, err := buildRawModbusTCPRequest(req)
	if err != nil {
		t.Fatalf("buildRawModbusTCPRequest err = %v", err)
	}
	if txID != 1 {
		t.Fatalf("txID = %d, want 1", txID)
	}
	if len(frame) != 12 {
		t.Fatalf("len(frame) = %d, want 12", len(frame))
	}
}

func TestValidateRawModbusTCPFrame_BadLength(t *testing.T) {
	_, err := validateRawModbusTCPFrame([]byte{0x00, 0x01, 0x00, 0x00, 0x00, 0x10, 0x01, 0x03})
	if err == nil {
		t.Fatalf("expected length mismatch error")
	}
}

func TestParseModbusTCPRawResponse_Read(t *testing.T) {
	request := []byte{0x00, 0x01, 0x00, 0x00, 0x00, 0x06, 0x01, 0x03, 0x00, 0x00, 0x00, 0x03}
	response := []byte{0x00, 0x01, 0x00, 0x00, 0x00, 0x09, 0x01, 0x03, 0x06, 0x00, 0x11, 0x00, 0x22, 0x00, 0x33}

	parsed, err := parseModbusTCPRawResponse("127.0.0.1:502", request, response, 1)
	if err != nil {
		t.Fatalf("parseModbusTCPRawResponse err = %v", err)
	}
	if parsed == nil {
		t.Fatalf("parsed is nil")
	}
	if parsed.FunctionCode != 0x03 {
		t.Fatalf("FunctionCode = %d, want 3", parsed.FunctionCode)
	}
	if parsed.Quantity == nil || *parsed.Quantity != 3 {
		t.Fatalf("Quantity = %v, want 3", parsed.Quantity)
	}
	if parsed.Address == nil || *parsed.Address != 0 {
		t.Fatalf("Address = %v, want 0", parsed.Address)
	}
	if len(parsed.Registers) != 3 || parsed.Registers[0] != 0x11 || parsed.Registers[2] != 0x33 {
		t.Fatalf("Registers = %v, want [17 34 51]", parsed.Registers)
	}
}

func TestParseModbusTCPRawResponse_Exception(t *testing.T) {
	request := []byte{0x00, 0x0A, 0x00, 0x00, 0x00, 0x06, 0x01, 0x03, 0x00, 0x00, 0x00, 0x01}
	response := []byte{0x00, 0x0A, 0x00, 0x00, 0x00, 0x03, 0x01, 0x83, 0x02}

	parsed, err := parseModbusTCPRawResponse("127.0.0.1:502", request, response, 10)
	if err != nil {
		t.Fatalf("parseModbusTCPRawResponse err = %v", err)
	}
	if parsed.ExceptionCode == nil || *parsed.ExceptionCode != 0x02 {
		t.Fatalf("ExceptionCode = %v, want 2", parsed.ExceptionCode)
	}
}
