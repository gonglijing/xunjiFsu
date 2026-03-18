package models

import "testing"

func TestCollectDataEnsureFields_MergesOnce(t *testing.T) {
	data := &CollectData{
		Fields: map[string]string{
			"temperature": "20",
		},
		Points: []CollectPoint{
			{FieldName: "temperature", Value: 21.5},
			{FieldName: "humidity", Value: 50},
		},
	}

	fields := data.EnsureFields()
	if len(fields) != 2 {
		t.Fatalf("len(fields) = %d, want 2", len(fields))
	}
	if fields["temperature"] != "21.5" {
		t.Fatalf("temperature = %q, want 21.5", fields["temperature"])
	}
	if fields["humidity"] != "50" {
		t.Fatalf("humidity = %q, want 50", fields["humidity"])
	}

	fields["temperature"] = "22.0"
	fields2 := data.EnsureFields()
	if fields2["temperature"] != "22.0" {
		t.Fatalf("EnsureFields should not re-merge after first call, got %q", fields2["temperature"])
	}
}

func TestCollectDataEnsureFields_UnicodeWhitespaceTrimmed(t *testing.T) {
	data := &CollectData{
		Points: []CollectPoint{
			{FieldName: "\u3000temperature\u3000", Value: 21.5},
			{FieldName: "\u3000humidity\u3000", Value: 50},
		},
	}

	fields := data.EnsureFields()
	if len(fields) != 2 {
		t.Fatalf("len(fields) = %d, want 2", len(fields))
	}
	if fields["temperature"] != "21.5" {
		t.Fatalf("temperature = %q, want 21.5", fields["temperature"])
	}
	if fields["humidity"] != "50" {
		t.Fatalf("humidity = %q, want 50", fields["humidity"])
	}
}
