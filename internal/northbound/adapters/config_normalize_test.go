package adapters

import "testing"

func TestApplyFallbackOrDefaultPositiveInt(t *testing.T) {
	value := 0
	applyFallbackOrDefaultPositiveInt(&value, 3000, 5000)
	if value != 3000 {
		t.Fatalf("value = %d, want 3000", value)
	}

	value = 0
	applyFallbackOrDefaultPositiveInt(&value, 0, 5000)
	if value != 5000 {
		t.Fatalf("value = %d, want 5000", value)
	}
}

func TestApplyMinimumPositiveInt(t *testing.T) {
	value := 300
	applyMinimumPositiveInt(&value, 500)
	if value != 500 {
		t.Fatalf("value = %d, want 500", value)
	}

	value = 0
	applyMinimumPositiveInt(&value, 500)
	if value != 0 {
		t.Fatalf("value = %d, want 0", value)
	}
}
