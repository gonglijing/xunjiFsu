package database

import (
	"errors"
	"testing"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/models"
)

type stubDataPointScanner struct {
	values []any
	err    error
}

func (s stubDataPointScanner) Scan(dest ...any) error {
	if s.err != nil {
		return s.err
	}

	for i := range dest {
		switch out := dest[i].(type) {
		case *int64:
			*out = s.values[i].(int64)
		case *string:
			*out = s.values[i].(string)
		case *time.Time:
			*out = s.values[i].(time.Time)
		}
	}

	return nil
}

func TestScanDataPoint(t *testing.T) {
	now := time.Now()
	point := &DataPoint{}
	scanner := stubDataPointScanner{
		values: []any{int64(1), int64(2), "dev-2", "temperature", "26.5", "float", now},
	}

	err := scanDataPoint(scanner, point)
	if err != nil {
		t.Fatalf("scanDataPoint returned error: %v", err)
	}
	if point.ID != 1 || point.DeviceID != 2 || point.DeviceName != "dev-2" {
		t.Fatalf("unexpected data point core fields: %+v", point)
	}
	if point.FieldName != "temperature" || point.Value != "26.5" || point.ValueType != "float" {
		t.Fatalf("unexpected data point value fields: %+v", point)
	}
}

func TestScanDataPoint_Error(t *testing.T) {
	err := scanDataPoint(stubDataPointScanner{err: errors.New("scan failed")}, &DataPoint{})
	if err == nil {
		t.Fatal("expected scanDataPoint error")
	}
}

func TestNormalizeDeviceName_SystemDevice(t *testing.T) {
	got := normalizeDeviceName(models.SystemStatsDeviceID, "ignored")
	if got != models.SystemStatsDeviceName {
		t.Fatalf("normalizeDeviceName(system) = %q, want %q", got, models.SystemStatsDeviceName)
	}
}

func TestNormalizeDeviceName_TrimmedName(t *testing.T) {
	got := normalizeDeviceName(1, " dev-1 ")
	if got != "dev-1" {
		t.Fatalf("normalizeDeviceName = %q, want %q", got, "dev-1")
	}
}

func TestEstimateQueryListCap(t *testing.T) {
	cases := []struct {
		name string
		args []any
		want int
	}{
		{name: "no args", args: nil, want: 0},
		{name: "last int limit", args: []any{int64(1), 200}, want: 200},
		{name: "last int64 limit", args: []any{"temperature", int64(128)}, want: 128},
		{name: "non positive", args: []any{0}, want: 0},
		{name: "too large", args: []any{50000}, want: 0},
		{name: "non numeric tail", args: []any{100, "tail"}, want: 0},
	}

	for _, tc := range cases {
		if got := estimateQueryListCap(tc.args); got != tc.want {
			t.Fatalf("%s: estimateQueryListCap(%v) = %d, want %d", tc.name, tc.args, got, tc.want)
		}
	}
}
