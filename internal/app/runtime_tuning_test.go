package app

import (
	"testing"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/collector"
	"github.com/gonglijing/xunjiFsu/internal/platform/config"
)

func TestApplyCollectorRuntimeTuning(t *testing.T) {
	collect := collector.NewCollector(nil, nil)
	cfg := &config.Config{
		CollectorDeviceSyncInterval:  3 * time.Second,
		CollectorCommandPollInterval: 700 * time.Millisecond,
		CollectorWorkers:             6,
	}

	applyCollectorRuntimeTuning(cfg, collect)

	deviceSyncInterval, commandPollInterval := collect.GetRuntimeIntervals()
	if deviceSyncInterval != 3*time.Second {
		t.Fatalf("deviceSyncInterval = %v, want %v", deviceSyncInterval, 3*time.Second)
	}
	if commandPollInterval != 700*time.Millisecond {
		t.Fatalf("commandPollInterval = %v, want %v", commandPollInterval, 700*time.Millisecond)
	}
	if got := collect.GetMaxConcurrentCollects(); got != 6 {
		t.Fatalf("GetMaxConcurrentCollects() = %d, want 6", got)
	}
}

func TestApplyCollectorRuntimeTuning_IgnoresNilInputs(t *testing.T) {
	cfg := &config.Config{
		CollectorDeviceSyncInterval:  2 * time.Second,
		CollectorCommandPollInterval: time.Second,
		CollectorWorkers:             4,
	}

	applyCollectorRuntimeTuning(nil, nil)
	applyCollectorRuntimeTuning(cfg, nil)
}
