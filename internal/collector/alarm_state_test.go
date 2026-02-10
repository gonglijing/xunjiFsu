package collector

import (
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
	if exists {
		t.Fatal("state should be released when threshold recovers")
	}
	if !shouldEmitAlarm(deviceID, threshold, true, now.Add(71*time.Second), time.Minute) {
		t.Fatal("matched again after recovery should emit immediately")
	}
}
