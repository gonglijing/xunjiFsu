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
