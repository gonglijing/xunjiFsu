package adapters

import "testing"

func TestPickConfigString(t *testing.T) {
	data := map[string]interface{}{
		"empty": "   ",
		"name":  "demo",
		"num":   123,
	}

	if got := pickConfigString(data, "missing", "empty", "name"); got != "demo" {
		t.Fatalf("pickConfigString() = %q, want %q", got, "demo")
	}
	if got := pickConfigString(data, "num"); got != "123" {
		t.Fatalf("pickConfigString() = %q, want %q", got, "123")
	}
}

func TestPickConfigInt(t *testing.T) {
	data := map[string]interface{}{
		"int":    12,
		"float":  15.8,
		"string": "42",
	}

	if got := pickConfigInt(data, 1, "int"); got != 12 {
		t.Fatalf("pickConfigInt() = %d, want %d", got, 12)
	}
	if got := pickConfigInt(data, 1, "float"); got != 15 {
		t.Fatalf("pickConfigInt() = %d, want %d", got, 15)
	}
	if got := pickConfigInt(data, 1, "string"); got != 42 {
		t.Fatalf("pickConfigInt() = %d, want %d", got, 42)
	}
	if got := pickConfigInt(data, 7, "missing"); got != 7 {
		t.Fatalf("pickConfigInt() = %d, want %d", got, 7)
	}
}

func TestPickConfigBool(t *testing.T) {
	data := map[string]interface{}{
		"trueStr":  "true",
		"falseStr": "no",
		"num":      1,
	}

	if got := pickConfigBool(data, false, "trueStr"); !got {
		t.Fatalf("pickConfigBool() = %v, want true", got)
	}
	if got := pickConfigBool(data, true, "falseStr"); got {
		t.Fatalf("pickConfigBool() = %v, want false", got)
	}
	if got := pickConfigBool(data, false, "num"); !got {
		t.Fatalf("pickConfigBool() = %v, want true", got)
	}
	if got := pickConfigBool(data, true, "missing"); !got {
		t.Fatalf("pickConfigBool() = %v, want true", got)
	}
}
