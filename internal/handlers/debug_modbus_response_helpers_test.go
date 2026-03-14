package handlers

import "testing"

func TestParseModbusRegisterPayload(t *testing.T) {
	quantity, registers, err := parseModbusRegisterPayload([]byte{0x03, 0x04, 0x00, 0x11, 0x00, 0x22}, 1, 2)
	if err != nil {
		t.Fatalf("parseModbusRegisterPayload() error = %v", err)
	}
	if quantity == nil || *quantity != 2 {
		t.Fatalf("quantity = %v, want 2", quantity)
	}
	if len(registers) != 2 || registers[0] != 0x11 || registers[1] != 0x22 {
		t.Fatalf("registers = %v, want [17 34]", registers)
	}
}

func TestParseModbusRegisterPayload_InvalidByteCount(t *testing.T) {
	_, _, err := parseModbusRegisterPayload([]byte{0x03, 0x03, 0x00, 0x11, 0x00}, 1, 2)
	if err == nil {
		t.Fatal("parseModbusRegisterPayload() expected error")
	}
}

func TestParseModbusWritePayload(t *testing.T) {
	address, value, err := parseModbusWritePayload([]byte{0x06, 0x00, 0x10, 0x00, 0x64}, 1, 5)
	if err != nil {
		t.Fatalf("parseModbusWritePayload() error = %v", err)
	}
	if address == nil || *address != 0x10 {
		t.Fatalf("address = %v, want 16", address)
	}
	if value == nil || *value != 0x64 {
		t.Fatalf("value = %v, want 100", value)
	}
}
