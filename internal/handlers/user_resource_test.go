package handlers

import (
	"testing"

	"github.com/gonglijing/xunjiFsu/internal/models"
)

func TestSanitizeUser(t *testing.T) {
	user := &models.User{
		ID:       1,
		Username: "demo",
		Password: "secret",
	}

	got := sanitizeUser(user)

	if got != user {
		t.Fatal("sanitizeUser should return the same pointer")
	}
	if got.Password != "" {
		t.Fatalf("got.Password = %q, want empty", got.Password)
	}
}

func TestSanitizeUsers(t *testing.T) {
	users := []*models.User{
		{Username: "a", Password: "x"},
		nil,
		{Username: "b", Password: "y"},
	}

	got := sanitizeUsers(users)

	if len(got) != 3 {
		t.Fatalf("len(got) = %d, want 3", len(got))
	}
	if got[0].Password != "" || got[2].Password != "" {
		t.Fatalf("passwords were not cleared: %#v", got)
	}
}

func TestNextResourceEnabledState(t *testing.T) {
	if nextResourceEnabledState(0) != 1 {
		t.Fatal("nextResourceEnabledState(0) should return 1")
	}
	if nextResourceEnabledState(1) != 0 {
		t.Fatal("nextResourceEnabledState(1) should return 0")
	}
}
