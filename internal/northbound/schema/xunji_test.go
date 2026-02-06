package schema

import "testing"

func TestFieldsByType_XunJi(t *testing.T) {
	fields, ok := FieldsByType("xunji")
	if !ok {
		t.Fatalf("expected xunji schema supported")
	}
	if len(fields) == 0 {
		t.Fatalf("expected non-empty xunji schema fields")
	}

	keys := map[string]struct{}{}
	required := map[string]bool{
		"productKey": false,
		"deviceKey":  false,
		"serverUrl":  false,
	}
	for _, field := range fields {
		if field.Key == "" {
			t.Fatalf("field key should not be empty")
		}
		if _, exists := keys[field.Key]; exists {
			t.Fatalf("duplicate field key found: %s", field.Key)
		}
		keys[field.Key] = struct{}{}
		if _, exists := required[field.Key]; exists && field.Required {
			required[field.Key] = true
		}
	}

	for key, found := range required {
		if !found {
			t.Fatalf("required field missing or not required: %s", key)
		}
	}
}

func TestFieldsByType_Unknown(t *testing.T) {
	fields, ok := FieldsByType("unknown")
	if ok {
		t.Fatalf("expected unknown type unsupported")
	}
	if len(fields) != 0 {
		t.Fatalf("expected no fields for unknown type")
	}
}

func TestFieldsByType_ReturnsClone(t *testing.T) {
	fields, ok := FieldsByType("xunji")
	if !ok || len(fields) == 0 {
		t.Fatalf("expected xunji fields")
	}

	originKey := XunJiConfigSchema[0].Key
	fields[0].Key = "mutated"
	if XunJiConfigSchema[0].Key != originKey {
		t.Fatalf("schema source should not be mutated by returned slice")
	}
}
