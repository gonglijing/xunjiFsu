package httpapi

import (
	"encoding/json"
	"net/http"
	"runtime"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/database"
)

// SystemMetrics 系统运行指标
type SystemMetrics struct {
	Timestamp time.Time        `json:"timestamp"`
	Uptime    string           `json:"uptime"`
	GoMetrics GoMetrics        `json:"go"`
	Database  DatabaseMetrics  `json:"database"`
	Collector CollectorMetrics `json:"collector"`
}

// GoMetrics Go运行时指标
type GoMetrics struct {
	Version           string  `json:"version"`
	Goroutines        int     `json:"goroutines"`
	MemoryAlloc       float64 `json:"memory_alloc_mb"`
	MemoryTotal       float64 `json:"memory_total_mb"`
	HeapAlloc         float64 `json:"heap_alloc_mb"`
	ProcessRSS        float64 `json:"process_rss_mb"`
	SystemMemoryTotal float64 `json:"system_memory_total_mb"`
	SystemMemoryUsed  float64 `json:"system_memory_used_mb"`
	SystemMemoryUsage float64 `json:"system_memory_usage"`
	NumGC             uint32  `json:"num_gc"`
	GCPause           float64 `json:"gc_pause_ms"`
}

// DatabaseMetrics 数据库指标
type DatabaseMetrics struct {
	ParamDBConns     int `json:"param_db_open_conns"`
	ParamDBIdleConns int `json:"param_db_idle_conns"`
	DataDBConns      int `json:"data_db_open_conns"`
	DataDBIdleConns  int `json:"data_db_idle_conns"`
	DataPointsCount  int `json:"data_points_count"`
	CacheCount       int `json:"cache_count"`
}

// CollectorMetrics 采集器指标
type CollectorMetrics struct {
	Running     bool `json:"running"`
	DeviceCount int  `json:"device_count"`
	TaskCount   int  `json:"task_count"`
}

var metricsStartTime = time.Now()

// Metrics 指标接口
func Metrics(w http.ResponseWriter, r *http.Request) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	systemTotal, systemUsed, _ := readSystemMemoryMB()
	systemUsage := 0.0
	if systemTotal > 0 {
		systemUsage = (systemUsed / systemTotal) * 100
	}

	paramStats := database.ParamDB.Stats()
	dataStats := database.DataDB.Stats()

	var dataPointsCount, cacheCount int
	_ = database.DataDB.QueryRow("SELECT COUNT(*) FROM data_points").Scan(&dataPointsCount)
	_ = database.DataDB.QueryRow("SELECT COUNT(*) FROM data_cache").Scan(&cacheCount)

	metrics := SystemMetrics{
		Timestamp: time.Now(),
		Uptime:    time.Since(metricsStartTime).String(),
		GoMetrics: GoMetrics{
			Version:           runtime.Version(),
			Goroutines:        runtime.NumGoroutine(),
			MemoryAlloc:       float64(m.Alloc) / 1024 / 1024,
			MemoryTotal:       float64(m.TotalAlloc) / 1024 / 1024,
			HeapAlloc:         float64(m.HeapAlloc) / 1024 / 1024,
			ProcessRSS:        readProcessRSSMB(),
			SystemMemoryTotal: systemTotal,
			SystemMemoryUsed:  systemUsed,
			SystemMemoryUsage: systemUsage,
			NumGC:             m.NumGC,
			GCPause:           readLastGCPauseMS(&m),
		},
		Database: DatabaseMetrics{
			ParamDBConns:     paramStats.OpenConnections,
			ParamDBIdleConns: paramStats.Idle,
			DataDBConns:      dataStats.OpenConnections,
			DataDBIdleConns:  dataStats.Idle,
			DataPointsCount:  dataPointsCount,
			CacheCount:       cacheCount,
		},
		Collector: CollectorMetrics{
			Running: false,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(metrics)
}

func readLastGCPauseMS(memStats *runtime.MemStats) float64 {
	if memStats == nil || memStats.NumGC == 0 {
		return 0
	}
	lastPauseIndex := (memStats.NumGC - 1) % uint32(len(memStats.PauseNs))
	return float64(memStats.PauseNs[lastPauseIndex]) / float64(time.Millisecond)
}
