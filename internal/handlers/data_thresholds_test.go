package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gonglijing/xunjiFsu/internal/models"
)

func TestParseThresholdPayload(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/thresholds", strings.NewReader(`{
		"device_id": 2,
		"field_name": " temperature ",
		"operator": " > ",
		"severity": " warning ",
		"message": " too hot ",
		"shielded": 9
	}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	threshold, ok := parseThresholdPayload(w, req)
	if !ok {
		t.Fatal("expected ok=true, got false")
	}
	if threshold == nil {
		t.Fatal("expected threshold, got nil")
	}
	if threshold.FieldName != "temperature" || threshold.Operator != ">" {
		t.Fatalf("unexpected normalized fields: %+v", threshold)
	}
	if threshold.Severity != "warning" || threshold.Message != "too hot" {
		t.Fatalf("unexpected normalized strings: %+v", threshold)
	}
	if threshold.Shielded != 0 {
		t.Fatalf("threshold.Shielded = %d, want 0", threshold.Shielded)
	}
}

func TestBuildThresholdCacheDeviceIDs(t *testing.T) {
	current := &models.Threshold{DeviceID: 2}
	previous := &models.Threshold{DeviceID: 5}

	got := buildThresholdCacheDeviceIDs(current, previous)
	if len(got) != 2 || got[0] != 2 || got[1] != 5 {
		t.Fatalf("buildThresholdCacheDeviceIDs = %#v, want [2 5]", got)
	}
}

func TestBuildThresholdCacheDeviceIDs_DeduplicatesIDs(t *testing.T) {
	current := &models.Threshold{DeviceID: 2}
	previous := &models.Threshold{DeviceID: 2}

	got := buildThresholdCacheDeviceIDs(current, previous)
	if len(got) != 1 || got[0] != 2 {
		t.Fatalf("buildThresholdCacheDeviceIDs = %#v, want [2]", got)
	}
}

func TestValidateAlarmRepeatIntervalSeconds(t *testing.T) {
	if err := validateAlarmRepeatIntervalSeconds(60); err != nil {
		t.Fatalf("validateAlarmRepeatIntervalSeconds returned error: %v", err)
	}
}

func TestValidateAlarmRepeatIntervalSeconds_Invalid(t *testing.T) {
	if err := validateAlarmRepeatIntervalSeconds(0); err == nil {
		t.Fatal("expected validateAlarmRepeatIntervalSeconds error")
	}
}
