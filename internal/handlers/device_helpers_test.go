package handlers

import (
	"strings"
	"testing"

	"github.com/gonglijing/xunjiFsu/internal/models"
)

func TestNormalizeWriteParams_ExplicitFieldAndValue(t *testing.T) {
	config := map[string]string{
		"field_name": "temperature",
		"value":      "26.5",
	}
	params := map[string]interface{}{
		"temperature": 26.5,
	}

	if err := normalizeWriteParams(config, params); err != nil {
		t.Fatalf("normalizeWriteParams returned error: %v", err)
	}

	if got := config["field_name"]; got != "temperature" {
		t.Fatalf("field_name = %q, want %q", got, "temperature")
	}
	if got := config["value"]; got != "26.5" {
		t.Fatalf("value = %q, want %q", got, "26.5")
	}
}

func TestNormalizeWriteParams_FromSingleParam(t *testing.T) {
	config := map[string]string{}
	params := map[string]interface{}{
		"setpoint": 30,
	}

	if err := normalizeWriteParams(config, params); err != nil {
		t.Fatalf("normalizeWriteParams returned error: %v", err)
	}

	if got := config["field_name"]; got != "setpoint" {
		t.Fatalf("field_name = %q, want %q", got, "setpoint")
	}
	if got := config["value"]; got != "30" {
		t.Fatalf("value = %q, want %q", got, "30")
	}
}

func TestNormalizeWriteParams_FromSubDeviceProperties(t *testing.T) {
	config := map[string]string{}
	params := map[string]interface{}{
		"subDevice": map[string]interface{}{
			"identity": map[string]interface{}{
				"productKey": "pk-1",
				"deviceKey":  "dk-1",
			},
			"properties": map[string]interface{}{
				"humidity": 55.2,
			},
		},
	}

	if err := normalizeWriteParams(config, params); err != nil {
		t.Fatalf("normalizeWriteParams returned error: %v", err)
	}

	if got := config["field_name"]; got != "humidity" {
		t.Fatalf("field_name = %q, want %q", got, "humidity")
	}
	if got := config["value"]; got != "55.2" {
		t.Fatalf("value = %q, want %q", got, "55.2")
	}
}

func TestNormalizeWriteParams_Ambiguous(t *testing.T) {
	config := map[string]string{}
	params := map[string]interface{}{
		"temperature": 25,
		"humidity":    60,
	}

	err := normalizeWriteParams(config, params)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("error = %q, want contains ambiguous", err.Error())
	}
}

func TestResolveWriteProperties_FromSubDeviceList(t *testing.T) {
	params := map[string]interface{}{
		"subDevices": []interface{}{
			map[string]interface{}{
				"properties": map[string]interface{}{
					"temperature": 25,
				},
			},
		},
	}

	properties, ok := resolveWriteProperties(params)
	if !ok {
		t.Fatal("resolveWriteProperties() ok=false, want=true")
	}
	if got := properties["temperature"]; got != 25 {
		t.Fatalf("temperature = %#v, want 25", got)
	}
}

func TestEnrichExecuteIdentity_UsesDeviceIdentity(t *testing.T) {
	config := map[string]string{}
	device := &models.Device{ProductKey: "dev-pk", DeviceKey: "dev-dk"}

	enrichExecuteIdentity(config, device)

	if got := config["product_key"]; got != "dev-pk" {
		t.Fatalf("product_key = %q, want %q", got, "dev-pk")
	}
	if got := config["productKey"]; got != "dev-pk" {
		t.Fatalf("productKey = %q, want %q", got, "dev-pk")
	}
	if got := config["device_key"]; got != "dev-dk" {
		t.Fatalf("device_key = %q, want %q", got, "dev-dk")
	}
	if got := config["deviceKey"]; got != "dev-dk" {
		t.Fatalf("deviceKey = %q, want %q", got, "dev-dk")
	}
}

func TestBuildExecuteDriverConfig_Write(t *testing.T) {
	params := map[string]interface{}{
		"setpoint": 42,
	}
	device := &models.Device{
		DeviceAddress: "1",
		ProductKey:    "pk",
		DeviceKey:     "dk",
	}

	config, err := buildExecuteDriverConfig(params, device, "write")
	if err != nil {
		t.Fatalf("buildExecuteDriverConfig returned error: %v", err)
	}

	if got := config["func_name"]; got != "write" {
		t.Fatalf("func_name = %q, want %q", got, "write")
	}
	if got := config["device_address"]; got != "1" {
		t.Fatalf("device_address = %q, want %q", got, "1")
	}
	if got := config["field_name"]; got != "setpoint" {
		t.Fatalf("field_name = %q, want %q", got, "setpoint")
	}
	if got := config["value"]; got != "42" {
		t.Fatalf("value = %q, want %q", got, "42")
	}
	if got := config["product_key"]; got != "pk" {
		t.Fatalf("product_key = %q, want %q", got, "pk")
	}
	if got := config["device_key"]; got != "dk" {
		t.Fatalf("device_key = %q, want %q", got, "dk")
	}
}

func TestBuildExecuteDriverConfig_WriteWithoutIdentityDoesNotFallbackGateway(t *testing.T) {
	params := map[string]interface{}{
		"setpoint": 42,
	}
	device := &models.Device{
		DeviceAddress: "1",
	}

	config, err := buildExecuteDriverConfig(params, device, "write")
	if err != nil {
		t.Fatalf("buildExecuteDriverConfig returned error: %v", err)
	}

	if got := config["product_key"]; got != "" {
		t.Fatalf("product_key = %q, want empty", got)
	}
	if got := config["device_key"]; got != "" {
		t.Fatalf("device_key = %q, want empty", got)
	}
}

func TestBuildExecuteDriverContext(t *testing.T) {
	resourceID := int64(9)
	device := &models.Device{
		ID:         7,
		Name:       "dev-a",
		ResourceID: &resourceID,
		DriverType: "modbus_tcp_wasm",
	}
	config := map[string]string{"func_name": "read"}

	ctx := buildExecuteDriverContext(device, config)
	if ctx == nil {
		t.Fatal("ctx is nil")
	}
	if ctx.DeviceID != 7 {
		t.Fatalf("DeviceID = %d, want %d", ctx.DeviceID, 7)
	}
	if ctx.DeviceName != "dev-a" {
		t.Fatalf("DeviceName = %q, want %q", ctx.DeviceName, "dev-a")
	}
	if ctx.ResourceID != 9 {
		t.Fatalf("ResourceID = %d, want %d", ctx.ResourceID, 9)
	}
	if ctx.ResourceType != "net" {
		t.Fatalf("ResourceType = %q, want %q", ctx.ResourceType, "net")
	}
	if ctx.Config["func_name"] != "read" {
		t.Fatalf("ctx config func_name = %q, want %q", ctx.Config["func_name"], "read")
	}
}

func TestBuildResourceMap(t *testing.T) {
	resources := []*models.Resource{
		nil,
		{ID: 1, Name: "serial-1"},
		{ID: 2, Name: "net-1"},
	}

	got := buildResourceMap(resources)

	if len(got) != 2 {
		t.Fatalf("len(got) = %d, want 2", len(got))
	}
	if got[1] == nil || got[1].Name != "serial-1" {
		t.Fatalf("got[1] = %#v", got[1])
	}
}

func TestResolveMapValue(t *testing.T) {
	if got, ok := resolveMapValue(map[string]interface{}{"k": "v"}); !ok || got["k"] != "v" {
		t.Fatalf("resolveMapValue() = %#v, %v; want map with k=v", got, ok)
	}
	if _, ok := resolveMapValue([]interface{}{"v"}); ok {
		t.Fatal("resolveMapValue() ok=true for slice input, want false")
	}
}

func TestEnrichDeviceDisplay(t *testing.T) {
	driverID := int64(7)
	resourceID := int64(3)
	device := &models.Device{
		DriverID:   &driverID,
		ResourceID: &resourceID,
	}

	enrichDeviceDisplay(device, map[int64]string{7: "demo-driver"}, map[int64]*models.Resource{
		3: {ID: 3, Name: "rs485", Type: "serial", Path: "/dev/ttyUSB0"},
	})

	if device.DriverName != "demo-driver" {
		t.Fatalf("device.DriverName = %q, want demo-driver", device.DriverName)
	}
	if device.ResourceName != "rs485" || device.ResourceType != "serial" || device.ResourcePath != "/dev/ttyUSB0" {
		t.Fatalf("unexpected resource display fields: %#v", device)
	}
}
