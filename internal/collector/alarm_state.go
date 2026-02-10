package collector

import (
	"sync"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/models"
)

const defaultAlarmRepeatInterval = time.Minute

type alarmStateKey struct {
	DeviceID       int64
	ThresholdID    int64
	FieldName      string
	Operator       string
	ThresholdValue float64
}

type alarmState struct {
	LastTriggered time.Time
}

var alarmStates = struct {
	mu   sync.Mutex
	data map[alarmStateKey]alarmState
}{
	data: make(map[alarmStateKey]alarmState),
}

func buildAlarmStateKey(deviceID int64, threshold *models.Threshold) alarmStateKey {
	if threshold == nil {
		return alarmStateKey{DeviceID: deviceID}
	}
	return alarmStateKey{
		DeviceID:       deviceID,
		ThresholdID:    threshold.ID,
		FieldName:      threshold.FieldName,
		Operator:       threshold.Operator,
		ThresholdValue: threshold.Value,
	}
}

// shouldEmitAlarm 更新阈值状态并返回是否应当发出新报警。
// 规则：
// 1) 首次进入报警态立即触发
// 2) 持续报警态按 repeatInterval 限频重发
// 3) 未命中阈值时退出报警态
func shouldEmitAlarm(deviceID int64, threshold *models.Threshold, matched bool, now time.Time, repeatInterval time.Duration) bool {
	if threshold == nil {
		return false
	}
	if now.IsZero() {
		now = time.Now()
	}
	if repeatInterval <= 0 {
		repeatInterval = defaultAlarmRepeatInterval
	}

	key := buildAlarmStateKey(deviceID, threshold)

	alarmStates.mu.Lock()
	defer alarmStates.mu.Unlock()

	state, exists := alarmStates.data[key]

	if !matched {
		delete(alarmStates.data, key)
		return false
	}

	emit := false
	if !exists {
		emit = true
	} else if state.LastTriggered.IsZero() || now.Sub(state.LastTriggered) >= repeatInterval {
		emit = true
	}

	if emit {
		state.LastTriggered = now
	}
	alarmStates.data[key] = state

	return emit
}

func clearAlarmStateForDevice(deviceID int64) {
	alarmStates.mu.Lock()
	defer alarmStates.mu.Unlock()

	for key := range alarmStates.data {
		if key.DeviceID == deviceID {
			delete(alarmStates.data, key)
		}
	}
}
