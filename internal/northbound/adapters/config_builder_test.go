package adapters

import (
	"encoding/json"
	"testing"

	"github.com/gonglijing/xunjiFsu/internal/models"
	"github.com/gonglijing/xunjiFsu/internal/northbound/nbtype"
)

func TestBuildConfigFromModel_XunjiUsesSharedFieldMapping(t *testing.T) {
	cfg := &models.NorthboundConfig{
		Type:           nbtype.TypeXunji,
		ServerURL:      "broker.example.com",
		Port:           1883,
		Username:       "user",
		Password:       "pass",
		ClientID:       "client-1",
		Topic:          "v1/gateway/demo",
		AlarmTopic:     "v1/gateway/alarm",
		QOS:            1,
		Retain:         true,
		KeepAlive:      45,
		Timeout:        12,
		UploadInterval: 9000,
		ExtConfig:      `{"custom":"value"}`,
	}

	decoded := decodeConfigJSON(t, BuildConfigFromModel(cfg))

	assertConfigValue(t, decoded, "broker", "tcp://broker.example.com:1883")
	assertConfigValue(t, decoded, "serverUrl", "tcp://broker.example.com:1883")
	assertConfigValue(t, decoded, "username", "user")
	assertConfigValue(t, decoded, "password", "pass")
	assertConfigValue(t, decoded, "client_id", "client-1")
	assertConfigValue(t, decoded, "topic", "v1/gateway/demo")
	assertConfigValue(t, decoded, "alarm_topic", "v1/gateway/alarm")
	assertConfigValue(t, decoded, "qos", float64(1))
	assertConfigValue(t, decoded, "retain", true)
	assertConfigValue(t, decoded, "keepAlive", float64(45))
	assertConfigValue(t, decoded, "connectTimeout", float64(12))
	assertConfigValue(t, decoded, "uploadIntervalMs", float64(9000))
	assertConfigValue(t, decoded, "custom", "value")
}

func TestBuildConfigFromModel_IThingsIncludesIdentityFields(t *testing.T) {
	cfg := &models.NorthboundConfig{
		Type:           nbtype.TypeIThings,
		ServerURL:      "ssl://ithings.example.com",
		Port:           8883,
		Username:       "gateway-user",
		Password:       "gateway-pass",
		ClientID:       "gateway-client",
		ProductKey:     "product-a",
		DeviceKey:      "device-a",
		QOS:            2,
		KeepAlive:      30,
		Timeout:        8,
		UploadInterval: 6000,
	}

	decoded := decodeConfigJSON(t, BuildConfigFromModel(cfg))

	assertConfigValue(t, decoded, "broker", "ssl://ithings.example.com:8883")
	assertConfigValue(t, decoded, "serverUrl", "ssl://ithings.example.com:8883")
	assertConfigValue(t, decoded, "productKey", "product-a")
	assertConfigValue(t, decoded, "deviceKey", "device-a")
	assertConfigValue(t, decoded, "gatewayMode", true)
	assertConfigValue(t, decoded, "uploadIntervalMs", float64(6000))
}

func TestValidateConfig_ServerURLFallsBackToBroker(t *testing.T) {
	tests := []struct {
		name string
		typ  string
		cfg  map[string]interface{}
	}{
		{
			name: nbtype.TypeXunji,
			typ:  nbtype.TypeXunji,
			cfg:  map[string]interface{}{"broker": "tcp://xunji.example.com:1883"},
		},
		{
			name: nbtype.TypePandaX,
			typ:  nbtype.TypePandaX,
			cfg:  map[string]interface{}{"broker": "tcp://pandax.example.com:1883", "username": "token"},
		},
		{
			name: nbtype.TypeIThings,
			typ:  nbtype.TypeIThings,
			cfg: map[string]interface{}{
				"broker":     "tcp://ithings.example.com:1883",
				"username":   "user",
				"productKey": "pk",
				"deviceKey":  "dk",
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			if err := ValidateConfig(testCase.typ, testCase.cfg); err != nil {
				t.Fatalf("ValidateConfig() error = %v", err)
			}
		})
	}
}

func TestValidateConfig_PandaXGatewayModeMustBeTrue(t *testing.T) {
	cfg := map[string]interface{}{
		"serverUrl":   "tcp://pandax.example.com:1883",
		"username":    "token",
		"gatewayMode": "0",
	}

	if err := ValidateConfig(nbtype.TypePandaX, cfg); err == nil {
		t.Fatal("ValidateConfig() expected error")
	}
}

func decodeConfigJSON(t *testing.T, raw string) map[string]interface{} {
	t.Helper()

	decoded := make(map[string]interface{})
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	return decoded
}

func assertConfigValue(t *testing.T, decoded map[string]interface{}, key string, want interface{}) {
	t.Helper()

	if got := decoded[key]; got != want {
		t.Fatalf("config[%q] = %#v, want %#v", key, got, want)
	}
}
