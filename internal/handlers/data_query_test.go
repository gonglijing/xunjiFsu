package handlers

import (
	"net/http/httptest"
	"testing"
	"time"
)

func TestParseOptionalInt64Query(t *testing.T) {
	req := httptest.NewRequest("GET", "/history?device_id=123", nil)

	value, err := parseOptionalInt64Query(req, "device_id")
	if err != nil {
		t.Fatalf("parseOptionalInt64Query returned error: %v", err)
	}
	if value == nil || *value != 123 {
		t.Fatalf("value = %v, want 123", value)
	}
}

func TestParseOptionalInt64Query_Invalid(t *testing.T) {
	req := httptest.NewRequest("GET", "/history?device_id=abc", nil)

	if _, err := parseOptionalInt64Query(req, "device_id"); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestParseHistoryDataQuery(t *testing.T) {
	req := httptest.NewRequest("GET", "/history?device_id=7&field_name=temp&start=2026-01-02T03:04:05Z&end=2026-01-02T04:04:05Z", nil)

	query, err := parseHistoryDataQuery(req)
	if err != nil {
		t.Fatalf("parseHistoryDataQuery returned error: %v", err)
	}
	if query.DeviceID == nil || *query.DeviceID != 7 {
		t.Fatalf("device id = %v, want 7", query.DeviceID)
	}
	if query.FieldName != "temp" {
		t.Fatalf("field name = %q, want %q", query.FieldName, "temp")
	}
	if query.StartTime.Format(time.RFC3339) != "2026-01-02T03:04:05Z" {
		t.Fatalf("start = %s", query.StartTime.Format(time.RFC3339))
	}
	if query.EndTime.Format(time.RFC3339) != "2026-01-02T04:04:05Z" {
		t.Fatalf("end = %s", query.EndTime.Format(time.RFC3339))
	}
}

func TestParseHistoryDataQuery_InvalidTime(t *testing.T) {
	req := httptest.NewRequest("GET", "/history?start=bad-time", nil)

	if _, err := parseHistoryDataQuery(req); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestParseHistoryDataQuery_StartAfterEnd(t *testing.T) {
	req := httptest.NewRequest("GET", "/history?device_id=1&start=2026-01-02T05:04:05Z&end=2026-01-02T04:04:05Z", nil)

	_, err := parseHistoryDataQuery(req)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != errHistoryStartAfterEndDetail {
		t.Fatalf("error = %q, want %q", err.Error(), errHistoryStartAfterEndDetail)
	}
}

func TestParseHistoryDataQuery_FilterRequiresDeviceID(t *testing.T) {
	req := httptest.NewRequest("GET", "/history?field_name=temp", nil)

	_, err := parseHistoryDataQuery(req)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != errHistoryFilterRequiresDevice {
		t.Fatalf("error = %q, want %q", err.Error(), errHistoryFilterRequiresDevice)
	}
}

func TestParseHistoryDataQuery_InvalidDeviceIDValue(t *testing.T) {
	req := httptest.NewRequest("GET", "/history?device_id=0", nil)

	_, err := parseHistoryDataQuery(req)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != errInvalidDeviceIDMessage {
		t.Fatalf("error = %q, want %q", err.Error(), errInvalidDeviceIDMessage)
	}
}

func TestParseHistoryDataQuery_SystemDeviceIDAllowed(t *testing.T) {
	req := httptest.NewRequest("GET", "/history?device_id=-1&field_name=cpu_usage", nil)

	query, err := parseHistoryDataQuery(req)
	if err != nil {
		t.Fatalf("parseHistoryDataQuery returned error: %v", err)
	}
	if query.DeviceID == nil || *query.DeviceID != -1 {
		t.Fatalf("device id = %v, want -1", query.DeviceID)
	}
	if query.FieldName != "cpu_usage" {
		t.Fatalf("field name = %q, want %q", query.FieldName, "cpu_usage")
	}
}

func TestParseHistoryPointQuery(t *testing.T) {
	req := httptest.NewRequest("GET", "/history?device_id=7&field_name=temp", nil)

	query, err := parseHistoryPointQuery(req)
	if err != nil {
		t.Fatalf("parseHistoryPointQuery returned error: %v", err)
	}
	if query.DeviceID != 7 {
		t.Fatalf("device id = %d, want 7", query.DeviceID)
	}
	if query.FieldName != "temp" {
		t.Fatalf("field name = %q, want %q", query.FieldName, "temp")
	}
}

func TestParseHistoryPointQuery_FieldNameRequired(t *testing.T) {
	req := httptest.NewRequest("GET", "/history?device_id=7", nil)

	_, err := parseHistoryPointQuery(req)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != errHistoryFieldNameRequired {
		t.Fatalf("error = %q, want %q", err.Error(), errHistoryFieldNameRequired)
	}
}

func TestParseHistoryPointQuery_InvalidDeviceID(t *testing.T) {
	req := httptest.NewRequest("GET", "/history?device_id=0&field_name=temp", nil)

	_, err := parseHistoryPointQuery(req)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != errInvalidDeviceIDMessage {
		t.Fatalf("error = %q, want %q", err.Error(), errInvalidDeviceIDMessage)
	}
}
