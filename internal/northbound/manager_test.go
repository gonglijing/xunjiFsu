package northbound

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/circuit"
	"github.com/gonglijing/xunjiFsu/internal/models"
	"github.com/gonglijing/xunjiFsu/internal/northbound/adapters"
)

// fakeAdapter 模拟适配器（实现 NorthboundAdapter 接口）
type fakeAdapter struct {
	name        string
	sendCalls   int32
	alarmCalls  int32
	reportCalls int32
	fail       bool
	enabled     bool
	connected   bool
	interval   time.Duration
	lastSend   time.Time
	commands   []*models.NorthboundCommand // commands to return from PullCommands
	lastResult *models.NorthboundCommandResult
}

func (f *fakeAdapter) Name() string                                    { return f.name }
func (f *fakeAdapter) Type() string                                    { return "fake" }
func (f *fakeAdapter) Initialize(config string) error                   { return nil }
func (f *fakeAdapter) Start()                                          { f.enabled = true }
func (f *fakeAdapter) Stop()                                           { f.enabled = false }
func (f *fakeAdapter) Close() error                                    { return nil }
func (f *fakeAdapter) SetInterval(interval time.Duration)              { f.interval = interval }
func (f *fakeAdapter) IsEnabled() bool                                { return f.enabled }
func (f *fakeAdapter) IsConnected() bool                              { return f.connected }
func (f *fakeAdapter) GetLastSendTime() time.Time                     { return f.lastSend }
func (f *fakeAdapter) PendingCommandCount() int                       { return len(f.commands) }
func (f *fakeAdapter) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"name":           f.name,
		"type":          "fake",
		"enabled":       f.enabled,
		"connected":     f.connected,
		"pending_data":  0,
		"pending_alarm": 0,
	}
}

func (f *fakeAdapter) Send(data *models.CollectData) error {
	atomic.AddInt32(&f.sendCalls, 1)
	if f.fail {
		return errors.New("send error")
	}
	return nil
}

func (f *fakeAdapter) SendAlarm(alarm *models.AlarmPayload) error {
	atomic.AddInt32(&f.alarmCalls, 1)
	if f.fail {
		return errors.New("alarm error")
	}
	return nil
}

func (f *fakeAdapter) PullCommands(limit int) ([]*models.NorthboundCommand, error) {
	if f.fail {
		return nil, errors.New("pull error")
	}
	if len(f.commands) == 0 {
		return nil, nil
	}
	count := limit
	if count > len(f.commands) {
		count = len(f.commands)
	}
	out := f.commands[:count]
	f.commands = f.commands[count:]
	return out, nil
}

func (f *fakeAdapter) ReportCommandResult(result *models.NorthboundCommandResult) error {
	atomic.AddInt32(&f.reportCalls, 1)
	f.lastResult = result
	if f.fail {
		return errors.New("report error")
	}
	return nil
}

// 确保 fakeAdapter 实现 adapters.NorthboundAdapter
var _ adapters.NorthboundAdapter = (*fakeAdapter)(nil)
var _ adapters.NorthboundAdapterWithCommands = (*fakeAdapter)(nil)

func TestNewNorthboundManager(t *testing.T) {
	mgr := NewNorthboundManager()
	if mgr == nil {
		t.Fatal("expected non-nil manager")
	}
}

func TestNorthboundManager_RegisterAndUnregisterAdapter(t *testing.T) {
	mgr := NewNorthboundManager()
	adapter := &fakeAdapter{name: "a1"}

	mgr.RegisterAdapter("a1", adapter)

	if !mgr.HasAdapter("a1") {
		t.Fatalf("expected adapter a1 to be registered")
	}
	if mgr.GetAdapterCount() != 1 {
		t.Fatalf("expected 1 adapter, got %d", mgr.GetAdapterCount())
	}

	// 注销适配器
	mgr.RemoveAdapter("a1")
	if mgr.HasAdapter("a1") {
		t.Fatalf("expected adapter a1 to be removed")
	}
	if mgr.GetAdapterCount() != 0 {
		t.Fatalf("expected 0 adapter after remove, got %d", mgr.GetAdapterCount())
	}
}

func TestNorthboundManager_SendData(t *testing.T) {
	mgr := NewNorthboundManager()
	adapter := &fakeAdapter{name: "a1"}
	adapter.enabled = true
	mgr.RegisterAdapter("a1", adapter)

	data := &models.CollectData{DeviceID: 1}
	mgr.SendData(data)

	// 由于是内置适配器，Send 应该被调用
	if atomic.LoadInt32(&adapter.sendCalls) != 1 {
		t.Fatalf("expected Send to be called once")
	}
}

func TestNorthboundManager_SendAlarm(t *testing.T) {
	mgr := NewNorthboundManager()
	adapter := &fakeAdapter{name: "a1"}
	adapter.enabled = true
	mgr.RegisterAdapter("a1", adapter)

	alarm := &models.AlarmPayload{DeviceID: 1}
	mgr.SendAlarm(alarm)

	if atomic.LoadInt32(&adapter.alarmCalls) != 1 {
		t.Fatalf("expected SendAlarm to be called once")
	}
}

func TestNorthboundManager_Intervals(t *testing.T) {
	mgr := NewNorthboundManager()
	adapter := &fakeAdapter{name: "a1"}
	mgr.RegisterAdapter("a1", adapter)

	if mgr.IsEnabled("a1") != true {
		t.Fatalf("expected adapter to be enabled by default")
	}

	mgr.SetInterval("a1", 50*time.Millisecond)
	if mgr.GetInterval("a1") < 50*time.Millisecond {
		t.Fatalf("expected interval to be at least 50ms")
	}

	names := mgr.ListRuntimeNames()
	found := false
	for _, n := range names {
		if n == "a1" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected a1 in runtime names")
	}
}

func TestNorthboundManager_PullCommands(t *testing.T) {
	mgr := NewNorthboundManager()
	adapter := &fakeAdapter{
		name: "a1",
		commands: []*models.NorthboundCommand{
			{RequestID: "r1", ProductKey: "pk", DeviceKey: "dk", FieldName: "temp", Value: "25"},
			{RequestID: "r2", ProductKey: "pk", DeviceKey: "dk", FieldName: "hum", Value: "60"},
		},
	}
	adapter.enabled = true
	mgr.RegisterAdapter("a1", adapter)

	cmds, err := mgr.PullCommands(10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cmds) != 2 {
		t.Fatalf("expected 2 commands, got %d", len(cmds))
	}
	if cmds[0].RequestID != "r1" || cmds[1].RequestID != "r2" {
		t.Fatalf("unexpected command order")
	}

	// 再次拉取应为空
	cmds, err = mgr.PullCommands(10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cmds) != 0 {
		t.Fatalf("expected 0 commands after drain, got %d", len(cmds))
	}
}

func TestNorthboundManager_PullCommands_DisabledAdapter(t *testing.T) {
	mgr := NewNorthboundManager()
	adapter := &fakeAdapter{
		name: "a1",
		commands: []*models.NorthboundCommand{
			{RequestID: "r1"},
		},
	}
	mgr.RegisterAdapter("a1", adapter)
	mgr.SetEnabled("a1", false)

	cmds, err := mgr.PullCommands(10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cmds) != 0 {
		t.Fatalf("expected 0 commands from disabled adapter, got %d", len(cmds))
	}
}

func TestNorthboundManager_ReportCommandResult(t *testing.T) {
	mgr := NewNorthboundManager()
	adapter := &fakeAdapter{name: "a1"}
	adapter.enabled = true
	mgr.RegisterAdapter("a1", adapter)

	result := &models.NorthboundCommandResult{
		RequestID:  "r1",
		ProductKey: "pk",
		DeviceKey:  "dk",
		FieldName:  "temp",
		Value:      "25",
		Success:    true,
		Code:       200,
	}
	mgr.ReportCommandResult(result)

	if atomic.LoadInt32(&adapter.reportCalls) != 1 {
		t.Fatalf("expected ReportCommandResult to be called once, got %d", atomic.LoadInt32(&adapter.reportCalls))
	}
	if adapter.lastResult == nil || adapter.lastResult.RequestID != "r1" {
		t.Fatalf("expected result with request_id=r1")
	}
}

func TestNorthboundManager_ReportCommandResult_DisabledSkipped(t *testing.T) {
	mgr := NewNorthboundManager()
	adapter := &fakeAdapter{name: "a1"}
	mgr.RegisterAdapter("a1", adapter)
	mgr.SetEnabled("a1", false)

	result := &models.NorthboundCommandResult{RequestID: "r1", Success: true}
	mgr.ReportCommandResult(result)

	if atomic.LoadInt32(&adapter.reportCalls) != 0 {
		t.Fatalf("expected ReportCommandResult not called for disabled adapter")
	}
}

func TestNorthboundManager_StartStop(t *testing.T) {
	mgr := NewNorthboundManager()
	adapter := &fakeAdapter{name: "a1"}
	mgr.RegisterAdapter("a1", adapter)

	// Start 应该不会自动启动适配器（需要 SetEnabled）
	mgr.Start()
	// 停止管理器应该停止所有适配器
	mgr.Stop()

	if adapter.enabled {
		t.Fatalf("expected adapter to be stopped")
	}
}

func TestNorthboundManager_GetStats(t *testing.T) {
	mgr := NewNorthboundManager()
	adapter := &fakeAdapter{name: "a1", enabled: true, connected: true}
	mgr.RegisterAdapter("a1", adapter)

	stats := mgr.GetStats()
	if stats == nil {
		t.Fatal("expected non-nil stats")
	}
	if _, ok := stats["a1"]; !ok {
		t.Fatalf("expected stats for adapter a1")
	}
}

// 保留熔断器相关测试（向后兼容）
func TestNorthboundManager_BreakerState(t *testing.T) {
	mgr := NewNorthboundManager()
	adapter := &fakeAdapter{name: "a1"}
	mgr.RegisterAdapter("a1", adapter)

	// 检查熔断器状态
	state := mgr.GetBreakerState("a1")
	if state != circuit.Closed {
		t.Fatalf("expected breaker state closed, got %v", state)
	}
}
