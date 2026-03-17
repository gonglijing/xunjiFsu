package database

import (
	"errors"
	"testing"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/models"
)

type stubDataCacheScanner struct {
	values []any
	err    error
}

func (s stubDataCacheScanner) Scan(dest ...any) error {
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

func TestScanDataCache(t *testing.T) {
	now := time.Now()
	item := &models.DataCache{}
	scanner := stubDataCacheScanner{
		values: []any{int64(1), int64(2), "temperature", "26.5", "float", now},
	}

	err := scanDataCache(scanner, item)
	if err != nil {
		t.Fatalf("scanDataCache returned error: %v", err)
	}
	if item.ID != 1 || item.DeviceID != 2 || item.FieldName != "temperature" {
		t.Fatalf("unexpected data cache core fields: %+v", item)
	}
	if item.Value != "26.5" || item.ValueType != "float" {
		t.Fatalf("unexpected data cache value fields: %+v", item)
	}
}

func TestScanDataCache_Error(t *testing.T) {
	err := scanDataCache(stubDataCacheScanner{err: errors.New("scan failed")}, &models.DataCache{})
	if err == nil {
		t.Fatal("expected scanDataCache error")
	}
}
