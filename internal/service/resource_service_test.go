package service

import "testing"

func TestNextResourceEnabledState(t *testing.T) {
	if nextResourceEnabledState(0) != 1 {
		t.Fatal("nextResourceEnabledState(0) should return 1")
	}
	if nextResourceEnabledState(1) != 0 {
		t.Fatal("nextResourceEnabledState(1) should return 0")
	}
}
