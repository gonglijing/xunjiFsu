package handlers

import "testing"

func TestNormalizeModbusTimeout(t *testing.T) {
	timeoutMs := 0
	if err := normalizeModbusTimeout(&timeoutMs, 800); err != nil {
		t.Fatalf("normalizeModbusTimeout() error = %v", err)
	}
	if timeoutMs != 800 {
		t.Fatalf("timeoutMs = %d, want 800", timeoutMs)
	}

	timeoutMs = 60001
	if err := normalizeModbusTimeout(&timeoutMs, 800); err == nil {
		t.Fatal("normalizeModbusTimeout() expected error")
	}
}

func TestNormalizeModbusFunctionCode(t *testing.T) {
	functionCode := 0
	if err := normalizeModbusFunctionCode(&functionCode); err != nil {
		t.Fatalf("normalizeModbusFunctionCode() error = %v", err)
	}
	if functionCode != modbusFuncReadHoldingRegisters {
		t.Fatalf("functionCode = %d, want %d", functionCode, modbusFuncReadHoldingRegisters)
	}

	functionCode = 5
	if err := normalizeModbusFunctionCode(&functionCode); err == nil {
		t.Fatal("normalizeModbusFunctionCode() expected error")
	}
}

func TestNormalizeModbusOperation(t *testing.T) {
	quantity := 0
	value := 0
	if err := normalizeModbusOperation(modbusFuncReadHoldingRegisters, &quantity, &value); err != nil {
		t.Fatalf("normalizeModbusOperation() error = %v", err)
	}
	if quantity != 1 {
		t.Fatalf("quantity = %d, want 1", quantity)
	}

	quantity = 1
	value = 0x10000
	if err := normalizeModbusOperation(modbusFuncWriteSingleRegister, &quantity, &value); err == nil {
		t.Fatal("normalizeModbusOperation() expected error")
	}
}

func TestIsRawModbusRequest(t *testing.T) {
	if isRawModbusRequest("   ") {
		t.Fatal("isRawModbusRequest() = true, want false")
	}
	if !isRawModbusRequest("01 03 00 00") {
		t.Fatal("isRawModbusRequest() = false, want true")
	}
}

func TestValidateModbusStructuredRequest(t *testing.T) {
	functionCode := 0
	quantity := 0
	value := 0
	if err := validateModbusStructuredRequest(&functionCode, 1, 0, &quantity, &value); err != nil {
		t.Fatalf("validateModbusStructuredRequest() error = %v", err)
	}
	if functionCode != modbusFuncReadHoldingRegisters {
		t.Fatalf("functionCode = %d, want %d", functionCode, modbusFuncReadHoldingRegisters)
	}
	if quantity != 1 {
		t.Fatalf("quantity = %d, want 1", quantity)
	}
}

func TestResolveModbusDirectOrResourceTarget_UsesDirectValue(t *testing.T) {
	got, err := resolveModbusDirectOrResourceTarget(" /dev/ttyUSB0 ", nil, "serial")
	if err != nil {
		t.Fatalf("resolveModbusDirectOrResourceTarget() error = %v", err)
	}
	if got != "/dev/ttyUSB0" {
		t.Fatalf("target = %q, want %q", got, "/dev/ttyUSB0")
	}
}
