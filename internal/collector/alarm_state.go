package collector

import (
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/database"
	"github.com/gonglijing/xunjiFsu/internal/models"
)

const (
	defaultAlarmRepeatInterval       = time.Minute
	alarmRepeatIntervalRefreshWindow = 5 * time.Second
	defaultAlarmStateTTL             = 24 * time.Hour
	maxAlarmStateTTL                 = 7 * 24 * time.Hour
	alarmStatePruneEveryCalls        = 512
)

type alarmStateKey struct {
	DeviceID       int64
	ThresholdID    int64
	FieldName      string
	Operator       string
	ThresholdValue float64
}

type alarmStateIDKey struct {
	DeviceID    int64
	ThresholdID int64
}

type alarmState struct {
	LastTriggered time.Time
}

var alarmStates = struct {
	mu   sync.Mutex
	data map[alarmStateKey]alarmState
	byID map[alarmStateIDKey]alarmState
}{
	data: make(map[alarmStateKey]alarmState),
	byID: make(map[alarmStateIDKey]alarmState),
}

var alarmStateCheckCounter uint64

var alarmRepeatIntervalCache = struct {
	valueNS     int64
	expiresAtNS int64
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
		FieldName:      strings.ToLower(strings.TrimSpace(threshold.FieldName)),
		Operator:       strings.TrimSpace(threshold.Operator),
		ThresholdValue: threshold.Value,
	}
}

func alarmStateIDFromKey(key alarmStateKey) (alarmStateIDKey, bool) {
	if key.ThresholdID <= 0 {
		return alarmStateIDKey{}, false
	}
	return alarmStateIDKey{
		DeviceID:    key.DeviceID,
		ThresholdID: key.ThresholdID,
	}, true
}

func getAlarmStateLocked(key alarmStateKey) (alarmState, bool) {
	if idKey, ok := alarmStateIDFromKey(key); ok {
		state, exists := alarmStates.byID[idKey]
		return state, exists
	}
	state, exists := alarmStates.data[key]
	return state, exists
}

func setAlarmStateLocked(key alarmStateKey, state alarmState) {
	if idKey, ok := alarmStateIDFromKey(key); ok {
		alarmStates.byID[idKey] = state
		return
	}
	alarmStates.data[key] = state
}

func hasAlarmStateLocked(key alarmStateKey) bool {
	if idKey, ok := alarmStateIDFromKey(key); ok {
		_, exists := alarmStates.byID[idKey]
		return exists
	}
	_, exists := alarmStates.data[key]
	return exists
}

func resolveAlarmRepeatInterval() time.Duration {
	if database.ParamDB == nil {
		return defaultAlarmRepeatInterval
	}

	nowNS := time.Now().UnixNano()

	cachedNS := atomic.LoadInt64(&alarmRepeatIntervalCache.valueNS)
	expiresAtNS := atomic.LoadInt64(&alarmRepeatIntervalCache.expiresAtNS)
	if cachedNS > 0 && nowNS < expiresAtNS {
		return time.Duration(cachedNS)
	}

	resolved := defaultAlarmRepeatInterval
	seconds, err := database.GetAlarmRepeatIntervalSeconds()
	if err != nil {
		log.Printf("Failed to load alarm repeat interval: %v", err)
	} else if seconds > 0 {
		resolved = time.Duration(seconds) * time.Second
	}

	atomic.StoreInt64(&alarmRepeatIntervalCache.valueNS, int64(resolved))
	atomic.StoreInt64(&alarmRepeatIntervalCache.expiresAtNS, nowNS+int64(alarmRepeatIntervalRefreshWindow))

	return resolved
}

func resolveAlarmStateTTL(repeatInterval time.Duration) time.Duration {
	ttl := defaultAlarmStateTTL
	if repeatInterval > 0 {
		candidate := repeatInterval * 120
		if candidate > ttl {
			ttl = candidate
		}
	}
	if ttl > maxAlarmStateTTL {
		ttl = maxAlarmStateTTL
	}
	return ttl
}

func maybePruneAlarmStates(now time.Time, repeatInterval time.Duration) {
	if atomic.AddUint64(&alarmStateCheckCounter, 1)%alarmStatePruneEveryCalls != 0 {
		return
	}

	ttl := resolveAlarmStateTTL(repeatInterval)
	alarmStates.mu.Lock()
	_ = pruneAlarmStatesLocked(now, ttl)
	alarmStates.mu.Unlock()
}

func pruneAlarmStatesLocked(now time.Time, ttl time.Duration) int {
	if now.IsZero() {
		now = time.Now()
	}
	if ttl <= 0 {
		ttl = defaultAlarmStateTTL
	}

	removed := 0
	for key, state := range alarmStates.data {
		if state.LastTriggered.IsZero() || now.Sub(state.LastTriggered) > ttl {
			delete(alarmStates.data, key)
			removed++
		}
	}
	for key, state := range alarmStates.byID {
		if state.LastTriggered.IsZero() || now.Sub(state.LastTriggered) > ttl {
			delete(alarmStates.byID, key)
			removed++
		}
	}
	return removed
}

// InvalidateAlarmRepeatIntervalCache 清理重复上报间隔缓存
func InvalidateAlarmRepeatIntervalCache() {
	atomic.StoreInt64(&alarmRepeatIntervalCache.valueNS, 0)
	atomic.StoreInt64(&alarmRepeatIntervalCache.expiresAtNS, 0)
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

	maybePruneAlarmStates(now, repeatInterval)

	if !matched {
		return false
	}

	key := buildAlarmStateKey(deviceID, threshold)
	return shouldEmitAlarmForKey(key, now, repeatInterval)
}

func shouldEmitAlarmForKey(key alarmStateKey, now time.Time, repeatInterval time.Duration) bool {
	if now.IsZero() {
		now = time.Now()
	}
	if repeatInterval <= 0 {
		repeatInterval = defaultAlarmRepeatInterval
	}

	if idKey, ok := alarmStateIDFromKey(key); ok {
		return shouldEmitAlarmForIDKey(idKey, now, repeatInterval)
	}

	alarmStates.mu.Lock()
	defer alarmStates.mu.Unlock()

	state, exists := getAlarmStateLocked(key)

	emit := false
	if !exists {
		emit = true
	} else if state.LastTriggered.IsZero() || now.Sub(state.LastTriggered) >= repeatInterval {
		emit = true
	}

	if !emit {
		return false
	}

	state.LastTriggered = now
	setAlarmStateLocked(key, state)

	return true
}

func shouldEmitAlarmForIDKey(key alarmStateIDKey, now time.Time, repeatInterval time.Duration) bool {
	if now.IsZero() {
		now = time.Now()
	}
	if repeatInterval <= 0 {
		repeatInterval = defaultAlarmRepeatInterval
	}

	alarmStates.mu.Lock()
	defer alarmStates.mu.Unlock()

	state, exists := alarmStates.byID[key]
	if exists && !state.LastTriggered.IsZero() && now.Sub(state.LastTriggered) < repeatInterval {
		return false
	}

	state.LastTriggered = now
	alarmStates.byID[key] = state
	return true
}

func clearAlarmStateForDevice(deviceID int64) {
	alarmStates.mu.Lock()
	defer alarmStates.mu.Unlock()

	for key := range alarmStates.data {
		if key.DeviceID == deviceID {
			delete(alarmStates.data, key)
		}
	}
	for key := range alarmStates.byID {
		if key.DeviceID == deviceID {
			delete(alarmStates.byID, key)
		}
	}
}
