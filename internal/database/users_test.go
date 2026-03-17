package database

import (
	"errors"
	"testing"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/models"
)

type stubUserScanner struct {
	values []any
	err    error
}

func (s stubUserScanner) Scan(dest ...any) error {
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

func TestScanUser(t *testing.T) {
	now := time.Now()
	user := &models.User{}
	scanner := stubUserScanner{
		values: []any{
			int64(1), "admin", "secret", "admin", now, now,
		},
	}

	err := scanUser(scanner, user)
	if err != nil {
		t.Fatalf("scanUser returned error: %v", err)
	}
	if user.ID != 1 || user.Username != "admin" || user.Password != "secret" || user.Role != "admin" {
		t.Fatalf("unexpected user: %+v", user)
	}
}

func TestScanUser_Error(t *testing.T) {
	err := scanUser(stubUserScanner{err: errors.New("scan failed")}, &models.User{})
	if err == nil {
		t.Fatal("expected scanUser error")
	}
}
