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

func TestResolveMapByEitherKey_PrefersPrimaryKey(t *testing.T) {
	values := map[string]interface{}{
		"primary": map[string]interface{}{"name": "primary"},
		"backup":  map[string]interface{}{"name": "backup"},
	}

	got, ok := resolveMapByEitherKey(values, "primary", "backup")
	if !ok {
		t.Fatal("resolveMapByEitherKey() ok=false, want=true")
	}
	if got["name"] != "primary" {
		t.Fatalf("resolveMapByEitherKey()=%v, want primary value", got)
	}
}

func TestResolveInterfaceSliceByEitherKey_UsesFallbackKey(t *testing.T) {
	values := map[string]interface{}{
		"backup": []interface{}{"first", "second"},
	}

	got, ok := resolveInterfaceSliceByEitherKey(values, "primary", "backup")
	if !ok {
		t.Fatal("resolveInterfaceSliceByEitherKey() ok=false, want=true")
	}
	if len(got) != 2 || got[0] != "first" || got[1] != "second" {
		t.Fatalf("resolveInterfaceSliceByEitherKey()=%v, want [first second]", got)
	}
}

func TestPickFirstNonEmpty_ReturnsFirstTrimmedValue(t *testing.T) {
	if got := pickFirstNonEmpty("  ", "\tvalue\t", "fallback"); got != "value" {
		t.Fatalf("pickFirstNonEmpty()=%q, want=value", got)
	}
}

func TestResolveMapValue(t *testing.T) {
	if got, ok := resolveMapValue(map[string]interface{}{"name": "demo"}); !ok || got["name"] != "demo" {
		t.Fatalf("resolveMapValue()=%v, %v, want map with name=demo", got, ok)
	}
	if _, ok := resolveMapValue([]interface{}{"demo"}); ok {
		t.Fatal("resolveMapValue() ok=true for slice input, want false")
	}
}
