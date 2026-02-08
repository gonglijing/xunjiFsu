package collector

import (
	"testing"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/models"
	"github.com/gonglijing/xunjiFsu/internal/northbound"
)

func TestResolveStorageInterval_DefaultAndCustom(t *testing.T) {
	// DefaultStorageIntervalSeconds = 300 (const)
	if got := resolveStorageInterval(0); got != 300*time.Second {
		t.Fatalf("expected default 300s, got %v", got)
	}
	if got := resolveStorageInterval(-10); got != 300*time.Second {
		t.Fatalf("expected negative seconds to use default, got %v", got)
	}
	if got := resolveStorageInterval(30); got != 30*time.Second {
		t.Fatalf("expected 30s, got %v", got)
	}
}

func TestResolveCollectInterval_DefaultAndCustom(t *testing.T) {
	if got := resolveCollectInterval(0); got != 5000*time.Millisecond {
		t.Fatalf("expected default 5000ms, got %v", got)
	}
	if got := resolveCollectInterval(-10); got != 5000*time.Millisecond {
		t.Fatalf("expected negative milliseconds to use default, got %v", got)
	}
	if got := resolveCollectInterval(1200); got != 1200*time.Millisecond {
		t.Fatalf("expected 1200ms, got %v", got)
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

type noOpNorthbound struct{}

func (n *noOpNorthbound) Initialize(config string) error             { return nil }
func (n *noOpNorthbound) Send(data *models.CollectData) error        { return nil }
func (n *noOpNorthbound) SendAlarm(alarm *models.AlarmPayload) error { return nil }
func (n *noOpNorthbound) Close() error                               { return nil }
func (n *noOpNorthbound) Name() string                               { return "noop" }
func (n *noOpNorthbound) PullCommands(limit int) ([]*models.NorthboundCommand, error) {
	return nil, nil
}
func (n *noOpNorthbound) ReportCommandResult(result *models.NorthboundCommandResult) error {
	return nil
}

// 仅验证 Collector 的构造和 IsRunning/Stop 逻辑（不启动后台 goroutine）
func TestCollector_IsRunningAndStop(t *testing.T) {
	mgr := northbound.NewNorthboundManager()
	c := NewCollector(nil, mgr)

	if c.IsRunning() {
		t.Fatalf("expected collector to be not running initially")
	}
}

func TestNewCollectTask_PreservePreviousTimes(t *testing.T) {
	device := &models.Device{ID: 1, CollectInterval: 1000, StorageInterval: 60}
	now := time.Now().Add(-time.Minute)
	previous := &collectTask{
		device:     device,
		lastRun:    now,
		lastStored: now,
	}

	task := newCollectTask(device, previous)

	if task.lastRun != now {
		t.Fatalf("lastRun not preserved")
	}
	if task.lastStored != now {
		t.Fatalf("lastStored not preserved")
	}
}

func TestCollectorTaskIdentityChecks(t *testing.T) {
	mgr := northbound.NewNorthboundManager()
	c := NewCollector(nil, mgr)

	device := &models.Device{ID: 7, CollectInterval: 1000, StorageInterval: 60}
	task := newCollectTask(device, nil)

	c.mu.Lock()
	c.tasks[device.ID] = task
	c.mu.Unlock()

	if !c.isTaskCurrent(task) {
		t.Fatalf("expected task current")
	}

	newTask := newCollectTask(device, task)
	c.mu.Lock()
	c.tasks[device.ID] = newTask
	c.mu.Unlock()

	if c.isTaskCurrent(task) {
		t.Fatalf("expected old task stale")
	}
	if !c.isTaskCurrent(newTask) {
		t.Fatalf("expected new task current")
	}
}

func TestMarkTaskCollected_OnlyUpdatesCurrentTask(t *testing.T) {
	mgr := northbound.NewNorthboundManager()
	c := NewCollector(nil, mgr)

	device := &models.Device{ID: 9, CollectInterval: 1000, StorageInterval: 60}
	oldTask := newCollectTask(device, nil)
	newTask := newCollectTask(device, oldTask)

	c.mu.Lock()
	c.tasks[device.ID] = newTask
	c.mu.Unlock()

	stamp := time.Now()
	c.markTaskCollected(oldTask, stamp, true)
	if !oldTask.lastRun.IsZero() || !oldTask.lastStored.IsZero() {
		t.Fatalf("stale task should not be updated")
	}

	c.markTaskCollected(newTask, stamp, true)
	if newTask.lastRun.IsZero() {
		t.Fatalf("current task lastRun should be updated")
	}
	if newTask.lastStored.IsZero() {
		t.Fatalf("current task lastStored should be updated when stored=true")
	}
}

func TestParseFloatFieldValue(t *testing.T) {
	if _, ok := parseFloatFieldValue(nil, "temp"); ok {
		t.Fatalf("nil fields should return false")
	}

	fields := map[string]string{
		"temp": "  12.5  ",
		"bad":  "abc",
	}

	if v, ok := parseFloatFieldValue(fields, "temp"); !ok || v != 12.5 {
		t.Fatalf("expected temp=12.5, got (%v, %v)", v, ok)
	}

	if _, ok := parseFloatFieldValue(fields, "missing"); ok {
		t.Fatalf("missing field should return false")
	}

	if _, ok := parseFloatFieldValue(fields, "bad"); ok {
		t.Fatalf("invalid float should return false")
	}
}

func TestNormalizeNorthboundCommand(t *testing.T) {
	if _, err := normalizeNorthboundCommand(nil); err == nil {
		t.Fatalf("nil command should return error")
	}

	if _, err := normalizeNorthboundCommand(&models.NorthboundCommand{}); err == nil {
		t.Fatalf("missing identity should return error")
	}

	if _, err := normalizeNorthboundCommand(&models.NorthboundCommand{ProductKey: "p", DeviceKey: "d"}); err == nil {
		t.Fatalf("missing field should return error")
	}

	normalizedCommand, err := normalizeNorthboundCommand(&models.NorthboundCommand{
		RequestID:  "  r1 ",
		ProductKey: "  p1 ",
		DeviceKey:  " d1 ",
		FieldName:  " status ",
		Value:      " 1 ",
		Source:     " cloud ",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if normalizedCommand.ProductKey != "p1" || normalizedCommand.DeviceKey != "d1" || normalizedCommand.FieldName != "status" || normalizedCommand.Value != "1" || normalizedCommand.Source != "cloud" || normalizedCommand.RequestID != "r1" {
		t.Fatalf("normalize failed: %+v", normalizedCommand)
	}
}

func TestBuildNorthboundCommandConfig(t *testing.T) {
	command := &models.NorthboundCommand{
		ProductKey: "pk",
		DeviceKey:  "dk",
		FieldName:  "switch",
		Value:      "on",
	}
	device := &models.Device{DeviceAddress: "addr-1"}

	config := buildNorthboundCommandConfig(command, device)
	if config == nil {
		t.Fatalf("config should not be nil")
	}
	if config["func_name"] != "write" || config["field_name"] != "switch" || config["value"] != "on" {
		t.Fatalf("config core fields mismatch: %+v", config)
	}
	if config["product_key"] != "pk" || config["productKey"] != "pk" || config["device_key"] != "dk" || config["deviceKey"] != "dk" {
		t.Fatalf("config identity fields mismatch: %+v", config)
	}
	if config["device_address"] != "addr-1" {
		t.Fatalf("config address mismatch: %+v", config)
	}

	if buildNorthboundCommandConfig(nil, device) != nil {
		t.Fatalf("nil command should return nil config")
	}
	if buildNorthboundCommandConfig(command, nil) != nil {
		t.Fatalf("nil device should return nil config")
	}
}

func TestBuildNorthboundCommandResult(t *testing.T) {
	command := &models.NorthboundCommand{
		RequestID:  "r1",
		ProductKey: "pk",
		DeviceKey:  "dk",
		FieldName:  "f",
		Value:      "v",
		Source:     "s",
	}

	successResult := buildNorthboundCommandResult(command, nil)
	if successResult == nil || !successResult.Success || successResult.Code != 200 {
		t.Fatalf("success result mismatch: %+v", successResult)
	}

	failResult := buildNorthboundCommandResult(command, assertErr("x"))
	if failResult == nil || failResult.Success || failResult.Code != 500 || failResult.Message != "x" {
		t.Fatalf("fail result mismatch: %+v", failResult)
	}

	if buildNorthboundCommandResult(nil, nil) != nil {
		t.Fatalf("nil command should return nil")
	}
}

func TestSyncDeviceTaskLocked(t *testing.T) {
	mgr := northbound.NewNorthboundManager()
	c := NewCollector(nil, mgr)

	device := &models.Device{ID: 100, Name: "d-1", Enabled: 1, CollectInterval: 1000, StorageInterval: 60}

	c.mu.Lock()
	action := c.syncDeviceTaskLocked(device)
	c.mu.Unlock()
	if action != deviceSyncActionAdded {
		t.Fatalf("expected added action, got %v", action)
	}

	device.Enabled = 1
	c.mu.Lock()
	action = c.syncDeviceTaskLocked(device)
	c.mu.Unlock()
	if action != deviceSyncActionUpdated {
		t.Fatalf("expected updated action, got %v", action)
	}

	device.Enabled = 0
	c.mu.Lock()
	action = c.syncDeviceTaskLocked(device)
	c.mu.Unlock()
	if action != deviceSyncActionRemoved {
		t.Fatalf("expected removed action, got %v", action)
	}

	c.mu.Lock()
	action = c.syncDeviceTaskLocked(device)
	c.mu.Unlock()
	if action != deviceSyncActionNone {
		t.Fatalf("expected none action, got %v", action)
	}
}

func TestStartTickerWorker_NilWorker(t *testing.T) {
	mgr := northbound.NewNorthboundManager()
	c := NewCollector(nil, mgr)

	c.stopChan = make(chan struct{})
	c.startTickerWorker(5*time.Millisecond, nil)
	close(c.stopChan)
}

func TestWaitForStopOrWake_Wake(t *testing.T) {
	mgr := northbound.NewNorthboundManager()
	c := NewCollector(nil, mgr)
	c.stopChan = make(chan struct{})
	c.wakeChan = make(chan struct{}, 1)

	go func() {
		time.Sleep(10 * time.Millisecond)
		c.wakeChan <- struct{}{}
	}()

	if stopped := c.waitForStopOrWake(200 * time.Millisecond); stopped {
		t.Fatalf("expected wake to return stopped=false")
	}
}

func TestWaitForStopOrWake_Stop(t *testing.T) {
	mgr := northbound.NewNorthboundManager()
	c := NewCollector(nil, mgr)
	c.stopChan = make(chan struct{})
	c.wakeChan = make(chan struct{}, 1)

	go func() {
		time.Sleep(10 * time.Millisecond)
		close(c.stopChan)
	}()

	if stopped := c.waitForStopOrWake(200 * time.Millisecond); !stopped {
		t.Fatalf("expected stop to return stopped=true")
	}
}

func TestPeekNextCurrentTaskLockedSkipsStale(t *testing.T) {
	mgr := northbound.NewNorthboundManager()
	c := NewCollector(nil, mgr)

	device := &models.Device{ID: 11, CollectInterval: 1000, StorageInterval: 60}
	oldTask := newCollectTask(device, nil)
	newTask := newCollectTask(device, oldTask)

	c.mu.Lock()
	c.tasks[device.ID] = newTask
	*c.taskHeap = append(*c.taskHeap, oldTask, newTask)
	peeked := c.peekNextCurrentTaskLocked()
	c.mu.Unlock()

	if peeked != newTask {
		t.Fatalf("expected peeked current task to be new task")
	}
}

type assertErr string

func (e assertErr) Error() string { return string(e) }
