package collector

import (
	"testing"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/database"
	"github.com/gonglijing/xunjiFsu/internal/models"
	"github.com/gonglijing/xunjiFsu/internal/northbound"
)

func TestResolveStorageInterval_DefaultAndCustom(t *testing.T) {
	// 使用一个非常小的默认值，避免引入真实数据库依赖
	oldDefault := database.DefaultStorageIntervalSeconds
	database.DefaultStorageIntervalSeconds = 60
	defer func() { database.DefaultStorageIntervalSeconds = oldDefault }()

	if got := resolveStorageInterval(0); got != 60*time.Second {
		t.Fatalf("expected default 60s, got %v", got)
	}
	if got := resolveStorageInterval(-10); got != 60*time.Second {
		t.Fatalf("expected negative seconds to use default, got %v", got)
	}
	if got := resolveStorageInterval(30); got != 30*time.Second {
		t.Fatalf("expected 30s, got %v", got)
	}
}

func TestShouldStoreHistory(t *testing.T) {
	task := &collectTask{
		storageInterval: 10 * time.Second,
	}
	now := time.Now()

	// 首次采集必须存历史
	if !shouldStoreHistory(task, now) {
		t.Fatalf("expected first store to return true")
	}
	task.lastStored = now

	// 未到间隔不应存
	if shouldStoreHistory(task, now.Add(5*time.Second)) {
		t.Fatalf("expected store=false before interval elapsed")
	}

	// 到达或超过间隔应存
	if !shouldStoreHistory(task, now.Add(10*time.Second)) {
		t.Fatalf("expected store=true at interval boundary")
	}
}

func TestThresholdChecker_Check(t *testing.T) {
	tc := NewThresholdChecker()

	cases := []struct {
		value     float64
		op        string
		threshold float64
		want      bool
	}{
		{10, ">", 5, true},
		{10, "<", 5, false},
		{5, ">=", 5, true},
		{4, ">=", 5, false},
		{5, "<=", 5, true},
		{6, "<=", 5, false},
		{5, "==", 5, true},
		{5, "!=", 6, true},
		{5, "unknown", 5, false},
	}

	for _, c := range cases {
		if got := tc.Check(c.value, c.op, c.threshold); got != c.want {
			t.Fatalf("Check(%v %s %v) = %v, want %v", c.value, c.op, c.threshold, got, c.want)
		}
	}
}

type fakeExecutor struct{}

func (f *fakeExecutor) Execute(device *models.Device) (*driver.DriverResult, error) {
	return &driver.DriverResult{}, nil
}

type noOpNorthbound struct{}

func (n *noOpNorthbound) Initialize(config string) error                     { return nil }
func (n *noOpNorthbound) Send(data *models.CollectData) error               { return nil }
func (n *noOpNorthbound) SendAlarm(alarm *models.AlarmPayload) error        { return nil }
func (n *noOpNorthbound) Close() error                                      { return nil }
func (n *noOpNorthbound) Name() string                                      { return "noop" }

// 仅验证 Collector 的构造和 IsRunning/Stop 逻辑（不启动后台 goroutine）
func TestCollector_IsRunningAndStop(t *testing.T) {
	mgr := northbound.NewNorthboundManager("")
	c := NewCollector(nil, mgr)

	if c.IsRunning() {
		t.Fatalf("expected collector to be not running initially")
	}
}

