package driver

import (
	"testing"

	"github.com/gonglijing/xunjiFsu/internal/models"
)

func TestResolveResourceDefaults(t *testing.T) {
	device := &models.Device{}
	resourceID, resourceType := resolveResource(device)
	if resourceID != 0 {
		t.Fatalf("expected resourceID 0, got %d", resourceID)
	}
	if resourceType != "modbus_rtu" {
		t.Fatalf("expected resourceType modbus_rtu, got %s", resourceType)
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
