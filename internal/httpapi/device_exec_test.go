package httpapi

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
	if !ok || payload == nil {
		t.Fatal("expected parsed payload")
	}
	if payload.Function != "write" {
		t.Fatalf("payload.Function = %q", payload.Function)
	}
	if got := payload.Params["setpoint"]; got != float64(42) {
		t.Fatalf("payload.Params[setpoint] = %#v", got)
	}
}

func TestParseDriverWritables(t *testing.T) {
	writables, err := parseDriverWritables(`{"writable":[{"field":"setpoint"},"switch"]}`)
	if err != nil {
		t.Fatalf("parseDriverWritables error: %v", err)
	}
	want := []any{
		map[string]any{"field": "setpoint"},
		"switch",
	}
	if !reflect.DeepEqual(writables, want) {
		t.Fatalf("writables = %#v", writables)
	}
}

func TestParseDriverWritables_EmptySchema(t *testing.T) {
	writables, err := parseDriverWritables("")
	if err != nil {
		t.Fatalf("parseDriverWritables error: %v", err)
	}
	if writables != nil {
		t.Fatalf("writables = %#v, want nil", writables)
	}
}

func TestParseDriverWritables_InvalidJSON(t *testing.T) {
	if _, err := parseDriverWritables("{invalid"); err == nil {
		t.Fatal("expected parseDriverWritables error")
	}
}
