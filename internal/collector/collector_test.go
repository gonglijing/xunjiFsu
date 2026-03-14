package collector

import (
	"container/heap"
	"context"
	"testing"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/driver"
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

func TestThresholdMatch(t *testing.T) {
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
		if got := thresholdMatch(c.value, c.op, c.threshold); got != c.want {
			t.Fatalf("thresholdMatch(%v %s %v) = %v, want %v", c.value, c.op, c.threshold, got, c.want)
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

func TestNewCollectTask_PreparesReadExecution(t *testing.T) {
	device := &models.Device{
		ID:              12,
		Name:            "d-12",
		DriverType:      "modbus_rtu",
		SerialPort:      "/dev/ttyUSB2",
		BaudRate:        9600,
		DataBits:        8,
		StopBits:        1,
		Parity:          "N",
		DeviceAddress:   "3",
		CollectInterval: 1000,
		StorageInterval: 60,
	}

	task := newCollectTask(device, nil)
	if task.preparedRead == nil {
		t.Fatalf("prepared read should not be nil")
	}
	if task.preparedRead.DriverContext == nil || task.preparedRead.DriverContext.DeviceID != device.ID {
		t.Fatalf("prepared driver context mismatch: %#v", task.preparedRead)
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
	if newTask.consecutiveFailures != 0 || newTask.lastError != "" || newTask.lastErrorKind != collectErrorKindNone || !newTask.lastErrorAt.IsZero() {
		t.Fatalf("current task failure state should be reset on success")
	}
}

func TestMarkTaskFailed_OnlyUpdatesCurrentTask(t *testing.T) {
	mgr := northbound.NewNorthboundManager()
	c := NewCollector(nil, mgr)

	device := &models.Device{ID: 91, Name: "d-91", CollectInterval: 1000, StorageInterval: 60}
	oldTask := newCollectTask(device, nil)
	newTask := newCollectTask(device, oldTask)

	c.mu.Lock()
	c.tasks[device.ID] = newTask
	c.mu.Unlock()

	countOld := c.markTaskFailed(oldTask, assertErr("old-error"), collectErrorKindDriver)
	if countOld != 0 {
		t.Fatalf("stale task should not be marked failed")
	}
	if oldTask.consecutiveFailures != 0 || oldTask.lastError != "" || oldTask.lastErrorKind != collectErrorKindNone || !oldTask.lastErrorAt.IsZero() {
		t.Fatalf("stale task failure state should remain unchanged")
	}

	count1 := c.markTaskFailed(newTask, assertErr("new-error"), collectErrorKindNetwork)
	if count1 != 1 {
		t.Fatalf("expected first failure count=1, got %d", count1)
	}
	count2 := c.markTaskFailed(newTask, assertErr("new-error-2"), collectErrorKindTimeout)
	if count2 != 2 {
		t.Fatalf("expected second failure count=2, got %d", count2)
	}
	if newTask.consecutiveFailures != 2 {
		t.Fatalf("expected consecutive failures=2, got %d", newTask.consecutiveFailures)
	}
	if newTask.lastError != "new-error-2" {
		t.Fatalf("expected last error updated, got %q", newTask.lastError)
	}
	if newTask.lastErrorKind != collectErrorKindTimeout {
		t.Fatalf("expected last error kind timeout, got %s", newTask.lastErrorKind)
	}
	if newTask.lastErrorAt.IsZero() {
		t.Fatalf("expected last error time to be set")
	}
}

func TestClassifyCollectError(t *testing.T) {
	if got := classifyCollectError(nil); got != collectErrorKindNone {
		t.Fatalf("nil error kind = %s, want %s", got, collectErrorKindNone)
	}
	if got := classifyCollectError(context.Canceled); got != collectErrorKindCanceled {
		t.Fatalf("canceled error kind = %s, want %s", got, collectErrorKindCanceled)
	}
	if got := classifyCollectError(context.DeadlineExceeded); got != collectErrorKindTimeout {
		t.Fatalf("deadline error kind = %s, want %s", got, collectErrorKindTimeout)
	}
	if got := classifyCollectError(assertErr("i/o timeout")); got != collectErrorKindTimeout {
		t.Fatalf("timeout string kind = %s, want %s", got, collectErrorKindTimeout)
	}
	if got := classifyCollectError(assertErr("connection refused by peer")); got != collectErrorKindNetwork {
		t.Fatalf("network string kind = %s, want %s", got, collectErrorKindNetwork)
	}
	if got := classifyCollectError(assertErr("serial port open failed")); got != collectErrorKindResource {
		t.Fatalf("resource string kind = %s, want %s", got, collectErrorKindResource)
	}
	if got := classifyCollectError(assertErr("driver plugin error")); got != collectErrorKindDriver {
		t.Fatalf("driver string kind = %s, want %s", got, collectErrorKindDriver)
	}
	if got := classifyCollectError(assertErr("unexpected bad state")); got != collectErrorKindUnknown {
		t.Fatalf("unknown string kind = %s, want %s", got, collectErrorKindUnknown)
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

	fieldsWithSpacing := map[string]string{
		" Humidity ": " 51.5 ",
	}
	if v, ok := parseFloatFieldValue(fieldsWithSpacing, "humidity"); !ok || v != 51.5 {
		t.Fatalf("expected case/trim match humidity=51.5, got (%v, %v)", v, ok)
	}
}

func TestNumericFieldLookup_CacheParsedValue(t *testing.T) {
	lookup := newNumericFieldLookup(map[string]string{
		"temperature": "42.1",
	})
	if lookup.parsed != nil {
		t.Fatalf("parsed cache should be lazily initialized")
	}

	v1, ok1 := lookup.getFloat("temperature")
	v2, ok2 := lookup.getFloat(" temperature ")
	if !ok1 || !ok2 || v1 != 42.1 || v2 != 42.1 {
		t.Fatalf("expected cached parsed value 42.1, got (%v,%v) and (%v,%v)", v1, ok1, v2, ok2)
	}

	if len(lookup.parsed) != 1 {
		t.Fatalf("expected parsed cache size 1, got %d", len(lookup.parsed))
	}
}

func TestDriverResultToCollectData_TrimAndOverrideFields(t *testing.T) {
	device := &models.Device{
		ID:         1,
		Name:       "dev1",
		ProductKey: " pk ",
		DeviceKey:  " dk ",
	}
	res := &driver.DriverResult{
		Data: map[string]string{
			" temp ": "10.5",
			"   ":    "invalid",
		},
		Points: []driver.DriverPoint{
			{FieldName: " temp ", Value: 11.2},
			{FieldName: "", Value: 1},
		},
	}

	collect := driverResultToCollectData(device, res)
	if collect == nil {
		t.Fatalf("collect data should not be nil")
	}
	if collect.ProductKey != "pk" || collect.DeviceKey != "dk" {
		t.Fatalf("expected trimmed identity, got product=%q device=%q", collect.ProductKey, collect.DeviceKey)
	}
	if got := collect.Fields["temp"]; got != "11.2" {
		t.Fatalf("expected point value override temp=11.2, got %q", got)
	}
	if _, exists := collect.Fields[""]; exists {
		t.Fatalf("blank field name should be ignored")
	}
}

func TestDriverResultToCollectData_ProductKeyFromDriver(t *testing.T) {
	device := &models.Device{
		ID:         2,
		Name:       "dev2",
		ProductKey: "oldPk",
		DeviceKey:  "dk2",
	}
	res := &driver.DriverResult{
		ProductKey: "  newPk  ",
		Data:       map[string]string{"temp": "20"},
	}

	collect := driverResultToCollectData(device, res)
	if collect.ProductKey != "newPk" {
		t.Fatalf("expected driver product key override, got %q", collect.ProductKey)
	}
}

func TestDriverResultToCollectData_EmptyResultKeepsNilFields(t *testing.T) {
	device := &models.Device{
		ID:        20,
		Name:      "dev-empty",
		DeviceKey: "dk-empty",
	}

	collect := driverResultToCollectData(device, &driver.DriverResult{})
	if collect == nil {
		t.Fatalf("collect data should not be nil")
	}
	if collect.Fields != nil {
		t.Fatalf("expected nil fields for empty result, got %#v", collect.Fields)
	}
}

func TestSyncDeviceProductKey_UpdateInMemory(t *testing.T) {
	c := NewCollector(nil, nil)
	device := &models.Device{ID: 3, ProductKey: "pk-old"}
	collect := &models.CollectData{ProductKey: "pk-new"}

	if err := c.syncDeviceProductKey(device, collect); err != nil {
		t.Fatalf("syncDeviceProductKey() error = %v", err)
	}
	if device.ProductKey != "pk-new" {
		t.Fatalf("device product key not updated: %q", device.ProductKey)
	}
}

func TestSyncDeviceProductKey_FixedByDriverID(t *testing.T) {
	c := NewCollector(nil, nil)
	driverID := int64(9)

	deviceA := &models.Device{ID: 31, DriverID: &driverID, ProductKey: ""}
	collectA := &models.CollectData{ProductKey: "prod-fixed"}
	if err := c.syncDeviceProductKey(deviceA, collectA); err != nil {
		t.Fatalf("syncDeviceProductKey() error = %v", err)
	}

	deviceB := &models.Device{ID: 32, DriverID: &driverID, ProductKey: ""}
	collectB := &models.CollectData{ProductKey: "prod-other"}
	if err := c.syncDeviceProductKey(deviceB, collectB); err != nil {
		t.Fatalf("syncDeviceProductKey() error = %v", err)
	}

	if collectB.ProductKey != "prod-fixed" {
		t.Fatalf("expected cached product key, got %q", collectB.ProductKey)
	}
	if deviceB.ProductKey != "prod-fixed" {
		t.Fatalf("expected deviceB product key updated to cached value, got %q", deviceB.ProductKey)
	}
}

func TestRemoveTaskLocked_PrunesUnusedDriverProductKeyCache(t *testing.T) {
	mgr := northbound.NewNorthboundManager()
	c := NewCollector(nil, mgr)

	driverID := int64(77)
	device := &models.Device{ID: 201, Enabled: 1, DriverID: &driverID, CollectInterval: 1000, StorageInterval: 60}

	c.mu.Lock()
	task := newCollectTask(device, nil)
	c.tasks[device.ID] = task
	heap.Push(c.taskHeap, task)
	c.mu.Unlock()

	c.driverIdentityMu.Lock()
	c.driverProductKeys[driverID] = "prod-77"
	c.driverIdentityMu.Unlock()

	c.mu.Lock()
	c.removeTaskLocked(device.ID)
	c.mu.Unlock()

	c.driverIdentityMu.RLock()
	_, exists := c.driverProductKeys[driverID]
	c.driverIdentityMu.RUnlock()
	if exists {
		t.Fatalf("expected unused driver product key cache to be pruned")
	}
}

func TestRemoveTaskLocked_KeepsSharedDriverProductKeyCache(t *testing.T) {
	mgr := northbound.NewNorthboundManager()
	c := NewCollector(nil, mgr)

	driverID := int64(88)
	device1 := &models.Device{ID: 301, Enabled: 1, DriverID: &driverID, CollectInterval: 1000, StorageInterval: 60}
	device2 := &models.Device{ID: 302, Enabled: 1, DriverID: &driverID, CollectInterval: 1000, StorageInterval: 60}

	c.mu.Lock()
	task1 := newCollectTask(device1, nil)
	task2 := newCollectTask(device2, nil)
	c.tasks[device1.ID] = task1
	c.tasks[device2.ID] = task2
	heap.Push(c.taskHeap, task1)
	heap.Push(c.taskHeap, task2)
	c.mu.Unlock()

	c.driverIdentityMu.Lock()
	c.driverProductKeys[driverID] = "prod-88"
	c.driverIdentityMu.Unlock()

	c.mu.Lock()
	c.removeTaskLocked(device1.ID)
	c.mu.Unlock()

	c.driverIdentityMu.RLock()
	value, exists := c.driverProductKeys[driverID]
	c.driverIdentityMu.RUnlock()
	if !exists || value != "prod-88" {
		t.Fatalf("expected shared driver product key cache to be kept, got exists=%v value=%q", exists, value)
	}
}

func TestDriverPointValueToString(t *testing.T) {
	cases := []struct {
		name  string
		input interface{}
		want  string
	}{
		{name: "string", input: "abc", want: "abc"},
		{name: "bytes", input: []byte("ab"), want: "ab"},
		{name: "bool", input: true, want: "true"},
		{name: "int", input: int64(123), want: "123"},
		{name: "float", input: 12.5, want: "12.5"},
		{name: "nil", input: nil, want: ""},
	}

	for _, tc := range cases {
		if got := driverPointValueToString(tc.input); got != tc.want {
			t.Fatalf("%s: got %q, want %q", tc.name, got, tc.want)
		}
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
	if action != deviceSyncActionNone {
		t.Fatalf("expected none action for unchanged config, got %v", action)
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

func TestUpsertTaskLocked_UnchangedDeviceDoesNotGrowHeap(t *testing.T) {
	mgr := northbound.NewNorthboundManager()
	c := NewCollector(nil, mgr)

	now := time.Now()
	device := &models.Device{
		ID:              101,
		Enabled:         1,
		Name:            "d-101",
		CollectInterval: 1000,
		StorageInterval: 60,
		UpdatedAt:       now,
	}

	c.mu.Lock()
	action1 := c.upsertTaskLocked(device)
	heapLen1 := len(*c.taskHeap)
	action2 := c.upsertTaskLocked(device)
	heapLen2 := len(*c.taskHeap)
	c.mu.Unlock()

	if action1 != deviceSyncActionAdded {
		t.Fatalf("expected first upsert added, got %v", action1)
	}
	if action2 != deviceSyncActionNone {
		t.Fatalf("expected second upsert none for unchanged device, got %v", action2)
	}
	if heapLen1 != 1 || heapLen2 != 1 {
		t.Fatalf("heap should not grow for unchanged device, got len1=%d len2=%d", heapLen1, heapLen2)
	}
}

func TestUpsertTaskLocked_ConfigChangedDoesNotGrowHeap(t *testing.T) {
	mgr := northbound.NewNorthboundManager()
	c := NewCollector(nil, mgr)

	now := time.Now()
	device := &models.Device{ID: 102, Enabled: 1, Name: "d-102", CollectInterval: 1000, StorageInterval: 60, UpdatedAt: now}

	c.mu.Lock()
	action1 := c.upsertTaskLocked(device)
	oldTask := c.tasks[device.ID]
	device2 := *device
	device2.CollectInterval = 2000
	device2.UpdatedAt = now.Add(time.Second)
	action2 := c.upsertTaskLocked(&device2)
	newTask := c.tasks[device.ID]
	heapLen := len(*c.taskHeap)
	c.mu.Unlock()

	if action1 != deviceSyncActionAdded {
		t.Fatalf("expected first upsert added, got %v", action1)
	}
	if action2 != deviceSyncActionUpdated {
		t.Fatalf("expected second upsert updated, got %v", action2)
	}
	if oldTask != newTask {
		t.Fatalf("expected changed config to refresh existing task in place")
	}
	if heapLen != 1 {
		t.Fatalf("expected heap len 1 after in-place refresh, got %d", heapLen)
	}
}

func TestUpsertTaskLocked_ResourceChangedUpdatesCurrentTask(t *testing.T) {
	mgr := northbound.NewNorthboundManager()
	c := NewCollector(nil, mgr)

	resourceOld := int64(401)
	resourceNew := int64(402)
	now := time.Now()
	device := &models.Device{ID: 401, Enabled: 1, ResourceID: &resourceOld, CollectInterval: 1000, StorageInterval: 60, UpdatedAt: now}

	c.mu.Lock()
	_ = c.upsertTaskLocked(device)

	device2 := *device
	device2.ResourceID = &resourceNew
	device2.UpdatedAt = now.Add(time.Second)
	action := c.upsertTaskLocked(&device2)
	current := c.tasks[device.ID]
	c.mu.Unlock()

	if action != deviceSyncActionUpdated {
		t.Fatalf("expected updated action, got %v", action)
	}
	if current == nil || current.device == nil || current.device.ResourceID == nil {
		t.Fatalf("updated task/resource should not be nil")
	}
	if *current.device.ResourceID != resourceNew {
		t.Fatalf("expected current task resource=%d, got %d", resourceNew, *current.device.ResourceID)
	}
}

func TestUpsertTaskLocked_ResourceChangedDoesNotAffectOtherTask(t *testing.T) {
	mgr := northbound.NewNorthboundManager()
	c := NewCollector(nil, mgr)

	resourceOld := int64(501)
	resourceNew := int64(502)
	now := time.Now()
	deviceA := &models.Device{ID: 501, Enabled: 1, ResourceID: &resourceOld, CollectInterval: 1000, StorageInterval: 60, UpdatedAt: now}
	deviceB := &models.Device{ID: 502, Enabled: 1, ResourceID: &resourceOld, CollectInterval: 1000, StorageInterval: 60, UpdatedAt: now}

	c.mu.Lock()
	_ = c.upsertTaskLocked(deviceA)
	_ = c.upsertTaskLocked(deviceB)

	deviceA2 := *deviceA
	deviceA2.ResourceID = &resourceNew
	deviceA2.UpdatedAt = now.Add(time.Second)
	_ = c.upsertTaskLocked(&deviceA2)
	currentA := c.tasks[deviceA.ID]
	currentB := c.tasks[deviceB.ID]
	c.mu.Unlock()

	if currentA == nil || currentA.device == nil || currentA.device.ResourceID == nil || *currentA.device.ResourceID != resourceNew {
		t.Fatalf("device A should switch to resource %d", resourceNew)
	}
	if currentB == nil || currentB.device == nil || currentB.device.ResourceID == nil || *currentB.device.ResourceID != resourceOld {
		t.Fatalf("device B should keep resource %d", resourceOld)
	}
}

func TestShouldRefreshCollectTask(t *testing.T) {
	now := time.Now()
	baseDevice := &models.Device{
		ID:              103,
		Enabled:         1,
		Name:            "d-103",
		CollectInterval: 1000,
		StorageInterval: 60,
		UpdatedAt:       now,
	}
	task := newCollectTask(baseDevice, nil)

	if shouldRefreshCollectTask(task, baseDevice) {
		t.Fatalf("unchanged device should not require refresh")
	}

	changedInterval := *baseDevice
	changedInterval.CollectInterval = 1500
	if !shouldRefreshCollectTask(task, &changedInterval) {
		t.Fatalf("collect interval change should require refresh")
	}

	changedUpdatedAt := *baseDevice
	changedUpdatedAt.UpdatedAt = now.Add(time.Second)
	if shouldRefreshCollectTask(task, &changedUpdatedAt) {
		t.Fatalf("updated_at only change should not require refresh")
	}

	changedName := *baseDevice
	changedName.Name = "d-103-new"
	changedName.UpdatedAt = now
	if !shouldRefreshCollectTask(task, &changedName) {
		t.Fatalf("device config change should require refresh")
	}
}

func TestRemoveTaskLocked_DeletesTask(t *testing.T) {
	mgr := northbound.NewNorthboundManager()
	c := NewCollector(nil, mgr)

	resourceID := int64(9)
	device := &models.Device{ID: 201, Enabled: 1, ResourceID: &resourceID, CollectInterval: 1000, StorageInterval: 60}

	c.mu.Lock()
	task := newCollectTask(device, nil)
	c.tasks[device.ID] = task
	heap.Push(c.taskHeap, task)
	c.removeTaskLocked(device.ID)
	_, exists := c.tasks[device.ID]
	heapLen := len(*c.taskHeap)
	c.mu.Unlock()

	if exists {
		t.Fatalf("task should be removed")
	}
	if heapLen != 0 {
		t.Fatalf("heap should not retain removed task, got len=%d", heapLen)
	}
}

func TestRemoveTaskLocked_KeepOtherTasks(t *testing.T) {
	mgr := northbound.NewNorthboundManager()
	c := NewCollector(nil, mgr)

	resourceID := int64(10)
	device1 := &models.Device{ID: 202, Enabled: 1, ResourceID: &resourceID, CollectInterval: 1000, StorageInterval: 60}
	device2 := &models.Device{ID: 203, Enabled: 1, ResourceID: &resourceID, CollectInterval: 1000, StorageInterval: 60}

	c.mu.Lock()
	task1 := newCollectTask(device1, nil)
	task2 := newCollectTask(device2, nil)
	c.tasks[device1.ID] = task1
	c.tasks[device2.ID] = task2
	heap.Push(c.taskHeap, task1)
	heap.Push(c.taskHeap, task2)
	c.removeTaskLocked(device1.ID)
	_, hasTask1 := c.tasks[device1.ID]
	_, hasTask2 := c.tasks[device2.ID]
	heapLen := len(*c.taskHeap)
	c.mu.Unlock()

	if hasTask1 {
		t.Fatalf("task1 should be removed")
	}
	if !hasTask2 {
		t.Fatalf("task2 should remain")
	}
	if heapLen != 1 {
		t.Fatalf("expected heap len 1 after removal, got %d", heapLen)
	}
}

func TestPruneMissingTasksLocked(t *testing.T) {
	mgr := northbound.NewNorthboundManager()
	c := NewCollector(nil, mgr)

	resourceA := int64(301)
	resourceB := int64(302)
	device1 := &models.Device{ID: 301, Enabled: 1, ResourceID: &resourceA, CollectInterval: 1000, StorageInterval: 60}
	device2 := &models.Device{ID: 302, Enabled: 1, ResourceID: &resourceB, CollectInterval: 1000, StorageInterval: 60}

	c.mu.Lock()
	c.tasks[device1.ID] = newCollectTask(device1, nil)
	c.tasks[device2.ID] = newCollectTask(device2, nil)

	removed := c.pruneMissingTasksLocked(map[int64]struct{}{device1.ID: {}})
	_, hasTask1 := c.tasks[device1.ID]
	_, hasTask2 := c.tasks[device2.ID]
	c.mu.Unlock()

	if removed != 1 {
		t.Fatalf("expected removed=1, got %d", removed)
	}
	if !hasTask1 || hasTask2 {
		t.Fatalf("expected task1 kept and task2 removed, hasTask1=%v hasTask2=%v", hasTask1, hasTask2)
	}
}

func TestStartAdjustableTickerWorker_NilWorker(t *testing.T) {
	mgr := northbound.NewNorthboundManager()
	c := NewCollector(nil, mgr)

	c.stopChan = make(chan struct{})
	c.startAdjustableTickerWorker(5*time.Millisecond, make(chan time.Duration, 1), nil)
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

func TestSetRuntimeIntervals_WhenNotRunning(t *testing.T) {
	mgr := northbound.NewNorthboundManager()
	c := NewCollector(nil, mgr)

	c.SetRuntimeIntervals(3*time.Second, 400*time.Millisecond)

	deviceSync, commandPoll := c.GetRuntimeIntervals()
	if deviceSync != 3*time.Second {
		t.Fatalf("device sync interval = %v, want 3s", deviceSync)
	}
	if commandPoll != 400*time.Millisecond {
		t.Fatalf("command poll interval = %v, want 400ms", commandPoll)
	}
}

func TestNotifyIntervalChange_ReplacesPending(t *testing.T) {
	ch := make(chan time.Duration, 1)

	notifyIntervalChange(ch, time.Second)
	notifyIntervalChange(ch, 2*time.Second)

	select {
	case got := <-ch:
		if got != 2*time.Second {
			t.Fatalf("interval = %v, want 2s", got)
		}
	default:
		t.Fatalf("expected interval in channel")
	}
}

type assertErr string

func (e assertErr) Error() string { return string(e) }
