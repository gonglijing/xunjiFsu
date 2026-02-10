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

type alarmState struct {
	LastTriggered time.Time
}

var alarmStates = struct {
	mu   sync.Mutex
	data map[alarmStateKey]alarmState
}{
	data: make(map[alarmStateKey]alarmState),
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

func resolveAlarmRepeatInterval() time.Duration {
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
