package collector

import (
	"log"
	"sync"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/database"
	"github.com/gonglijing/xunjiFsu/internal/models"
)

// thresholdCache 阈值缓存
type thresholdCache struct {
	mu          sync.RWMutex
	thresholds  map[int64][]*models.Threshold // deviceID -> thresholds
	lastRefresh time.Time
	interval    time.Duration
	stopChan    chan struct{}
	running     bool
}

// 缓存实例
var cache *thresholdCache

// init 初始化阈值缓存
func init() {
	cache = &thresholdCache{
		thresholds:  make(map[int64][]*models.Threshold),
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

	for _, threshold := range thresholds {
		if threshold == nil || threshold.DeviceID == 0 {
			continue
		}
		if _, exists := validDeviceIDs[threshold.DeviceID]; !exists {
			continue
		}
		next[threshold.DeviceID] = append(next[threshold.DeviceID], threshold)
	}

	c.mu.Lock()
	c.thresholds = next
	c.lastRefresh = time.Now()
	c.mu.Unlock()

	log.Printf("Threshold cache refreshed, %d devices with thresholds, %d thresholds", len(next), len(thresholds))
}

// GetDeviceThresholds 获取设备的阈值配置（优先从缓存获取）
func GetDeviceThresholds(deviceID int64) ([]*models.Threshold, error) {
	cache.mu.RLock()
	thresholds, exists := cache.thresholds[deviceID]
	needsRefresh := time.Since(cache.lastRefresh) > cache.interval*2 // 超过2倍刷新间隔需要刷新
	cache.mu.RUnlock()

	if needsRefresh {
		cache.Refresh()
		cache.mu.RLock()
		thresholds, exists = cache.thresholds[deviceID]
		cache.mu.RUnlock()
	}

	if !exists {
		loaded, err := database.GetThresholdsByDeviceID(deviceID)
		if err != nil {
			return nil, err
		}
		if len(loaded) > 0 {
			cache.mu.Lock()
			cache.thresholds[deviceID] = loaded
			cache.mu.Unlock()
		}
		return loaded, nil
	}

	return thresholds, nil
}

// InvalidateDeviceCache 使指定设备的缓存失效
func InvalidateDeviceCache(deviceID int64) {
	cache.mu.Lock()
	delete(cache.thresholds, deviceID)
	cache.mu.Unlock()
}

// InvalidateAllCache 使所有缓存失效
func InvalidateAllCache() {
	cache.mu.Lock()
	cache.thresholds = make(map[int64][]*models.Threshold)
	cache.lastRefresh = time.Time{}
	cache.mu.Unlock()
}
