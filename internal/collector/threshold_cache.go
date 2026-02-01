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
}

// 缓存实例
var cache *thresholdCache

// init 初始化阈值缓存
func init() {
	cache = &thresholdCache{
		thresholds:  make(map[int64][]*models.Threshold),
		interval:    time.Minute, // 缓存刷新间隔
		lastRefresh: time.Time{},
		stopChan:    make(chan struct{}),
	}
}

// StartThresholdCache 启动阈值缓存刷新任务
func StartThresholdCache() {
	go cache.refreshLoop()
	log.Println("Threshold cache started")
}

// StopThresholdCache 停止阈值缓存刷新任务
func StopThresholdCache() {
	close(cache.stopChan)
	log.Println("Threshold cache stopped")
}

// refreshLoop 定期刷新缓存
func (c *thresholdCache) refreshLoop() {
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	for {
		select {
		case <-c.stopChan:
			return
		case <-ticker.C:
			c.Refresh()
		}
	}
}

// Refresh 刷新所有阈值缓存
func (c *thresholdCache) Refresh() {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 获取所有设备
	devices, err := database.GetAllDevices()
	if err != nil {
		log.Printf("Failed to refresh threshold cache: %v", err)
		return
	}

	for _, device := range devices {
		thresholds, err := database.GetEnabledThresholdsByDeviceID(device.ID)
		if err != nil {
			log.Printf("Failed to load thresholds for device %d: %v", device.ID, err)
			continue
		}
		c.thresholds[device.ID] = thresholds
	}

	c.lastRefresh = time.Now()
	log.Printf("Threshold cache refreshed, %d devices", len(c.thresholds))
}

// GetDeviceThresholds 获取设备的阈值配置（优先从缓存获取）
func GetDeviceThresholds(deviceID int64) ([]*models.Threshold, error) {
	cache.mu.RLock()
	thresholds, exists := cache.thresholds[deviceID]
	needsRefresh := time.Since(cache.lastRefresh) > cache.interval*2 // 超过2倍刷新间隔需要刷新
	cache.mu.RUnlock()

	// 如果缓存不存在或需要刷新，从数据库加载
	if !exists || needsRefresh {
		cache.mu.Lock()
		// 双重检查
		if thresholds, exists = cache.thresholds[deviceID]; !exists {
			thresholds, err := database.GetEnabledThresholdsByDeviceID(deviceID)
			if err != nil {
				cache.mu.Unlock()
				return nil, err
			}
			cache.thresholds[deviceID] = thresholds
		}
		cache.mu.Unlock()
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
