package handlers

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestParseExecuteDriverPayload(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/devices/1/execute", strings.NewReader(`{"function":"write","params":{"setpoint":42}}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	payload, ok := parseExecuteDriverPayload(w, req)
	if !ok {
		t.Fatal("expected ok=true, got false")
	}
	if payload == nil {
		t.Fatal("expected payload, got nil")
	}
	if payload.Function != "write" {
		t.Fatalf("payload.Function = %q, want %q", payload.Function, "write")
	}
	if got := payload.Params["setpoint"]; got != float64(42) {
		t.Fatalf("payload.Params[setpoint] = %#v, want %#v", got, float64(42))
	}
}

func TestParseDriverWritables(t *testing.T) {
	writables, err := parseDriverWritables(`{"writable":[{"field":"setpoint"},"switch"]}`)
	if err != nil {
		t.Fatalf("parseDriverWritables returned error: %v", err)
	}

	want := []interface{}{
		map[string]interface{}{"field": "setpoint"},
		"switch",
	}
	if !reflect.DeepEqual(writables, want) {
		t.Fatalf("writables = %#v, want %#v", writables, want)
	}
}

func TestParseDriverWritables_EmptySchema(t *testing.T) {
	writables, err := parseDriverWritables("")
	if err != nil {
		t.Fatalf("parseDriverWritables returned error: %v", err)
	}
	if writables != nil {
		t.Fatalf("writables = %#v, want nil", writables)
	}
}

func TestParseDriverWritables_InvalidJSON(t *testing.T) {
	_, err := parseDriverWritables("{invalid")
	if err == nil {
		t.Fatal("expected parseDriverWritables error")
	}
}
