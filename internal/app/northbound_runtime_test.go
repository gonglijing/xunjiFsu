package app

import (
	"testing"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/models"
	"github.com/gonglijing/xunjiFsu/internal/northbound"
	"github.com/gonglijing/xunjiFsu/internal/northbound/adapters"
)

func TestApplyNorthboundSchedulerConfig(t *testing.T) {
	manager := northbound.NewNorthboundManager()
	adapter := adapters.NewMQTTAdapter("nb-demo")
	manager.RegisterAdapter("nb-demo", adapter)

	cfg := &models.NorthboundConfig{Name: "nb-demo", UploadInterval: 1500, Enabled: 1}
	applyNorthboundSchedulerConfig(manager, cfg)

	if got := manager.GetInterval("nb-demo"); got != 1500*time.Millisecond {
		t.Fatalf("GetInterval() = %v, want %v", got, 1500*time.Millisecond)
	}
	if !manager.IsEnabled("nb-demo") {
		t.Fatal("IsEnabled() = false, want true")
	}
}

func TestApplyNorthboundSchedulerConfig_IgnoresNilInputs(t *testing.T) {
	manager := northbound.NewNorthboundManager()
	cfg := &models.NorthboundConfig{Name: "nb-demo", UploadInterval: 1000, Enabled: 1}

	applyNorthboundSchedulerConfig(nil, cfg)
	applyNorthboundSchedulerConfig(manager, nil)
}

func TestApplyNorthboundSchedulerConfig_DisablesAdapter(t *testing.T) {
	manager := northbound.NewNorthboundManager()
	adapter := adapters.NewMQTTAdapter("nb-demo")
	manager.RegisterAdapter("nb-demo", adapter)
	manager.SetEnabled("nb-demo", true)

	cfg := &models.NorthboundConfig{Name: "nb-demo", UploadInterval: 800, Enabled: 0}
	applyNorthboundSchedulerConfig(manager, cfg)

	if manager.IsEnabled("nb-demo") {
		t.Fatal("IsEnabled() = true, want false")
	}
	if got := manager.GetInterval("nb-demo"); got != 800*time.Millisecond {
		t.Fatalf("GetInterval() = %v, want %v", got, 800*time.Millisecond)
	}
}
