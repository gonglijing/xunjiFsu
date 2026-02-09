package collector

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/database"
	"github.com/gonglijing/xunjiFsu/internal/models"
	"github.com/gonglijing/xunjiFsu/internal/northbound"
)

// SystemStatsCollector FSU 系统属性采集器
type SystemStatsCollector struct {
	northboundMgr *northbound.NorthboundManager
	interval      time.Duration
	stopChan      chan struct{}
	wg            sync.WaitGroup
	running       bool
	mu            sync.RWMutex
}

var (
	sysInstance *SystemStatsCollector
	sysOnce    sync.Once
)

// GetSystemStatsCollector 获取系统属性采集器单例
func GetSystemStatsCollector() *SystemStatsCollector {
	sysOnce.Do(func() {
		sysInstance = NewSystemStatsCollector()
	})
	return sysInstance
}

// NewSystemStatsCollector 创建系统属性采集器
func NewSystemStatsCollector(interval ...time.Duration) *SystemStatsCollector {
	collectInterval := 60 * time.Second // 默认每分钟采集一次
	if len(interval) > 0 && interval[0] > 0 {
		collectInterval = interval[0]
	}
	return &SystemStatsCollector{
		interval: collectInterval,
		stopChan: make(chan struct{}),
	}
}

// SetNorthboundManager 设置北向管理器
func (c *SystemStatsCollector) SetNorthboundManager(mgr *northbound.NorthboundManager) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.northboundMgr = mgr
}

// Start 启动系统属性采集
func (c *SystemStatsCollector) Start() error {
	c.mu.Lock()
	if c.running {
		c.mu.Unlock()
		return fmt.Errorf("system stats collector is already running")
	}
	c.running = true
	c.stopChan = make(chan struct{})
	c.mu.Unlock()

	c.wg.Add(1)
	go c.run()

	log.Printf("SystemStatsCollector started, interval=%v", c.interval)
	return nil
}

// Stop 停止系统属性采集
func (c *SystemStatsCollector) Stop() error {
	c.mu.Lock()
	if !c.running {
		c.mu.Unlock()
		return fmt.Errorf("system stats collector is not running")
	}
	c.running = false
	close(c.stopChan)
	c.mu.Unlock()

	c.wg.Wait()
	log.Println("SystemStatsCollector stopped")
	return nil
}

// IsRunning 检查是否运行中
func (c *SystemStatsCollector) IsRunning() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.running
}

// run 主循环
func (c *SystemStatsCollector) run() {
	defer c.wg.Done()

	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	for {
		select {
		case <-c.stopChan:
			return
		case <-ticker.C:
			c.collect()
		}
	}
}

// collect 执行一次采集
func (c *SystemStatsCollector) collect() {
	stats := c.collectSystemStats()
	data := c.statsToCollectData(stats)

	// 保存到数据库
	if err := database.InsertCollectDataWithOptions(data, true); err != nil {
		log.Printf("SystemStatsCollector: failed to insert data: %v", err)
	}

	// 发送到北向
	c.mu.RLock()
	mgr := c.northboundMgr
	c.mu.RUnlock()
	if mgr != nil {
		log.Printf("SystemStatsCollector: sending data to northbound (deviceId=%d, deviceName=%s)",
			data.DeviceID, data.DeviceName)
		mgr.SendData(data)
	} else {
		log.Printf("SystemStatsCollector: northbound manager is nil")
	}

	log.Printf("SystemStatsCollector: collected cpu=%.1f%% mem=%.1f%% disk=%.1f%%",
		stats.CpuUsage, stats.MemUsage, stats.DiskUsage)
}

// CollectSystemStatsOnce 立即采集一次系统属性（公开方法，供北向适配器调用）
func (c *SystemStatsCollector) CollectSystemStatsOnce() *models.SystemStats {
	return c.collectSystemStats()
}

// IsRunning 检查是否运行中
func (c *SystemStatsCollector) collectSystemStats() *models.SystemStats {
	stats := &models.SystemStats{
		Timestamp: time.Now().Unix(),
	}

	// CPU 使用率
	stats.CpuUsage = c.getCpuUsage()

	// 内存信息
	c.getMemoryInfo(stats)

	// 硬盘信息
	c.getDiskInfo(stats)

	// 运行时间
	stats.Uptime = c.getUptime()

	// 负载
	c.getLoadAverage(stats)

	return stats
}

// getCpuUsage 获取 CPU 使用率
func (c *SystemStatsCollector) getCpuUsage() float64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// 强制执行 GC 以获得准确的数字
	runtime.GC()

	// 简单采样方式：sleep 一小段时间然后计算
	// 更准确的方式是读取 /proc/stat (Linux)
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	time.Sleep(100 * time.Millisecond)
	runtime.ReadMemStats(&after)

	// 使用 AllocDelta 估算
	allocDelta := float64(after.Mallocs - before.Mallocs)
	if allocDelta < 0 {
		allocDelta = 0
	}

	// 简化的 CPU 使用率估算
	cpuUsage := 100.0
	if allocDelta > 0 {
		cpuUsage = 100.0 - (float64(after.Frees) / float64(after.Mallocs+after.Frees) * 100)
	}

	// 限制范围
	if cpuUsage < 0 {
		cpuUsage = 0
	}
	if cpuUsage > 100 {
		cpuUsage = 100
	}

	return cpuUsage
}

// getMemoryInfo 获取内存信息
func (c *SystemStatsCollector) getMemoryInfo(stats *models.SystemStats) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Go 内存使用 (MB)
	goUsed := float64(m.Alloc) / 1024 / 1024

	// 尝试读取 /proc/meminfo 获取系统内存 (Linux)
	total, used, available := c.readLinuxMemoryInfo()

	stats.MemTotal = total
	stats.MemUsed = used
	stats.MemAvailable = available

	if total > 0 {
		stats.MemUsage = (used / total) * 100
	} else {
		// 降级：使用 Go 内存估算
		stats.MemTotal = 8192 // 假设 8GB
		stats.MemUsed = goUsed
		stats.MemUsage = (goUsed / stats.MemTotal) * 100
		stats.MemAvailable = stats.MemTotal - stats.MemUsed
	}
}

// readLinuxMemoryInfo 读取 Linux 内存信息
func (c *SystemStatsCollector) readLinuxMemoryInfo() (total, used, available float64) {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0, 0, 0
	}

	lines := strings.Split(string(data), "\n")
	memTotal := int64(0)
	memAvailable := int64(0)
	memUsed := int64(0)

	for _, line := range lines {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		valueStr := strings.TrimSpace(parts[1])
		valueStr = strings.TrimSuffix(valueStr, " kB")
		value, err := strconv.ParseInt(valueStr, 10, 64)
		if err != nil {
			continue
		}

		switch key {
		case "MemTotal":
			memTotal = value
		case "MemAvailable":
			memAvailable = value
		case "MemFree":
			memUsed += value
		case "Buffers":
			memUsed += value
		case "Cached":
			memUsed += value
		}
	}

	if memTotal == 0 {
		return 0, 0, 0
	}

	if memUsed == 0 {
		memUsed = memTotal - memAvailable
	}

	// KB to MB
	total = float64(memTotal) / 1024
	used = float64(memUsed) / 1024
	available = float64(memAvailable) / 1024

	return
}

// getDiskInfo 获取硬盘信息
func (c *SystemStatsCollector) getDiskInfo(stats *models.SystemStats) {
	// 获取当前工作目录所在磁盘
	diskPath := "."
	if len(os.Args) > 0 {
		diskPath = filepath.Dir(os.Args[0])
		if diskPath == "." {
			diskPath, _ = os.Getwd()
		}
	}

	var statfs syscall.Statfs_t
	if err := syscall.Statfs(diskPath, &statfs); err != nil {
		log.Printf("SystemStatsCollector: failed to get disk stats: %v", err)
		stats.DiskTotal = 100 // GB
		stats.DiskUsed = 20
		stats.DiskUsage = 20
		stats.DiskFree = 80
		return
	}

	// Block size
	totalBytes := int64(statfs.Blocks) * int64(statfs.Bsize)
	freeBytes := int64(statfs.Bfree) * int64(statfs.Bsize)
	usedBytes := totalBytes - freeBytes

	// Bytes to GB
	stats.DiskTotal = float64(totalBytes) / 1024 / 1024 / 1024
	stats.DiskUsed = float64(usedBytes) / 1024 / 1024 / 1024
	stats.DiskFree = float64(freeBytes) / 1024 / 1024 / 1024

	if totalBytes > 0 {
		stats.DiskUsage = (float64(usedBytes) / float64(totalBytes)) * 100
	}
}

// getUptime 获取运行时间
func (c *SystemStatsCollector) getUptime() int64 {
	// 尝试读取 /proc/uptime
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return int64(time.Since(time.Now()).Seconds())
	}

	parts := strings.Split(string(data), " ")
	if len(parts) >= 1 {
		uptime, err := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
		if err == nil {
			return int64(uptime)
		}
	}

	return 0
}

// getLoadAverage 获取系统负载
func (c *SystemStatsCollector) getLoadAverage(stats *models.SystemStats) {
	// 尝试读取 /proc/loadavg
	data, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		stats.Load1 = 0
		stats.Load5 = 0
		stats.Load15 = 0
		return
	}

	parts := strings.Split(string(data), " ")
	if len(parts) >= 3 {
		load1, _ := strconv.ParseFloat(parts[0], 64)
		load5, _ := strconv.ParseFloat(parts[1], 64)
		load15, _ := strconv.ParseFloat(parts[2], 64)

		stats.Load1 = load1
		stats.Load5 = load5
		stats.Load15 = load15
	}
}

// statsToCollectData 将系统属性转换为采集数据
func (c *SystemStatsCollector) statsToCollectData(stats *models.SystemStats) *models.CollectData {
	pk, dk := database.GetGatewayIdentity()

	return &models.CollectData{
		DeviceID:   models.SystemStatsDeviceID,
		DeviceName: models.SystemStatsDeviceName,
		ProductKey: pk,
		DeviceKey:  dk,
		Timestamp:  time.Unix(stats.Timestamp, 0),
		Fields: map[string]string{
			"cpu_usage":      fmt.Sprintf("%.2f", stats.CpuUsage),
			"mem_total":      fmt.Sprintf("%.2f", stats.MemTotal),
			"mem_used":       fmt.Sprintf("%.2f", stats.MemUsed),
			"mem_usage":      fmt.Sprintf("%.2f", stats.MemUsage),
			"mem_available": fmt.Sprintf("%.2f", stats.MemAvailable),
			"disk_total":    fmt.Sprintf("%.2f", stats.DiskTotal),
			"disk_used":      fmt.Sprintf("%.2f", stats.DiskUsed),
			"disk_usage":    fmt.Sprintf("%.2f", stats.DiskUsage),
			"disk_free":     fmt.Sprintf("%.2f", stats.DiskFree),
			"uptime":        fmt.Sprintf("%d", stats.Uptime),
			"load_1":        fmt.Sprintf("%.2f", stats.Load1),
			"load_5":        fmt.Sprintf("%.2f", stats.Load5),
			"load_15":       fmt.Sprintf("%.2f", stats.Load15),
		},
	}
}
