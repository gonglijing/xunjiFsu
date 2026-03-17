package database

import (
	"errors"
	"testing"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/models"
)

type stubDriverScanner struct {
	values []any
	err    error
}

func (s stubDriverScanner) Scan(dest ...any) error {
	if s.err != nil {
		return s.err
	}

	for i := range dest {
		switch out := dest[i].(type) {
		case *int64:
			*out = s.values[i].(int64)
		case *string:
			*out = s.values[i].(string)
		case *int:
			*out = s.values[i].(int)
		case *time.Time:
			*out = s.values[i].(time.Time)
		}
	}

	return nil
}

func TestScanDriver(t *testing.T) {
	now := time.Now()
	driver := &models.Driver{}
	scanner := stubDriverScanner{
		values: []any{
			int64(3), "demo", "drivers/demo.wasm", "desc", "1.0.0", `{"x":1}`, 1, now, now,
		},
	}

	err := scanDriver(scanner, driver)
	if err != nil {
		t.Fatalf("scanDriver returned error: %v", err)
	}
	if driver.ID != 3 || driver.Name != "demo" || driver.FilePath != "drivers/demo.wasm" {
		t.Fatalf("unexpected driver fields: %+v", driver)
	}
	if driver.Version != "1.0.0" || driver.Enabled != 1 {
		t.Fatalf("unexpected driver state: %+v", driver)
	}
}

func TestScanDriver_Error(t *testing.T) {
	err := scanDriver(stubDriverScanner{err: errors.New("scan failed")}, &models.Driver{})
	if err == nil {
		t.Fatal("expected scanDriver error")
	}
}
