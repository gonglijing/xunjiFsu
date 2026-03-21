package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/models"
	"github.com/gonglijing/xunjiFsu/internal/northbound"
	"github.com/gonglijing/xunjiFsu/internal/service"
)

func TestNormalizeNorthboundConfig_AppliesDefaults(t *testing.T) {
	config := &models.NorthboundConfig{Name: " demo ", Type: " MQTT ", ServerURL: " tcp://127.0.0.1:1883 ", QOS: 3}
	normalizeNorthboundConfig(config)
	if config.Name != "demo" || config.Type != "mqtt" || config.Port != defaultMQTTPort {
		t.Fatalf("config = %#v", config)
	}
}

func TestHasSchemaConfig(t *testing.T) {
	if hasSchemaConfig(&models.NorthboundConfig{Config: "  {}  "}) {
		t.Fatal("expected false")
	}
	if !hasSchemaConfig(&models.NorthboundConfig{Config: `{"server":"mqtt://a"}`}) {
		t.Fatal("expected true")
	}
}

func TestValidateNorthboundConfig_RequiredFieldsFallback(t *testing.T) {
	err := validateNorthboundConfig(&models.NorthboundConfig{Name: "demo", Type: "MQTT"})
	if err == nil || !strings.Contains(err.Error(), "server_url or config is required") {
		t.Fatalf("err = %v", err)
	}
}

func TestParseNorthboundConfigRequest_InvalidBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/northbound", strings.NewReader(`{"name":`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	config, ok := parseNorthboundConfigRequest(w, req)
	if ok || config != nil || w.Result().StatusCode != http.StatusBadRequest {
		t.Fatalf("ok=%v config=%#v status=%d", ok, config, w.Result().StatusCode)
	}
}

func TestParseNorthboundConfigRequest_ValidConfig(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/northbound", strings.NewReader(`{"name":" demo ","type":"MQTT","server_url":" tcp://127.0.0.1:1883 ","topic":" test "}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	config, ok := parseNorthboundConfigRequest(w, req)
	if !ok || config == nil || config.Name != "demo" || config.Type != "mqtt" || config.Topic != "test" {
		t.Fatalf("config=%#v ok=%v", config, ok)
	}
}

func TestValidateConfigBySchema_RequiredStringPass(t *testing.T) {
	if err := validateConfigBySchema("pandax", `{"serverUrl":"tcp://127.0.0.1:1883","username":"token"}`); err != nil {
		t.Fatalf("validateConfigBySchema err = %v", err)
	}
}

func TestLoadNorthboundSchema(t *testing.T) {
	response, err := loadNorthboundSchema("pandax")
	if err != nil {
		t.Fatalf("loadNorthboundSchema err = %v", err)
	}
	if response.Type != "pandax" {
		t.Fatalf("response.Type = %#v", response.Type)
	}
}

func TestNorthboundStatusHelpers(t *testing.T) {
	configs := []*models.NorthboundConfig{nil, {ID: 1, Name: " alpha "}, {ID: 3, Name: "beta"}}
	configByName := service.NorthboundConfigsByName(configs)
	if len(configByName) != 2 {
		t.Fatalf("len(configByName) = %d", len(configByName))
	}
	names := service.ListNorthboundStatusNames(configByName, []string{" gamma ", "alpha", "delta"})
	if strings.Join(names, ",") != "alpha,beta,delta,gamma" {
		t.Fatalf("names = %v", names)
	}
	item := service.BuildNorthboundStatusItems(map[string]*models.NorthboundConfig{
		"demo": {ID: 7, Name: "demo", Type: "MQTT", Enabled: 1, UploadInterval: 5000},
	}, []string{"demo"}, nil)[0]
	if item.ID != 7 || item.Type != "mqtt" {
		t.Fatalf("item = %#v", item)
	}
	_ = northbound.RuntimeStatus{LastSentAt: time.Now()}
}

func TestNorthboundConfigInvalidResponseShape(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/northbound", strings.NewReader(`{"name":"demo","type":"mqtt"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	_, _ = parseNorthboundConfigRequest(w, req)
	var parsed APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &parsed); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if parsed.Code != errNorthboundConfigInvalid.Code {
		t.Fatalf("code = %q", parsed.Code)
	}
}
