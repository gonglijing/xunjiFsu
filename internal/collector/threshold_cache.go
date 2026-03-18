package collector

import (
	"log"
	"strings"
	"sync"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/database"
	"github.com/gonglijing/xunjiFsu/internal/models"
)

// thresholdCache 阈值缓存
type thresholdCache struct {
	mu          sync.RWMutex
	thresholds  map[int64][]*models.Threshold // deviceID -> thresholds
	rules       map[int64][]thresholdEvalRule
	lastRefresh time.Time
	interval    time.Duration
	stopChan    chan struct{}
	running     bool
}

type thresholdEvalRule struct {
	threshold           *models.Threshold
	fieldName           string
	normalizedFieldName string
	operator            string
	alarmKey            alarmStateKey
	alarmIDKey          alarmStateIDKey
	hasAlarmIDKey       bool
	shielded            bool
	thresholdValue      float64
}

// 缓存实例
var cache *thresholdCache

// init 初始化阈值缓存
func init() {
	cache = &thresholdCache{
		thresholds:  make(map[int64][]*models.Threshold),
		rules:       make(map[int64][]thresholdEvalRule),
		interval:    time.Minute, // 缓存刷新间隔
		lastRefresh: time.Time{},
	}
}

// StartThresholdCache 启动阈值缓存刷新任务
func StartThresholdCache() {
	cache.mu.Lock()
	if cache.running {
		cache.mu.Unlock()
		return
	}
	stopChan := make(chan struct{})
	cache.stopChan = stopChan
	cache.running = true
	cache.mu.Unlock()

	cache.Refresh()
	go cache.refreshLoop(stopChan)
	log.Println("Threshold cache started")
}

// StopThresholdCache 停止阈值缓存刷新任务
func StopThresholdCache() {
	cache.mu.Lock()
	if !cache.running {
		cache.mu.Unlock()
		return
	}
	stopChan := cache.stopChan
	cache.stopChan = nil
	cache.running = false
	cache.mu.Unlock()

	close(stopChan)
	log.Println("Threshold cache stopped")
}

// refreshLoop 定期刷新缓存
func (c *thresholdCache) refreshLoop(stopChan <-chan struct{}) {
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	for {
		select {
		case <-stopChan:
			return
		case <-ticker.C:
			c.Refresh()
		}
	}
}

// Refresh 刷新所有阈值缓存
func (c *thresholdCache) Refresh() {
	// 获取所有设备
	devices, err := database.GetAllDevices()
	if err != nil {
		log.Printf("Failed to refresh threshold cache: %v", err)
		return
	}
	thresholds, err := database.GetAllThresholds()
	if err != nil {
		log.Printf("Failed to refresh threshold cache thresholds: %v", err)
		return
	}

	validDeviceIDs := make(map[int64]struct{}, len(devices))
	for _, device := range devices {
		if device == nil || device.ID == 0 {
			continue
		}
		validDeviceIDs[device.ID] = struct{}{}
	}
	next := make(map[int64][]*models.Threshold, len(validDeviceIDs))
	nextRules := make(map[int64][]thresholdEvalRule, len(validDeviceIDs))

	for _, threshold := range thresholds {
		if threshold == nil || threshold.DeviceID == 0 {
			continue
		}
		if _, exists := validDeviceIDs[threshold.DeviceID]; !exists {
			continue
		}
		normalizeThresholdForRuntime(threshold)
		next[threshold.DeviceID] = append(next[threshold.DeviceID], threshold)
		nextRules[threshold.DeviceID] = append(nextRules[threshold.DeviceID], buildThresholdEvalRule(threshold))
	}

	c.mu.Lock()
	c.thresholds = next
	c.rules = nextRules
	c.lastRefresh = time.Now()
	c.mu.Unlock()

	log.Printf("Threshold cache refreshed, %d devices with thresholds, %d thresholds", len(next), len(thresholds))
}

func getDeviceThresholdRules(deviceID int64) ([]thresholdEvalRule, error) {
	_, rules, err := getThresholdCacheEntry(deviceID)
	return rules, err
}

func getThresholdCacheEntry(deviceID int64) ([]*models.Threshold, []thresholdEvalRule, error) {
	cache.mu.RLock()
	thresholds, exists := cache.thresholds[deviceID]
	rules := cache.rules[deviceID]
	needsRefresh := time.Since(cache.lastRefresh) > cache.interval*2 // 超过2倍刷新间隔需要刷新
	cache.mu.RUnlock()

	if needsRefresh {
		cache.Refresh()
		cache.mu.RLock()
		thresholds, exists = cache.thresholds[deviceID]
		rules = cache.rules[deviceID]
		cache.mu.RUnlock()
	}

	if !exists {
		loaded, err := database.GetThresholdsByDeviceID(deviceID)
		if err != nil {
			return nil, nil, err
		}
		if len(loaded) > 0 {
			loadedRules := make([]thresholdEvalRule, 0, len(loaded))
			for _, threshold := range loaded {
				if threshold == nil {
					continue
				}
				normalizeThresholdForRuntime(threshold)
				loadedRules = append(loadedRules, buildThresholdEvalRule(threshold))
			}
			cache.mu.Lock()
			cache.thresholds[deviceID] = loaded
			cache.rules[deviceID] = loadedRules
			cache.mu.Unlock()
			return loaded, loadedRules, nil
		}
		return loaded, nil, nil
	}

	if len(thresholds) > 0 && len(rules) == 0 {
		rebuiltRules := make([]thresholdEvalRule, 0, len(thresholds))
		for _, threshold := range thresholds {
			if threshold == nil {
				continue
			}
			normalizeThresholdForRuntime(threshold)
			rebuiltRules = append(rebuiltRules, buildThresholdEvalRule(threshold))
		}
		cache.mu.Lock()
		cache.rules[deviceID] = rebuiltRules
		cache.mu.Unlock()
		rules = rebuiltRules
	}

	return thresholds, rules, nil
}

// InvalidateDeviceCache 使指定设备的缓存失效
func InvalidateDeviceCache(deviceID int64) {
	cache.mu.Lock()
	delete(cache.thresholds, deviceID)
	delete(cache.rules, deviceID)
	cache.mu.Unlock()
}

func normalizeThresholdForRuntime(threshold *models.Threshold) {
	if threshold == nil {
		return
	}
	threshold.FieldName = strings.TrimSpace(threshold.FieldName)
}

func buildThresholdEvalRule(threshold *models.Threshold) thresholdEvalRule {
	fieldName := ""
	normalized := ""
	operator := ""
	thresholdValue := 0.0
	shielded := false
	if threshold != nil {
		fieldName = strings.TrimSpace(threshold.FieldName)
		normalized = normalizeFieldName(fieldName)
		operator = strings.TrimSpace(threshold.Operator)
		thresholdValue = threshold.Value
		shielded = threshold.Shielded == 1
	}
	alarmKey := buildAlarmStateKey(0, threshold)
	alarmIDKey, hasAlarmIDKey := alarmStateIDFromKey(alarmKey)
	return thresholdEvalRule{
		threshold:           threshold,
		fieldName:           fieldName,
		normalizedFieldName: normalized,
		operator:            operator,
		alarmKey:            alarmKey,
		alarmIDKey:          alarmIDKey,
		hasAlarmIDKey:       hasAlarmIDKey,
		shielded:            shielded,
		thresholdValue:      thresholdValue,
	}
}
