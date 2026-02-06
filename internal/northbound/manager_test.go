package northbound

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/circuit"
	"github.com/gonglijing/xunjiFsu/internal/models"
)

type fakeAdapter struct {
	name       string
	sendCalls  int32
	alarmCalls int32
	fail       bool
}

func (f *fakeAdapter) Initialize(config string) error { return nil }
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
func (f *fakeAdapter) Close() error { return nil }
func (f *fakeAdapter) Name() string { return f.name }

func TestNewNorthboundManager_AndPluginDir(t *testing.T) {
	mgr := NewNorthboundManager("/plugins")
	if mgr.PluginDir() != "/plugins" {
		t.Fatalf("expected plugin dir /plugins, got %s", mgr.PluginDir())
	}
}

func TestNorthboundManager_RegisterAndUnregisterAdapter(t *testing.T) {
	mgr := NewNorthboundManager("")
	adapter := &fakeAdapter{name: "a1"}

	mgr.RegisterAdapter("a1", adapter)

	if !mgr.HasAdapter("a1") {
		t.Fatalf("expected adapter a1 to be registered")
	}
	if mgr.GetAdapterCount() != 1 {
		t.Fatalf("expected 1 adapter, got %d", mgr.GetAdapterCount())
	}

	// 熔断器应已为该适配器创建
	if mgr.GetBreakerState("a1") != circuit.Closed {
		t.Fatalf("expected breaker state closed")
	}

	// Unregister / Remove 应清理适配器和熔断器
	mgr.RemoveAdapter("a1")
	if mgr.HasAdapter("a1") {
		t.Fatalf("expected adapter a1 to be removed")
	}
	if mgr.GetAdapterCount() != 0 {
		t.Fatalf("expected 0 adapter after remove, got %d", mgr.GetAdapterCount())
	}
}

func TestNorthboundManager_SendDataAndAlarm_WithBreaker(t *testing.T) {
	mgr := NewNorthboundManager("")
	adapter := &fakeAdapter{name: "a1"}
	mgr.RegisterAdapter("a1", adapter)
	mgr.SetInterval("a1", 0) // 立即发送
	mgr.SetEnabled("a1", true)

	data := &models.CollectData{DeviceID: 1}
	mgr.SendData(data)

	// 直接调用内部 uploadToAdapters 来避免依赖定时循环
	mgr.uploadToAdapters(data)

	if atomic.LoadInt32(&adapter.sendCalls) == 0 {
		t.Fatalf("expected Send to be called at least once")
	}

	alarm := &models.AlarmPayload{DeviceID: 1}
	mgr.SendAlarm(alarm)
	if atomic.LoadInt32(&adapter.alarmCalls) == 0 {
		t.Fatalf("expected SendAlarm to be called at least once")
	}
}

func TestNorthboundManager_IntervalsAndPending(t *testing.T) {
	mgr := NewNorthboundManager("")
	adapter := &fakeAdapter{name: "a1"}
	mgr.RegisterAdapter("a1", adapter)

	if mgr.IsEnabled("a1") != true {
		t.Fatalf("expected adapter to be enabled by default")
	}

	mgr.SetInterval("a1", 50*time.Millisecond)
	if mgr.GetInterval("a1") < 50*time.Millisecond {
		t.Fatalf("expected interval to be at least 50ms")
	}

	data := &models.CollectData{DeviceID: 1}
	mgr.SendData(data)
	if !mgr.HasPending("a1") {
		t.Fatalf("expected pending data after SendData")
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

