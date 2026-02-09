package schema

import (
	"sort"
	"testing"
)

func TestXunJiConfigSchemaSupported(t *testing.T) {
	_, ok := FieldsByType("xunji")
	if !ok {
		t.Fatalf("expected xunji schema supported")
	}
}

func TestXunJiConfigSchemaNonEmpty(t *testing.T) {
	fields, _ := FieldsByType("xunji")
	if len(fields) == 0 {
		t.Fatalf("expected non-empty xunji schema fields")
	}
}

func TestCloneFieldsImmutability(t *testing.T) {
	originKey := XunJiConfigSchema[0].Key
	clone, _ := FieldsByType("xunji")
	clone[0].Key = "modified"

	if XunJiConfigSchema[0].Key != originKey {
		t.Fatalf("schema source should not be mutated by returned slice")
	}
}

func TestSupportedTypes(t *testing.T) {
	types := append([]string(nil), SupportedNorthboundSchemaTypes...)
	sort.Strings(types)

	expected := map[string]bool{"ithings": true, "mqtt": true, "pandax": true, "xunji": true}
	if len(types) != len(expected) {
		t.Fatalf("unexpected supported types len, got: %v", types)
	}
	for _, item := range types {
		if !expected[item] {
			t.Fatalf("unexpected supported type: %s", item)
		}
	}
}

func TestPandaXConfigSchemaSupported(t *testing.T) {
	fields, ok := FieldsByType("pandax")
	if !ok {
		t.Fatalf("expected pandax schema supported")
	}
	if len(fields) == 0 {
		t.Fatalf("expected non-empty pandax schema")
	}

	hasServerURL := false
	hasUsername := false
	hasPassword := false
	hasQOS := false

	for _, field := range fields {
		switch field.Key {
		case "serverUrl":
			hasServerURL = true
		case "username":
			hasUsername = true
		case "password":
			hasPassword = true
		case "qos":
			hasQOS = true
		}
	}

	if !hasServerURL || !hasUsername || !hasPassword || !hasQOS {
		t.Fatalf("pandax schema missing required add-form fields")
	}
	if len(fields) != 4 {
		t.Fatalf("pandax schema should only expose 4 add-form fields, got %d", len(fields))
	}
}

func TestIThingsConfigSchemaSupported(t *testing.T) {
	fields, ok := FieldsByType("ithings")
	if !ok {
		t.Fatalf("expected ithings schema supported")
	}
	if len(fields) == 0 {
		t.Fatalf("expected non-empty ithings schema")
	}
}
