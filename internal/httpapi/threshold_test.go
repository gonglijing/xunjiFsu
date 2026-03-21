package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gonglijing/xunjiFsu/internal/database"
	"github.com/gonglijing/xunjiFsu/internal/models"
	"github.com/gonglijing/xunjiFsu/internal/service"
)

func TestParseThresholdRequest(t *testing.T) {
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

	threshold, ok := parseThresholdRequest(w, req)
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

	got := service.BuildThresholdCacheDeviceIDs(current, previous)
	if len(got) != 2 || got[0] != 2 || got[1] != 5 {
		t.Fatalf("BuildThresholdCacheDeviceIDs = %#v, want [2 5]", got)
	}
}

func TestBuildThresholdCacheDeviceIDs_DeduplicatesIDs(t *testing.T) {
	current := &models.Threshold{DeviceID: 2}
	previous := &models.Threshold{DeviceID: 2}

	got := service.BuildThresholdCacheDeviceIDs(current, previous)
	if len(got) != 1 || got[0] != 2 {
		t.Fatalf("BuildThresholdCacheDeviceIDs = %#v, want [2]", got)
	}
}

func TestValidateAlarmRepeatIntervalSeconds(t *testing.T) {
	if err := service.ValidateAlarmRepeatIntervalSeconds(60); err != nil {
		t.Fatalf("ValidateAlarmRepeatIntervalSeconds returned error: %v", err)
	}
}

func TestValidateAlarmRepeatIntervalSeconds_Invalid(t *testing.T) {
	if err := service.ValidateAlarmRepeatIntervalSeconds(0); err == nil {
		t.Fatal("expected ValidateAlarmRepeatIntervalSeconds error")
	}
}

func TestNormalizeThresholdInput(t *testing.T) {
	threshold := &models.Threshold{
		FieldName: " temperature ",
		Operator:  " > ",
		Severity:  " warning ",
		Message:   " too hot ",
		Shielded:  9,
	}

	service.NormalizeThresholdInput(threshold)

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

func TestGetAlarmRepeatIntervalResponseShape(t *testing.T) {
	_ = database.DefaultRetentionDays
}

func TestAlarmRepeatIntervalView_JSONShape(t *testing.T) {
	data, err := json.Marshal(alarmRepeatIntervalView{Seconds: 60})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if string(data) != `{"seconds":60}` {
		t.Fatalf("json = %s", data)
	}
}
