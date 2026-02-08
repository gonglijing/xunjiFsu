package driver

import (
	"io"
	"net"
	"testing"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/models"
)

func TestResolveResourceDefaults(t *testing.T) {
	device := &models.Device{}
	resourceID, resourceType := resolveResource(device)
	if resourceID != 0 {
		t.Fatalf("expected resourceID 0, got %d", resourceID)
	}
	if resourceType != "serial" {
		t.Fatalf("expected resourceType serial, got %s", resourceType)
	}
}

func TestResolveResourceDriverTypeFallback(t *testing.T) {
	id := int64(9)
	device := &models.Device{
		ResourceID: &id,
		DriverType: "net",
	}
	resourceID, resourceType := resolveResource(device)
	if resourceID != id {
		t.Fatalf("expected resourceID %d, got %d", id, resourceID)
	}
	if resourceType != "net" {
		t.Fatalf("expected resourceType net, got %s", resourceType)
	}
}

func TestResolveResourceExplicitType(t *testing.T) {
	device := &models.Device{
		ResourceType: "serial",
		DriverType:   "net",
	}
	_, resourceType := resolveResource(device)
	if resourceType != "serial" {
		t.Fatalf("expected resourceType serial, got %s", resourceType)
	}
}

func TestBuildDeviceConfigModbusRTU(t *testing.T) {
	device := &models.Device{
		DriverType:    "modbus_rtu",
		SerialPort:    "/dev/ttyUSB0",
		BaudRate:      9600,
		DataBits:      8,
		StopBits:      1,
		Parity:        "N",
		DeviceAddress: "1",
	}
	config := buildDeviceConfig(device)
	assertMapEqual(t, config, map[string]string{
		"serial_port":    "/dev/ttyUSB0",
		"baud_rate":      "9600",
		"data_bits":      "8",
		"stop_bits":      "1",
		"parity":         "N",
		"device_address": "1",
		"func_name":      "read",
	})
}

func TestBuildDeviceConfigTCP(t *testing.T) {
	device := &models.Device{
		DriverType: "modbus_tcp",
		IPAddress:  "127.0.0.1",
		PortNum:    502,
	}
	config := buildDeviceConfig(device)
	assertMapEqual(t, config, map[string]string{
		"ip_address": "127.0.0.1",
		"port_num":   "502",
		"func_name":  "read",
	})
	if _, ok := config["serial_port"]; ok {
		t.Fatalf("did not expect serial_port in tcp config")
	}
}

func assertMapEqual(t *testing.T, got, want map[string]string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("expected map size %d, got %d", len(want), len(got))
	}
	for key, wantVal := range want {
		gotVal, ok := got[key]
		if !ok {
			t.Fatalf("missing key %s", key)
		}
		if gotVal != wantVal {
			t.Fatalf("key %s expected %s got %s", key, wantVal, gotVal)
		}
	}
}

type fakeSerialPort struct {
	closed bool
}

func (p *fakeSerialPort) Write(_ []byte) (int, error) {
	return 0, nil
}

func (p *fakeSerialPort) Read(_ []byte) (int, error) {
	return 0, io.EOF
}

func (p *fakeSerialPort) Close() error {
	p.closed = true
	return nil
}

func TestStartExecution(t *testing.T) {
	manager := NewDriverManager()
	executor := NewDriverExecutor(manager)
	device := &models.Device{ID: 1, Name: "dev1"}

	done, err := executor.startExecution(device)
	if err != nil {
		t.Fatalf("startExecution error: %v", err)
	}

	if _, err := executor.startExecution(device); err == nil {
		t.Fatalf("expected concurrent startExecution to fail")
	}

	done()

	if done2, err := executor.startExecution(device); err != nil {
		t.Fatalf("expected startExecution after done to succeed: %v", err)
	} else {
		done2()
	}
}

func TestUnregisterSerialPortCloses(t *testing.T) {
	manager := NewDriverManager()
	executor := NewDriverExecutor(manager)
	port := &fakeSerialPort{}

	executor.RegisterSerialPort(1, port)
	executor.UnregisterSerialPort(1)

	if !port.closed {
		t.Fatalf("expected serial port to be closed on unregister")
	}
}

func TestSetResourcePathClosesTCPOnChange(t *testing.T) {
	manager := NewDriverManager()
	executor := NewDriverExecutor(manager)

	c1, c2 := net.Pipe()
	defer c2.Close()

	executor.RegisterTCP(1, c1)
	executor.SetResourcePath(1, "127.0.0.1:502")
	executor.SetResourcePath(1, "127.0.0.1:503")

	if _, err := c1.Write([]byte("x")); err == nil {
		t.Fatalf("expected closed conn after path change")
	}

	if _, err := c2.Write([]byte("x")); err == nil {
		t.Fatalf("expected other end to be closed")
	}
}

func TestSetTimeoutsOverridesDefaults(t *testing.T) {
	manager := NewDriverManager()
	executor := NewDriverExecutor(manager)

	executor.SetTimeouts(2*time.Second, 3*time.Second, 4*time.Second)

	if got := executor.serialReadTimeout(); got != 2*time.Second {
		t.Fatalf("expected serial timeout 2s, got %v", got)
	}
	if got := executor.tcpDialTimeout(); got != 3*time.Second {
		t.Fatalf("expected tcp dial timeout 3s, got %v", got)
	}
	if got := executor.tcpReadTimeout(); got != 4*time.Second {
		t.Fatalf("expected tcp read timeout 4s, got %v", got)
	}
}

func TestSetRetriesOverridesDefaults(t *testing.T) {
	manager := NewDriverManager()
	executor := NewDriverExecutor(manager)

	if got := executor.serialOpenAttempts(); got != 1 {
		t.Fatalf("expected default serial attempts 1, got %d", got)
	}
	if got := executor.tcpDialAttempts(); got != 1 {
		t.Fatalf("expected default tcp attempts 1, got %d", got)
	}

	executor.SetRetries(2, 3, 150*time.Millisecond, 250*time.Millisecond)

	if got := executor.serialOpenAttempts(); got != 3 {
		t.Fatalf("expected serial attempts 3, got %d", got)
	}
	if got := executor.tcpDialAttempts(); got != 4 {
		t.Fatalf("expected tcp attempts 4, got %d", got)
	}
	if got := executor.serialOpenBackoff(); got != 150*time.Millisecond {
		t.Fatalf("expected serial backoff 150ms, got %v", got)
	}
	if got := executor.tcpDialBackoff(); got != 250*time.Millisecond {
		t.Fatalf("expected tcp backoff 250ms, got %v", got)
	}
}

func TestGetTCPConnReturnsRegisteredConnectionWithoutProbeWrite(t *testing.T) {
	manager := NewDriverManager()
	executor := NewDriverExecutor(manager)

	c1, c2 := net.Pipe()
	defer c1.Close()
	defer c2.Close()

	executor.RegisterTCP(12, c1)

	got := executor.GetTCPConn(12)
	if got != c1 {
		t.Fatalf("expected existing registered connection")
	}

	readDone := make(chan struct{})
	go func() {
		buf := make([]byte, 1)
		_, _ = c2.Read(buf)
		close(readDone)
	}()

	select {
	case <-readDone:
		t.Fatalf("expected no unsolicited probe write on existing TCP connection")
	case <-time.After(50 * time.Millisecond):
	}
}
