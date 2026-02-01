package handlers

import (
	"encoding/json"
	"net/http"
	"runtime"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/collector"
	"github.com/gonglijing/xunjiFsu/internal/database"
)

// SystemMetrics 系统运行指标
type SystemMetrics struct {
	Timestamp    time.Time       `json:"timestamp"`
	Uptime       string          `json:"uptime"`
	GoMetrics    GoMetrics       `json:"go"`
	Database     DatabaseMetrics `json:"database"`
	Collector    CollectorMetrics `json:"collector"`
}

// GoMetrics Go运行时指标
type GoMetrics struct {
	Version      string  `json:"version"`
	Goroutines   int     `json:"goroutines"`
	MemoryAlloc  float64 `json:"memory_alloc_mb"`
	MemoryTotal  float64 `json:"memory_total_mb"`
	HeapAlloc    float64 `json:"heap_alloc_mb"`
	NumGC        uint32  `json:"num_gc"`
	GCPause      float64 `json:"gc_pause_ms"`
}

// DatabaseMetrics 数据库指标
type DatabaseMetrics struct {
	ParamDBConns    int `json:"param_db_open_conns"`
	ParamDBIdleConns int `json:"param_db_idle_conns"`
	DataDBConns     int `json:"data_db_open_conns"`
	DataDBIdleConns int `json:"data_db_idle_conns"`
	DataPointsCount int `json:"data_points_count"`
	CacheCount      int `json:"cache_count"`
}

// CollectorMetrics 采集器指标
type CollectorMetrics struct {
	Running       bool     `json:"running"`
	DeviceCount   int      `json:"device_count"`
	TaskCount     int      `json:"task_count"`
}

// startTime 程序启动时间
var metricsStartTime = time.Now()

// Metrics 指标接口
func Metrics(w http.ResponseWriter, r *http.Request) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// 获取数据库指标
	paramStats := database.ParamDB.Stats()
	dataStats := database.DataDB.Stats()

	var dataPointsCount, cacheCount int
	database.DataDB.QueryRow("SELECT COUNT(*) FROM data_points").Scan(&dataPointsCount)
	database.DataDB.QueryRow("SELECT COUNT(*) FROM data_cache").Scan(&cacheCount)

	metrics := SystemMetrics{
		Timestamp: time.Now(),
		Uptime:    time.Since(metricsStartTime).String(),
		GoMetrics: GoMetrics{
			Version:      runtime.Version(),
			Goroutines:   runtime.NumGoroutine(),
			MemoryAlloc:  float64(m.Alloc) / 1024 / 1024,
			MemoryTotal:  float64(m.TotalAlloc) / 1024 / 1024,
			HeapAlloc:    float64(m.HeapAlloc) / 1024 / 1024,
			NumGC:        m.NumGC,
			GCPause:      float64(m.GCCPUFraction) * 1000,
		},
		Database: DatabaseMetrics{
			ParamDBConns:    paramStats.OpenConnections,
			ParamDBIdleConns: paramStats.Idle,
			DataDBConns:     dataStats.OpenConnections,
			DataDBIdleConns: dataStats.Idle,
			DataPointsCount: dataPointsCount,
			CacheCount:      cacheCount,
		},
		Collector: CollectorMetrics{
			Running: false, // 需要从外部设置
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

// CollectMetrics 收集器指标收集器
type CollectMetrics struct {
	lastUptime time.Time
}

// NewCollectMetrics 创建指标收集器
func NewCollectMetrics() *CollectMetrics {
	return &CollectMetrics{
		lastUptime: time.Now(),
	}
}

// Record 记录指标
func (m *CollectMetrics) Record(collector *collector.Collector) {
	// 可以在这里记录历史指标
	_ = collector
	_ = m.lastUptime
}

// ResetUptime 重置运行时间计数
func ResetMetricsUptime() {
	metricsStartTime = time.Now()
}
