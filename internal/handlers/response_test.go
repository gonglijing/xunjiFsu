package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
)

type parsedErrorResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
	Code    string `json:"code"`
}

func TestParseFormValue_StringOnlyField(t *testing.T) {
	value := parseFormValue("device_address", "001")
	stringValue, ok := value.(string)
	if !ok {
		t.Fatalf("value type = %T, want string", value)
	}
	if stringValue != "001" {
		t.Fatalf("value = %q, want %q", stringValue, "001")
	}
}

func TestParseFormValue_NumberAndBool(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  interface{}
	}{
		{name: "int", input: "42", want: int64(42)},
		{name: "float", input: "3.14", want: 3.14},
		{name: "bool true", input: "true", want: true},
		{name: "bool one", input: "1", want: int64(1)},
		{name: "bool false", input: "false", want: false},
		{name: "bool zero", input: "0", want: int64(0)},
		{name: "string", input: "abc", want: "abc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseFormValue("any", tt.input)
			if got != tt.want {
				t.Fatalf("parseFormValue(any, %q) = %#v, want %#v", tt.input, got, tt.want)
			}
		})
	}
}

func TestApplyFormDefaults(t *testing.T) {
	formData := map[string]interface{}{
		"parity": "E",
	}

	applyFormDefaults(formData)

	if got := formData["parity"]; got != "E" {
		t.Fatalf("parity = %#v, want %#v", got, "E")
	}
	if got := formData["protocol"]; got != "tcp" {
		t.Fatalf("protocol = %#v, want %#v", got, "tcp")
	}
	if got := formData["baud_rate"]; got != 9600 {
		t.Fatalf("baud_rate = %#v, want %#v", got, 9600)
	}
}

func TestParseRequest_FormDefaultsAndTypes(t *testing.T) {
	body := strings.NewReader("name=test-device&port_num=502&enabled=true")
	req := httptest.NewRequest("POST", "/devices", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	var payload struct {
		Name          string `json:"name"`
		PortNum       int    `json:"port_num"`
		Enabled       bool   `json:"enabled"`
		Protocol      string `json:"protocol"`
		Parity        string `json:"parity"`
		InterfaceType string `json:"interface_type"`
		BaudRate      int    `json:"baud_rate"`
		DataBits      int    `json:"data_bits"`
		StopBits      int    `json:"stop_bits"`
	}

	if err := ParseRequest(req, &payload); err != nil {
		t.Fatalf("ParseRequest returned error: %v", err)
	}

	if payload.Name != "test-device" {
		t.Fatalf("Name = %q, want %q", payload.Name, "test-device")
	}
	if payload.PortNum != 502 {
		t.Fatalf("PortNum = %d, want %d", payload.PortNum, 502)
	}
	if !payload.Enabled {
		t.Fatal("Enabled = false, want true")
	}
	if payload.Protocol != "tcp" {
		t.Fatalf("Protocol = %q, want %q", payload.Protocol, "tcp")
	}
	if payload.Parity != "N" {
		t.Fatalf("Parity = %q, want %q", payload.Parity, "N")
	}
	if payload.InterfaceType != "network" {
		t.Fatalf("InterfaceType = %q, want %q", payload.InterfaceType, "network")
	}
	if payload.BaudRate != 9600 || payload.DataBits != 8 || payload.StopBits != 1 {
		t.Fatalf("serial defaults mismatch: baud=%d data=%d stop=%d", payload.BaudRate, payload.DataBits, payload.StopBits)
	}
}

func TestParseIDOrWriteBadRequest(t *testing.T) {
	tests := []struct {
		name       string
		url        string
		vars       map[string]string
		wantStatus int
		wantOK     bool
		wantID     int64
	}{
		{name: "valid", url: "/users/12", vars: map[string]string{"id": "12"}, wantStatus: http.StatusOK, wantOK: true, wantID: 12},
		{name: "invalid", url: "/users/x", vars: map[string]string{"id": "x"}, wantStatus: http.StatusBadRequest, wantOK: false, wantID: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			req = mux.SetURLVars(req, tt.vars)
			w := httptest.NewRecorder()

			id, ok := parseIDOrWriteBadRequestDefault(w, req)

			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if id != tt.wantID {
				t.Fatalf("id = %d, want %d", id, tt.wantID)
			}
			if w.Result().StatusCode != tt.wantStatus {
				t.Fatalf("status = %d, want %d", w.Result().StatusCode, tt.wantStatus)
			}
			if !tt.wantOK {
				var parsed parsedErrorResponse
				if err := json.Unmarshal(w.Body.Bytes(), &parsed); err != nil {
					t.Fatalf("unmarshal response: %v", err)
				}
				if parsed.Code != apiErrInvalidID.Code {
					t.Fatalf("code = %q, want %q", parsed.Code, apiErrInvalidID.Code)
				}
			}
		})
	}
}

func TestParseRequestOrWriteBadRequest(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantOK     bool
		wantStatus int
	}{
		{name: "valid json", body: `{"name":"demo"}`, wantOK: true, wantStatus: http.StatusOK},
		{name: "invalid json", body: `{"name":`, wantOK: false, wantStatus: http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			var payload struct {
				Name string `json:"name"`
			}
			ok := parseRequestOrWriteBadRequestDefault(w, req, &payload)

			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if w.Result().StatusCode != tt.wantStatus {
				t.Fatalf("status = %d, want %d", w.Result().StatusCode, tt.wantStatus)
			}
			if !tt.wantOK {
				var parsed parsedErrorResponse
				if err := json.Unmarshal(w.Body.Bytes(), &parsed); err != nil {
					t.Fatalf("unmarshal response: %v", err)
				}
				if parsed.Code != apiErrInvalidRequestBody.Code {
					t.Fatalf("code = %q, want %q", parsed.Code, apiErrInvalidRequestBody.Code)
				}
			}
		})
	}
}

func TestWriteErrorDef(t *testing.T) {
	w := httptest.NewRecorder()
	def := APIErrorDef{Code: "E_UNIT", Message: "unit error"}

	WriteBadRequestDef(w, def)

	if w.Result().StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Result().StatusCode, http.StatusBadRequest)
	}

	var parsed parsedErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &parsed); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if parsed.Code != def.Code || parsed.Error != def.Message {
		t.Fatalf("response mismatch: got code=%q err=%q", parsed.Code, parsed.Error)
	}
}

func TestWriteBadRequestCode(t *testing.T) {
	w := httptest.NewRecorder()

	WriteBadRequestCode(w, "E_UNIT_BAD", "unit bad")

	if w.Result().StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Result().StatusCode, http.StatusBadRequest)
	}
	var parsed parsedErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &parsed); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if parsed.Code != "E_UNIT_BAD" || parsed.Error != "unit bad" {
		t.Fatalf("response mismatch: code=%q err=%q", parsed.Code, parsed.Error)
	}
}

func TestWriteDeleted(t *testing.T) {
	w := httptest.NewRecorder()

	WriteDeleted(w)

	if w.Result().StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Result().StatusCode, http.StatusOK)
	}
	if !strings.Contains(w.Body.String(), `"success":true`) {
		t.Fatalf("body = %s, want success true", w.Body.String())
	}
}

func TestParseDefaultHelpers(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/users/9", strings.NewReader(`{"name":"demo"}`))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": "9"})
	w := httptest.NewRecorder()

	id, ok := parseIDOrWriteBadRequestDefault(w, req)
	if !ok || id != 9 {
		t.Fatalf("id parse failed: ok=%v id=%d", ok, id)
	}

	var payload struct {
		Name string `json:"name"`
	}
	if !parseRequestOrWriteBadRequestDefault(w, req, &payload) {
		t.Fatal("expected parseRequestOrWriteBadRequestDefault success")
	}
	if payload.Name != "demo" {
		t.Fatalf("payload.Name = %q, want demo", payload.Name)
	}
}
