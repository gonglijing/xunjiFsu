package collector

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/models"
)

func TestShouldEmitAlarm_EdgeAndRateLimit(t *testing.T) {
	threshold := &models.Threshold{
		ID:        1001,
		FieldName: "humidity",
		Operator:  ">",
		Value:     50,
	}
	deviceID := int64(10)
	clearAlarmStateForDevice(deviceID)
	defer clearAlarmStateForDevice(deviceID)

	now := time.Now()
	if !shouldEmitAlarm(deviceID, threshold, true, now, time.Minute) {
		t.Fatal("first matched state should emit alarm")
	}
	if shouldEmitAlarm(deviceID, threshold, true, now.Add(10*time.Second), time.Minute) {
		t.Fatal("within repeat interval should not emit again")
	}
	if !shouldEmitAlarm(deviceID, threshold, true, now.Add(61*time.Second), time.Minute) {
		t.Fatal("after repeat interval should emit again")
	}

	if shouldEmitAlarm(deviceID, threshold, false, now.Add(70*time.Second), time.Minute) {
		t.Fatal("unmatched state should not emit")
	}

	key := buildAlarmStateKey(deviceID, threshold)
	alarmStates.mu.Lock()
	_, exists := alarmStates.data[key]
	alarmStates.mu.Unlock()
	if !exists {
		t.Fatal("state should be retained after recovery for repeat suppression")
	}
	if shouldEmitAlarm(deviceID, threshold, true, now.Add(71*time.Second), time.Minute) {
		t.Fatal("matched again within repeat interval should still be suppressed")
	}
	if !shouldEmitAlarm(deviceID, threshold, true, now.Add(122*time.Second), time.Minute) {
		t.Fatal("matched again after repeat interval should emit")
	}
}

func TestBuildAlarmStateKey_UseThresholdIDAsPrimaryIdentity(t *testing.T) {
	deviceID := int64(99)
	threshold := &models.Threshold{
		ID:        3001,
		FieldName: "temp",
		Operator:  ">",
		Value:     30,
	}

	first := buildAlarmStateKey(deviceID, threshold)

	threshold.FieldName = "temperature"
	threshold.Operator = ">="
	threshold.Value = 31

	second := buildAlarmStateKey(deviceID, threshold)

	if first != second {
		t.Fatalf("same threshold id should produce same alarm key, first=%+v second=%+v", first, second)
	}

	if first.ThresholdID != threshold.ID {
		t.Fatalf("expected threshold id in key, got %d want %d", first.ThresholdID, threshold.ID)
	}

	if first.FieldName != "" || first.Operator != "" || first.ThresholdValue != 0 {
		t.Fatalf("id-based key should not depend on field/operator/value, got %+v", first)
	}
}

func TestBuildAlarmStateKey_WithoutIDNormalizesFieldAndOperator(t *testing.T) {
	deviceID := int64(101)
	thresholdA := &models.Threshold{
		FieldName: " Humidity ",
		Operator:  " > ",
		Value:     50,
	}
	thresholdB := &models.Threshold{
		FieldName: "humidity",
		Operator:  ">",
		Value:     50,
	}

	keyA := buildAlarmStateKey(deviceID, thresholdA)
	keyB := buildAlarmStateKey(deviceID, thresholdB)

	if keyA != keyB {
		t.Fatalf("normalized keys should match, keyA=%+v keyB=%+v", keyA, keyB)
	}
	if keyA.FieldName != "humidity" || keyA.Operator != ">" {
		t.Fatalf("expected normalized key fields, got %+v", keyA)
	}
}

func TestPruneAlarmStatesLocked(t *testing.T) {
	deviceID := int64(77)
	threshold := &models.Threshold{ID: 7001}
	key := buildAlarmStateKey(deviceID, threshold)

	alarmStates.mu.Lock()
	alarmStates.data[key] = alarmState{LastTriggered: time.Now().Add(-2 * time.Hour)}
	alarmStates.mu.Unlock()
	defer clearAlarmStateForDevice(deviceID)

	alarmStates.mu.Lock()
	removed := pruneAlarmStatesLocked(time.Now(), time.Hour)
	_, exists := alarmStates.data[key]
	alarmStates.mu.Unlock()

	if removed == 0 {
		t.Fatal("expected at least one stale state to be removed")
	}
	if exists {
		t.Fatal("stale alarm state should be removed")
	}
}

func TestResolveAlarmStateTTL(t *testing.T) {
	if ttl := resolveAlarmStateTTL(0); ttl != defaultAlarmStateTTL {
		t.Fatalf("ttl=%v, want %v", ttl, defaultAlarmStateTTL)
	}
	if ttl := resolveAlarmStateTTL(2 * time.Hour); ttl != maxAlarmStateTTL {
		t.Fatalf("ttl=%v, want capped %v", ttl, maxAlarmStateTTL)
	}
}

func TestInvalidateAlarmRepeatIntervalCache(t *testing.T) {
	atomic.StoreInt64(&alarmRepeatIntervalCache.valueNS, int64(30*time.Second))
	atomic.StoreInt64(&alarmRepeatIntervalCache.expiresAtNS, time.Now().Add(time.Minute).UnixNano())

	InvalidateAlarmRepeatIntervalCache()

	if got := atomic.LoadInt64(&alarmRepeatIntervalCache.valueNS); got != 0 {
		t.Fatalf("expected valueNS=0, got %d", got)
	}
	if got := atomic.LoadInt64(&alarmRepeatIntervalCache.expiresAtNS); got != 0 {
		t.Fatalf("expected expiresAtNS=0, got %d", got)
	}
}
