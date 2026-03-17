package handlers

import (
	"testing"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/models"
	"github.com/gonglijing/xunjiFsu/internal/northbound"
)

func TestNorthboundConfigsByName_SkipsNilAndBlankNames(t *testing.T) {
	configs := []*models.NorthboundConfig{
		nil,
		{ID: 1, Name: " alpha "},
		{ID: 2, Name: "   "},
		{ID: 3, Name: "beta"},
	}

	got := northboundConfigsByName(configs)

	if len(got) != 2 {
		t.Fatalf("len(got) = %d, want 2", len(got))
	}
	if got["alpha"] == nil || got["alpha"].ID != 1 {
		t.Fatalf("got[alpha] = %#v, want ID 1", got["alpha"])
	}
	if got["beta"] == nil || got["beta"].ID != 3 {
		t.Fatalf("got[beta] = %#v, want ID 3", got["beta"])
	}
}

func TestListNorthboundStatusNames_MergesAndSorts(t *testing.T) {
	configByName := map[string]*models.NorthboundConfig{
		"beta":  {ID: 2, Name: "beta"},
		"alpha": {ID: 1, Name: "alpha"},
	}

	got := listNorthboundStatusNames(configByName, []string{" gamma ", "alpha", "", "beta", "delta"})

	want := []string{"alpha", "beta", "delta", "gamma"}
	if len(got) != len(want) {
		t.Fatalf("len(got) = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestBuildNorthboundStatusItem_FromConfigAndRuntime(t *testing.T) {
	lastSentAt := time.Date(2026, 3, 18, 10, 20, 30, 0, time.UTC)
	config := &models.NorthboundConfig{
		ID:             7,
		Name:           "demo",
		Type:           "MQTT",
		Enabled:        1,
		UploadInterval: 5000,
	}
	runtime := northbound.RuntimeStatus{
		Registered:       true,
		Enabled:          true,
		Connected:        true,
		UploadIntervalMS: 3000,
		Pending:          true,
		BreakerState:     "closed",
		LastSentAt:       lastSentAt,
	}

	got := buildNorthboundStatusItem("demo", config, runtime)

	if got.ID != 7 {
		t.Fatalf("got.ID = %d, want 7", got.ID)
	}
	if got.Type != "mqtt" {
		t.Fatalf("got.Type = %q, want mqtt", got.Type)
	}
	if !got.Configured || !got.Registered || !got.Enabled || !got.Connected || !got.Pending {
		t.Fatalf("unexpected runtime flags: %#v", got)
	}
	if !got.DBEnabled || got.DBUploadInterval != 5000 {
		t.Fatalf("unexpected db fields: %#v", got)
	}
	if got.UploadInterval != 3000 {
		t.Fatalf("got.UploadInterval = %d, want 3000", got.UploadInterval)
	}
	if got.LastSentAt != "2026-03-18T10:20:30Z" {
		t.Fatalf("got.LastSentAt = %q, want RFC3339 timestamp", got.LastSentAt)
	}
}

func TestBuildNorthboundStatusItem_RuntimeOnly(t *testing.T) {
	runtime := northbound.RuntimeStatus{
		Registered:       true,
		Enabled:          false,
		Connected:        false,
		UploadIntervalMS: 1000,
		BreakerState:     "open",
	}

	got := buildNorthboundStatusItem("runtime-only", nil, runtime)

	if got.Configured {
		t.Fatal("got.Configured = true, want false")
	}
	if got.Name != "runtime-only" {
		t.Fatalf("got.Name = %q, want runtime-only", got.Name)
	}
	if got.Type != "" || got.ID != 0 {
		t.Fatalf("unexpected config fields for runtime-only item: %#v", got)
	}
	if got.LastSentAt != "" {
		t.Fatalf("got.LastSentAt = %q, want empty", got.LastSentAt)
	}
}
