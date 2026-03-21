package httpapi

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
	if !ok || payload == nil || payload.SerialPort != "/dev/ttyUSB0" {
		t.Fatalf("payload = %#v, ok=%v", payload, ok)
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
	if config.ReadTimeout != 1200*time.Millisecond {
		t.Fatalf("config.ReadTimeout = %v", config.ReadTimeout)
	}
}

func TestBuildModbusTCPConfig(t *testing.T) {
	config := buildModbusTCPConfig(&modbusTCPDebugRequest{TimeoutMs: 2300})
	if config.Timeout != 2300*time.Millisecond {
		t.Fatalf("config.Timeout = %v", config.Timeout)
	}
}

func TestBuildModbusTCPDebugRequest_Raw(t *testing.T) {
	frame, txID, err := buildModbusTCPDebugRequest(&modbusTCPDebugRequest{RawRequest: "00 01 00 00 00 06 01 03 00 00 00 03"})
	if err != nil {
		t.Fatalf("buildModbusTCPDebugRequest err = %v", err)
	}
	if txID != 1 || len(frame) != 12 {
		t.Fatalf("txID=%d len=%d", txID, len(frame))
	}
}

func TestBuildModbusSerialDebugRequest_Structured(t *testing.T) {
	frame, expectLen, err := buildModbusSerialDebugRequest(&modbusSerialDebugRequest{
		SlaveID:      1,
		FunctionCode: modbusFuncReadHoldingRegisters,
		Address:      0,
		Quantity:     2,
	})
	if err != nil {
		t.Fatalf("buildModbusSerialDebugRequest err = %v", err)
	}
	if expectLen != 9 || len(frame) != 8 {
		t.Fatalf("expectLen=%d len=%d", expectLen, len(frame))
	}
}

func TestResolveModbusDirectOrResourceTarget_UsesDirectValue(t *testing.T) {
	got, err := resolveModbusDirectOrResourceTarget(" /dev/ttyUSB0 ", nil, "serial")
	if err != nil || got != "/dev/ttyUSB0" {
		t.Fatalf("got=%q err=%v", got, err)
	}
}

func TestResolveModbusTCPTarget_DirectEndpoint(t *testing.T) {
	endpoint, fromResource, err := resolveModbusTCPTarget(" 127.0.0.1:502 ", nil)
	if err != nil || fromResource || endpoint != "127.0.0.1:502" {
		t.Fatalf("endpoint=%q fromResource=%v err=%v", endpoint, fromResource, err)
	}
}

func TestValidateResolvedModbusTCPEndpoint(t *testing.T) {
	if err := validateResolvedModbusTCPEndpoint("127.0.0.1:502", false); err != nil {
		t.Fatalf("validateResolvedModbusTCPEndpoint error: %v", err)
	}
}

func TestValidateResolvedModbusTCPEndpoint_FromResource(t *testing.T) {
	err := validateResolvedModbusTCPEndpoint("bad-endpoint", true)
	if err == nil || !strings.Contains(err.Error(), "resource path is invalid") {
		t.Fatalf("err = %v", err)
	}
}

func TestNormalizeModbusTimeout(t *testing.T) {
	timeoutMs := 0
	if err := normalizeModbusTimeout(&timeoutMs, 800); err != nil || timeoutMs != 800 {
		t.Fatalf("timeoutMs=%d err=%v", timeoutMs, err)
	}
}

func TestValidateModbusSerialOperation_RawRequest(t *testing.T) {
	req := &modbusSerialDebugRequest{RawRequest: "01 03 00 00 00 01"}
	if err := validateModbusSerialOperation(req); err != nil || req.ExpectRespLen != 256 {
		t.Fatalf("err=%v expect=%d", err, req.ExpectRespLen)
	}
}
