package service

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

	got := SanitizeUser(user)

	if got != user {
		t.Fatal("SanitizeUser should return the same pointer")
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

	got := SanitizeUsers(users)

	if len(got) != 3 {
		t.Fatalf("len(got) = %d, want 3", len(got))
	}
	if got[0].Password != "" || got[2].Password != "" {
		t.Fatalf("passwords were not cleared: %#v", got)
	}
}
