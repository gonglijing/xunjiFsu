package collector

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
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
	diskPath      string
	stopChan      chan struct{}
	lastStats     *models.SystemStats
	wg            sync.WaitGroup
	running       bool
	mu            sync.RWMutex
}

var (
	sysInstance *SystemStatsCollector
	sysOnce     sync.Once
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
		diskPath: resolveSystemStatsDiskPath(),
		stopChan: make(chan struct{}),
	}
}

func resolveSystemStatsDiskPath() string {
	diskPath := "."
	if len(os.Args) > 0 {
		diskPath = filepath.Dir(os.Args[0])
		if diskPath == "." {
			if wd, err := os.Getwd(); err == nil && wd != "" {
				diskPath = wd
			}
		}
	}
	return diskPath
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

	// 启动后立即采集一次，避免重启后实时页面长时间无系统测点
	c.collect()

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
	c.setLastStats(stats)
	data := c.statsToCollectData(stats)

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

	// 保存到数据库
	if err := database.EnqueueCollectDataWrite(data, true); err != nil {
		log.Printf("SystemStatsCollector: failed to insert data: %v", err)
	}

	log.Printf("SystemStatsCollector: collected cpu=%.1f%% mem=%.1f%% disk=%.1f%%",
		stats.CpuUsage, stats.MemUsage, stats.DiskUsage)
}

// CollectSystemStatsOnce 立即采集一次系统属性（公开方法，供北向适配器调用）
func (c *SystemStatsCollector) CollectSystemStatsOnce() *models.SystemStats {
	if stats := c.getLastStats(); stats != nil {
		return stats
	}
	stats := c.collectSystemStats()
	c.setLastStats(stats)
	return cloneSystemStats(stats)
}

func (c *SystemStatsCollector) getLastStats() *models.SystemStats {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return cloneSystemStats(c.lastStats)
}

func (c *SystemStatsCollector) setLastStats(stats *models.SystemStats) {
	c.mu.Lock()
	c.lastStats = cloneSystemStats(stats)
	c.mu.Unlock()
}

func cloneSystemStats(stats *models.SystemStats) *models.SystemStats {
	if stats == nil {
		return nil
	}
	clone := *stats
	return &clone
}

// collectSystemStats 采集一次系统属性
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
	return c.readProcCPUUsage()
}

func (c *SystemStatsCollector) readProcCPUUsage() float64 {
	first, ok := readProcStatCPUTotalIdle()
	if !ok {
		return 0
	}

	time.Sleep(100 * time.Millisecond)

	second, ok := readProcStatCPUTotalIdle()
	if !ok {
		return 0
	}

	totalDelta := second.total - first.total
	idleDelta := second.idle - first.idle
	if totalDelta <= 0 {
		return 0
	}

	usage := (float64(totalDelta-idleDelta) / float64(totalDelta)) * 100
	if usage < 0 {
		return 0
	}
	if usage > 100 {
		return 100
	}
	return usage
}

type procCPUStat struct {
	total uint64
	idle  uint64
}

func readProcStatCPUTotalIdle() (procCPUStat, bool) {
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return procCPUStat{}, false
	}
	return parseProcStatCPUTotalIdleBytes(data)
}

func parseProcStatCPUTotalIdleBytes(data []byte) (procCPUStat, bool) {
	line := firstLineBytes(data)
	if len(line) < 5 || line[0] != 'c' || line[1] != 'p' || line[2] != 'u' || line[3] != ' ' {
		return procCPUStat{}, false
	}

	field := 0
	i := 4
	var total uint64
	var idle uint64
	for i < len(line) {
		for i < len(line) && line[i] == ' ' {
			i++
		}
		if i >= len(line) {
			break
		}

		value, next, ok := parseUintField(line, i)
		if !ok {
			return procCPUStat{}, false
		}
		field++
		total += value
		if field == 4 {
			idle = value
		}
		i = next
	}
	if field < 4 {
		return procCPUStat{}, false
	}

	return procCPUStat{total: total, idle: idle}, true
}

func firstLineBytes(data []byte) []byte {
	line := data
	if idx := indexByte(data, '\n'); idx >= 0 {
		line = data[:idx]
	}
	start := 0
	for start < len(line) && (line[start] == ' ' || line[start] == '\t' || line[start] == '\r') {
		start++
	}
	end := len(line)
	for end > start && (line[end-1] == ' ' || line[end-1] == '\t' || line[end-1] == '\r') {
		end--
	}
	return line[start:end]
}

func parseUintField(data []byte, start int) (uint64, int, bool) {
	if start >= len(data) {
		return 0, start, false
	}
	var value uint64
	i := start
	for i < len(data) {
		ch := data[i]
		if ch < '0' || ch > '9' {
			break
		}
		value = value*10 + uint64(ch-'0')
		i++
	}
	if i == start {
		return 0, start, false
	}
	return value, i, true
}

func parseFloatField(data []byte, start int) (float64, int, bool) {
	if start >= len(data) {
		return 0, start, false
	}
	i := start
	var whole uint64
	digits := 0
	for i < len(data) {
		ch := data[i]
		if ch < '0' || ch > '9' {
			break
		}
		whole = whole*10 + uint64(ch-'0')
		i++
		digits++
	}
	if digits == 0 {
		return 0, start, false
	}
	value := float64(whole)
	if i < len(data) && data[i] == '.' {
		i++
		scale := 0.1
		for i < len(data) {
			ch := data[i]
			if ch < '0' || ch > '9' {
				break
			}
			value += float64(ch-'0') * scale
			scale *= 0.1
			i++
		}
	}
	return value, i, true
}

// getMemoryInfo 获取内存信息
func (c *SystemStatsCollector) getMemoryInfo(stats *models.SystemStats) {
	total, used, available := ReadSystemMemoryMB()

	stats.MemTotal = total
	stats.MemUsed = used
	stats.MemAvailable = available

	if total > 0 {
		stats.MemUsage = (used / total) * 100
	} else {
		stats.MemTotal = 0
		stats.MemUsed = 0
		stats.MemUsage = 0
		stats.MemAvailable = 0
	}
}

// getDiskInfo 获取硬盘信息
func (c *SystemStatsCollector) getDiskInfo(stats *models.SystemStats) {
	var statfs syscall.Statfs_t
	if err := syscall.Statfs(c.diskPath, &statfs); err != nil {
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
		return 0
	}
	return parseProcUptimeSeconds(data)
}

func parseProcUptimeSeconds(data []byte) int64 {
	for i := 0; i < len(data); i++ {
		if data[i] == ' ' || data[i] == '\n' || data[i] == '\t' || data[i] == '\r' {
			value, _, ok := parseFloatField(data, 0)
			if ok {
				return int64(value)
			}
			return 0
		}
	}
	value, _, ok := parseFloatField(data, 0)
	if ok {
		return int64(value)
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
	parseProcLoadAverage(data, stats)
}

func parseProcLoadAverage(data []byte, stats *models.SystemStats) {
	if stats == nil {
		return
	}
	load1, next, ok := parseFloatField(data, skipProcSpaces(data, 0))
	if !ok {
		stats.Load1 = 0
		stats.Load5 = 0
		stats.Load15 = 0
		return
	}
	load5, next, ok := parseFloatField(data, skipProcSpaces(data, next))
	if !ok {
		stats.Load1 = 0
		stats.Load5 = 0
		stats.Load15 = 0
		return
	}
	load15, _, ok := parseFloatField(data, skipProcSpaces(data, next))
	if !ok {
		stats.Load1 = 0
		stats.Load5 = 0
		stats.Load15 = 0
		return
	}
	stats.Load1 = load1
	stats.Load5 = load5
	stats.Load15 = load15
}

func skipProcSpaces(data []byte, start int) int {
	for start < len(data) {
		switch data[start] {
		case ' ', '\n', '\t', '\r':
			start++
		default:
			return start
		}
	}
	return start
}

// statsToCollectData 将系统属性转换为采集数据
func formatSystemMetricValue(value float64) string {
	return strconv.FormatFloat(value, 'f', 2, 64)
}

func (c *SystemStatsCollector) statsToCollectData(stats *models.SystemStats) *models.CollectData {
	fields := make(map[string]string, 13)
	fields["cpu_usage"] = formatSystemMetricValue(stats.CpuUsage)
	fields["mem_total"] = formatSystemMetricValue(stats.MemTotal)
	fields["mem_used"] = formatSystemMetricValue(stats.MemUsed)
	fields["mem_usage"] = formatSystemMetricValue(stats.MemUsage)
	fields["mem_available"] = formatSystemMetricValue(stats.MemAvailable)
	fields["disk_total"] = formatSystemMetricValue(stats.DiskTotal)
	fields["disk_used"] = formatSystemMetricValue(stats.DiskUsed)
	fields["disk_usage"] = formatSystemMetricValue(stats.DiskUsage)
	fields["disk_free"] = formatSystemMetricValue(stats.DiskFree)
	fields["uptime"] = strconv.FormatInt(stats.Uptime, 10)
	fields["load_1"] = formatSystemMetricValue(stats.Load1)
	fields["load_5"] = formatSystemMetricValue(stats.Load5)
	fields["load_15"] = formatSystemMetricValue(stats.Load15)

	return &models.CollectData{
		DeviceID:   models.SystemStatsDeviceID,
		DeviceName: models.SystemStatsDeviceName,
		Timestamp:  time.Unix(stats.Timestamp, 0),
		Fields:     fields,
	}
}
