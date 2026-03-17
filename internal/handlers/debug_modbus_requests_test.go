package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestParseModbusSerialPayload(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/debug/modbus/serial", strings.NewReader(`{"serial_port":"/dev/ttyUSB0","timeout_ms":1000}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	payload, ok := parseModbusSerialPayload(w, req)
	if !ok {
		t.Fatal("expected ok=true, got false")
	}
	if payload == nil || payload.SerialPort != "/dev/ttyUSB0" {
		t.Fatalf("payload = %#v", payload)
	}
}

func TestBuildModbusSerialConfig(t *testing.T) {
	config := buildModbusSerialConfig(&modbusSerialDebugRequest{
		BaudRate:  9600,
		DataBits:  8,
		Parity:    "N",
		StopBits:  1,
		TimeoutMs: 1200,
	})

	if config.BaudRate != 9600 || config.DataBits != 8 || config.Parity != "N" || config.StopBits != 1 {
		t.Fatalf("config = %#v", config)
	}
	if config.ReadTimeout != 1200*time.Millisecond {
		t.Fatalf("config.ReadTimeout = %v, want %v", config.ReadTimeout, 1200*time.Millisecond)
	}
}

func TestBuildModbusTCPConfig(t *testing.T) {
	config := buildModbusTCPConfig(&modbusTCPDebugRequest{TimeoutMs: 2300})
	if config.Timeout != 2300*time.Millisecond {
		t.Fatalf("config.Timeout = %v, want %v", config.Timeout, 2300*time.Millisecond)
	}
}

func TestBuildModbusTCPDebugRequest_Raw(t *testing.T) {
	req := &modbusTCPDebugRequest{RawRequest: "00 01 00 00 00 06 01 03 00 00 00 03"}

	frame, txID, err := buildModbusTCPDebugRequest(req)
	if err != nil {
		t.Fatalf("buildModbusTCPDebugRequest err = %v", err)
	}
	if txID != 1 {
		t.Fatalf("txID = %d, want 1", txID)
	}
	if len(frame) != 12 {
		t.Fatalf("len(frame) = %d, want 12", len(frame))
	}
}

func TestBuildModbusSerialDebugRequest_Structured(t *testing.T) {
	req := &modbusSerialDebugRequest{
		SlaveID:      1,
		FunctionCode: modbusFuncReadHoldingRegisters,
		Address:      0,
		Quantity:     2,
	}

	frame, expectLen, err := buildModbusSerialDebugRequest(req)
	if err != nil {
		t.Fatalf("buildModbusSerialDebugRequest err = %v", err)
	}
	if expectLen != 9 {
		t.Fatalf("expectLen = %d, want 9", expectLen)
	}
	if len(frame) != 8 {
		t.Fatalf("len(frame) = %d, want 8", len(frame))
	}
}
