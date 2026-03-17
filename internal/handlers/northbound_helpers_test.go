package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gonglijing/xunjiFsu/internal/models"
)

func TestNormalizeNorthboundConfig_AppliesDefaults(t *testing.T) {
	config := &models.NorthboundConfig{
		Name:           " demo ",
		Type:           " MQTT ",
		ServerURL:      " tcp://127.0.0.1:1883 ",
		UploadInterval: 0,
		Port:           0,
		QOS:            3,
		KeepAlive:      0,
		Timeout:        0,
	}

	normalizeNorthboundConfig(config)

	if config.Name != "demo" {
		t.Fatalf("config.Name = %q, want %q", config.Name, "demo")
	}
	if config.Type != "mqtt" {
		t.Fatalf("config.Type = %q, want %q", config.Type, "mqtt")
	}
	if config.ServerURL != "tcp://127.0.0.1:1883" {
		t.Fatalf("config.ServerURL = %q, want trimmed value", config.ServerURL)
	}
	if config.UploadInterval != defaultNorthboundUploadInterval {
		t.Fatalf("config.UploadInterval = %d, want %d", config.UploadInterval, defaultNorthboundUploadInterval)
	}
	if config.Port != defaultMQTTPort {
		t.Fatalf("config.Port = %d, want %d", config.Port, defaultMQTTPort)
	}
	if config.QOS != 0 {
		t.Fatalf("config.QOS = %d, want 0", config.QOS)
	}
	if config.KeepAlive != defaultNorthboundKeepAlive {
		t.Fatalf("config.KeepAlive = %d, want %d", config.KeepAlive, defaultNorthboundKeepAlive)
	}
	if config.Timeout != defaultNorthboundTimeout {
		t.Fatalf("config.Timeout = %d, want %d", config.Timeout, defaultNorthboundTimeout)
	}
}

func TestHasSchemaConfig(t *testing.T) {
	tests := []struct {
		name string
		cfg  *models.NorthboundConfig
		want bool
	}{
		{name: "nil", cfg: nil, want: false},
		{name: "empty", cfg: &models.NorthboundConfig{Config: ""}, want: false},
		{name: "empty object", cfg: &models.NorthboundConfig{Config: "{}"}, want: false},
		{name: "empty object with spaces", cfg: &models.NorthboundConfig{Config: "  {}  "}, want: false},
		{name: "non empty", cfg: &models.NorthboundConfig{Config: `{"server":"mqtt://a"}`}, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasSchemaConfig(tt.cfg); got != tt.want {
				t.Fatalf("hasSchemaConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateNorthboundConfig_RequiredFieldsFallback(t *testing.T) {
	config := &models.NorthboundConfig{
		Name: "demo",
		Type: "MQTT",
	}

	err := validateNorthboundConfig(config)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "server_url or config is required for MQTT type") {
		t.Fatalf("error = %q, want required server_url error", err.Error())
	}
}

func TestValidateNorthboundConfig_SchemaConfigBypassesLegacyRequiredFields(t *testing.T) {
	config := &models.NorthboundConfig{
		Name:   "demo",
		Type:   "mqtt",
		Config: `{"broker":"tcp://127.0.0.1:1883","topic":"demo"}`,
	}

	if err := validateNorthboundConfig(config); err != nil {
		t.Fatalf("validateNorthboundConfig returned error: %v", err)
	}
}

func TestValidateNorthboundConfig_RejectsUnknownType(t *testing.T) {
	invalidType := "legacy_type"
	config := &models.NorthboundConfig{
		Name:   "demo",
		Type:   invalidType,
		Config: `{"serverUrl":"tcp://127.0.0.1:1883","productKey":"pk","deviceKey":"dk"}`,
	}

	err := validateNorthboundConfig(config)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "invalid type") {
		t.Fatalf("error = %q, want invalid type", err.Error())
	}
	if config.Type != invalidType {
		t.Fatalf("config.Type = %q, want %q", config.Type, invalidType)
	}
}

func TestValidateNorthboundConfig_SupportsSagooType(t *testing.T) {
	config := &models.NorthboundConfig{
		Name:   "demo",
		Type:   "sagoo",
		Config: `{"serverUrl":"tcp://127.0.0.1:1883","productKey":"pk","deviceKey":"dk"}`,
	}

	if err := validateNorthboundConfig(config); err != nil {
		t.Fatalf("validateNorthboundConfig returned error: %v", err)
	}
	if config.Type != "sagoo" {
		t.Fatalf("config.Type = %q, want %q", config.Type, "sagoo")
	}
}

func TestWriteNorthboundConfigInvalid(t *testing.T) {
	w := httptest.NewRecorder()

	writeNorthboundConfigInvalid(w, nil)

	if w.Result().StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Result().StatusCode, http.StatusBadRequest)
	}

	var parsed parsedErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &parsed); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if parsed.Code != apiErrNorthboundConfigInvalid.Code {
		t.Fatalf("code = %q, want %q", parsed.Code, apiErrNorthboundConfigInvalid.Code)
	}
}

func TestParseAndPrepareNorthboundConfig_InvalidBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/northbound", strings.NewReader(`{"name":`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	config, ok := parseAndPrepareNorthboundConfig(w, req)
	if ok {
		t.Fatal("expected ok=false, got true")
	}
	if config != nil {
		t.Fatalf("config = %#v, want nil", config)
	}
	if w.Result().StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Result().StatusCode, http.StatusBadRequest)
	}
}

func TestParseAndPrepareNorthboundConfig_InvalidConfig(t *testing.T) {
	body := `{"name":"demo","type":"mqtt"}`
	req := httptest.NewRequest(http.MethodPost, "/northbound", strings.NewReader(body))
	req.SetPathValue("id", "1")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	config, ok := parseAndPrepareNorthboundConfig(w, req)
	if ok {
		t.Fatal("expected ok=false, got true")
	}
	if config != nil {
		t.Fatalf("config = %#v, want nil", config)
	}

	var parsed parsedErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &parsed); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if parsed.Code != apiErrNorthboundConfigInvalid.Code {
		t.Fatalf("code = %q, want %q", parsed.Code, apiErrNorthboundConfigInvalid.Code)
	}
}

func TestParseAndPrepareNorthboundConfig_ValidConfig(t *testing.T) {
	body := `{"name":" demo ","type":"MQTT","server_url":" tcp://127.0.0.1:1883 ","topic":" test "}`
	req := httptest.NewRequest(http.MethodPost, "/northbound", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	config, ok := parseAndPrepareNorthboundConfig(w, req)
	if !ok {
		t.Fatal("expected ok=true, got false")
	}
	if config == nil {
		t.Fatal("expected config, got nil")
	}
	if config.Name != "demo" {
		t.Fatalf("config.Name = %q, want demo", config.Name)
	}
	if config.Type != "mqtt" {
		t.Fatalf("config.Type = %q, want mqtt", config.Type)
	}
	if config.ServerURL != "tcp://127.0.0.1:1883" {
		t.Fatalf("config.ServerURL = %q, want trimmed value", config.ServerURL)
	}
	if config.Topic != "test" {
		t.Fatalf("config.Topic = %q, want test", config.Topic)
	}
}

func TestValidateConfigBySchema_TrimmedRequiredString(t *testing.T) {
	err := validateConfigBySchema("pandax", `{"serverUrl":"   ","username":"token"}`)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "MQTT 地址") {
		t.Fatalf("error = %q, want contains %q", err.Error(), "MQTT 地址")
	}
}

func TestValidateConfigBySchema_RequiredStringPass(t *testing.T) {
	err := validateConfigBySchema("pandax", `{"serverUrl":"tcp://127.0.0.1:1883","username":"token"}`)
	if err != nil {
		t.Fatalf("validateConfigBySchema returned error: %v", err)
	}
}

func TestNorthboundAdapterConfig_PrefersSchemaConfig(t *testing.T) {
	config := &models.NorthboundConfig{
		Type:   "mqtt",
		Config: `{"server":"tcp://127.0.0.1:1883","topic":"demo"}`,
		Topic:  "fallback",
	}

	got := northboundAdapterConfig(config)
	if got != config.Config {
		t.Fatalf("northboundAdapterConfig() = %q, want schema config", got)
	}
}

func TestPrepareNorthboundConfig_ValidatesAfterNormalize(t *testing.T) {
	config := &models.NorthboundConfig{
		Name:      " demo ",
		Type:      " MQTT ",
		ServerURL: " tcp://127.0.0.1:1883 ",
		Topic:     " topic ",
	}

	if err := prepareNorthboundConfig(config); err != nil {
		t.Fatalf("prepareNorthboundConfig returned error: %v", err)
	}
	if config.Name != "demo" || config.Type != "mqtt" {
		t.Fatalf("unexpected normalized config: %#v", config)
	}
}
