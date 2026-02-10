package collector

import (
	"log"
	"sync"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/database"
	"github.com/gonglijing/xunjiFsu/internal/models"
)

const (
	defaultAlarmRepeatInterval       = time.Minute
	alarmRepeatIntervalRefreshWindow = 5 * time.Second
)

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

var alarmRepeatIntervalCache = struct {
	mu        sync.Mutex
	value     time.Duration
	expiresAt time.Time
}{}

func buildAlarmStateKey(deviceID int64, threshold *models.Threshold) alarmStateKey {
	if threshold == nil {
		return alarmStateKey{DeviceID: deviceID}
	}
	if threshold.ID > 0 {
		return alarmStateKey{
			DeviceID:    deviceID,
			ThresholdID: threshold.ID,
		}
	}
	return alarmStateKey{
		DeviceID:       deviceID,
		FieldName:      threshold.FieldName,
		Operator:       threshold.Operator,
		ThresholdValue: threshold.Value,
	}
}

func resolveAlarmRepeatInterval() time.Duration {
	now := time.Now()

	alarmRepeatIntervalCache.mu.Lock()
	cached := alarmRepeatIntervalCache.value
	expiresAt := alarmRepeatIntervalCache.expiresAt
	alarmRepeatIntervalCache.mu.Unlock()

	if cached > 0 && now.Before(expiresAt) {
		return cached
	}

	resolved := defaultAlarmRepeatInterval
	seconds, err := database.GetAlarmRepeatIntervalSeconds()
	if err != nil {
		log.Printf("Failed to load alarm repeat interval: %v", err)
	} else if seconds > 0 {
		resolved = time.Duration(seconds) * time.Second
	}

	alarmRepeatIntervalCache.mu.Lock()
	alarmRepeatIntervalCache.value = resolved
	alarmRepeatIntervalCache.expiresAt = now.Add(alarmRepeatIntervalRefreshWindow)
	alarmRepeatIntervalCache.mu.Unlock()

	return resolved
}

// InvalidateAlarmRepeatIntervalCache 清理重复上报间隔缓存
func InvalidateAlarmRepeatIntervalCache() {
	alarmRepeatIntervalCache.mu.Lock()
	alarmRepeatIntervalCache.value = 0
	alarmRepeatIntervalCache.expiresAt = time.Time{}
	alarmRepeatIntervalCache.mu.Unlock()
}

// shouldEmitAlarm 更新阈值状态并返回是否应当发出新报警。
// 规则：
// 1) 首次进入报警态立即触发
// 2) 同一阈值按 repeatInterval 限频重发
// 3) 未命中阈值时不触发，且保留最后触发时间（保证窗口内最多一次）
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
