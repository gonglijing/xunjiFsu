package handlers

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestParseRawModbusRTUBytes_CRCPlaceholders(t *testing.T) {
	frame, err := parseRawModbusRTUBytes("01 03 00 00 00 03 CRCA CRCB")
	if err != nil {
		t.Fatalf("parseRawModbusRTUBytes err = %v", err)
	}
	if len(frame) != 8 {
		t.Fatalf("len(frame) = %d, want 8", len(frame))
	}

	payload := []byte{0x01, 0x03, 0x00, 0x00, 0x00, 0x03}
	crc := crc16Modbus(payload)
	want := append(payload, byte(crc&0xFF), byte(crc>>8))
	if !bytes.Equal(frame, want) {
		t.Fatalf("frame = % X, want % X", frame, want)
	}
}

func TestParseRawModbusRTUBytes_WithCRCKeyword(t *testing.T) {
	frame, err := parseRawModbusRTUBytes("01 03 00 00 00 03 CRC")
	if err != nil {
		t.Fatalf("parseRawModbusRTUBytes err = %v", err)
	}
	if len(frame) != 8 {
		t.Fatalf("len(frame) = %d, want 8", len(frame))
	}

	payload := []byte{0x01, 0x03, 0x00, 0x00, 0x00, 0x03}
	crc := crc16Modbus(payload)
	if frame[6] != byte(crc&0xFF) || frame[7] != byte(crc>>8) {
		t.Fatalf("crc bytes = % X, want %02X %02X", frame[6:8], byte(crc&0xFF), byte(crc>>8))
	}
}

func TestParseRawModbusRTUBytes_InvalidCRCPlaceholder(t *testing.T) {
	_, err := parseRawModbusRTUBytes("01 03 00 00 00 03 CRCA")
	if err == nil {
		t.Fatalf("expected error for incomplete crc placeholder")
	}
}

func TestParseRawModbusRTURawResponse_ReadRegisters(t *testing.T) {
	request := []byte{0x01, 0x03, 0x00, 0x00, 0x00, 0x03}
	requestCRC := crc16Modbus(request)
	request = append(request, byte(requestCRC&0xFF), byte(requestCRC>>8))

	response := []byte{0x01, 0x03, 0x06, 0x00, 0x11, 0x00, 0x22, 0x00, 0x33}
	crc := crc16Modbus(response)
	response = append(response, byte(crc&0xFF), byte(crc>>8))

	parsed, err := parseModbusRTURawResponse("/dev/ttyUSB0", request, response)
	if err != nil {
		t.Fatalf("parseModbusRTURawResponse err = %v", err)
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

func TestParseRawModbusRTURawResponse_CRCMismatch(t *testing.T) {
	request := []byte{0x01, 0x03, 0x00, 0x00, 0x00, 0x01, 0x84, 0x0A}
	response := []byte{0x01, 0x03, 0x02, 0x00, 0x64, 0x00, 0x00}

	_, err := parseModbusRTURawResponse("/dev/ttyUSB0", request, response)
	if err == nil {
		t.Fatalf("expected crc mismatch error")
	}
}

func TestParseRawModbusRTUBytes_DecimalTokens(t *testing.T) {
	frame, err := parseRawModbusRTUBytes("1 3 0 0 0 3 crca crcb")
	if err != nil {
		t.Fatalf("parseRawModbusRTUBytes err = %v", err)
	}
	if len(frame) != 8 {
		t.Fatalf("len(frame) = %d, want 8", len(frame))
	}
	crc := binary.LittleEndian.Uint16(frame[6:8])
	want := crc16Modbus(frame[:6])
	if crc != want {
		t.Fatalf("crc = 0x%04X, want 0x%04X", crc, want)
	}
}
