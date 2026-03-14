package adapters

import (
	"encoding/json"
	"testing"
)

func TestJSONFieldValueMap_MarshalJSON(t *testing.T) {
	raw, err := json.Marshal(jsonFieldValueMap{
		"bool_true":  " true ",
		"int_val":    "42",
		"float_val":  "23.5",
		"string_val": "  room A  ",
		"empty_val":  "   ",
	})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	decoded := make(map[string]interface{})
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if decoded["bool_true"] != true {
		t.Fatalf("bool_true=%v, want=true", decoded["bool_true"])
	}
	if decoded["int_val"] != float64(42) {
		t.Fatalf("int_val=%v, want=42", decoded["int_val"])
	}
	if decoded["float_val"] != 23.5 {
		t.Fatalf("float_val=%v, want=23.5", decoded["float_val"])
	}
	if decoded["string_val"] != "  room A  " {
		t.Fatalf("string_val=%q, want exact original", decoded["string_val"])
	}
	if decoded["empty_val"] != "" {
		t.Fatalf("empty_val=%q, want empty string", decoded["empty_val"])
	}
}

func TestJSONConvertedValue_MarshalJSON(t *testing.T) {
	raw, err := json.Marshal(jsonConvertedValue(" 23.5 "))
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if string(raw) != "23.5" {
		t.Fatalf("raw=%s, want=23.5", raw)
	}
}

func TestJSONSingleConvertedField_MarshalJSON(t *testing.T) {
	raw, err := json.Marshal(jsonSingleConvertedField{
		Key:   "running",
		Value: " true ",
	})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	decoded := make(map[string]interface{})
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if decoded["running"] != true {
		t.Fatalf("running=%v, want=true", decoded["running"])
	}
}

func TestJSONSingleRawField_MarshalJSON(t *testing.T) {
	raw, err := json.Marshal(jsonSingleRawField{
		Key:   "temperature",
		Value: "23.5",
	})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	decoded := make(map[string]interface{})
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if decoded["temperature"] != "23.5" {
		t.Fatalf("temperature=%v, want raw string", decoded["temperature"])
	}
}
