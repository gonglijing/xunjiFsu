package driver

import "testing"

func TestFormatDriverValue(t *testing.T) {
	cases := []struct {
		name  string
		input interface{}
		want  string
	}{
		{name: "nil", input: nil, want: ""},
		{name: "string", input: "abc", want: "abc"},
		{name: "bytes", input: []byte("ab"), want: "ab"},
		{name: "bool", input: true, want: "true"},
		{name: "int", input: int64(12), want: "12"},
		{name: "uint", input: uint(7), want: "7"},
		{name: "float64", input: 1.25, want: "1.250000"},
		{name: "float32", input: float32(2.5), want: "2.500000"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := formatDriverValue(tc.input); got != tc.want {
				t.Fatalf("formatDriverValue(%T)=%q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestMapResultFields(t *testing.T) {
	result := &DriverResult{
		Data: map[string]string{"legacy": "1"},
	}
	fields := mapResultFields(result)
	if fields["legacy"] != "1" {
		t.Fatalf("legacy field mismatch: %#v", fields)
	}

	pointResult := &DriverResult{
		Data: map[string]string{"legacy": "1"},
		Points: []DriverPoint{
			{FieldName: "temp", Value: 25.5},
			{FieldName: "", Value: 1},
		},
	}
	fields = mapResultFields(pointResult)
	if len(fields) != 1 || fields["temp"] != "25.500000" {
		t.Fatalf("point fields mismatch: %#v", fields)
	}
}

func TestMapResultFields_ReusesCleanDataMap(t *testing.T) {
	data := map[string]string{"legacy": "1", "temp": "2"}
	result := &DriverResult{Data: data}

	fields := mapResultFields(result)
	fields["extra"] = "3"

	if data["extra"] != "3" {
		t.Fatalf("expected clean data map to be reused")
	}
}

func TestMapResultFields_SkipIdentityFields(t *testing.T) {
	result := &DriverResult{
		Data: map[string]string{
			"temperature": "26.1",
			"product_key": "prodA",
		},
	}
	fields := mapResultFields(result)
	if _, ok := fields["product_key"]; ok {
		t.Fatalf("identity field should be skipped: %#v", fields)
	}
	if fields["temperature"] != "26.1" {
		t.Fatalf("temperature mismatch: %#v", fields)
	}
}

func TestMapResultFields_TrimsDirtyKeysIntoNewMap(t *testing.T) {
	data := map[string]string{
		" temp ": "26.1",
		"   ":    "bad",
	}
	result := &DriverResult{Data: data}

	fields := mapResultFields(result)
	fields["temp"] = "27.0"

	if _, ok := fields[""]; ok {
		t.Fatalf("blank key should be removed: %#v", fields)
	}
	if fields["temp"] != "27.0" {
		t.Fatalf("trimmed temp mismatch: %#v", fields)
	}
	if data[" temp "] != "26.1" {
		t.Fatalf("expected original dirty data map to remain unchanged")
	}
}

func TestNormalizeDriverResultIdentity(t *testing.T) {
	result := &DriverResult{Data: map[string]string{"product_key": "prodA", "temp": "1"}}
	raw := []byte(`{"success":true,"product_key":"prodB"}`)
	normalizeDriverResultIdentity(result, raw)

	if result.ProductKey != "prodA" {
		t.Fatalf("product key mismatch: %s", result.ProductKey)
	}
	if _, ok := result.Data["product_key"]; ok {
		t.Fatalf("product_key should be removed from data: %#v", result.Data)
	}
}

func TestNormalizeDriverResultIdentity_FromRawOutput(t *testing.T) {
	result := &DriverResult{Data: map[string]string{"temp": "1"}}
	raw := []byte(`{"success":true,"data":{"productKey":"prodX"}}`)
	normalizeDriverResultIdentity(result, raw)

	if result.ProductKey != "prodX" {
		t.Fatalf("product key mismatch: %s", result.ProductKey)
	}
}
