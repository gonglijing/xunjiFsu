package service

import (
	"runtime"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/database"
)

type SystemMetrics struct {
	Timestamp time.Time        `json:"timestamp"`
	Uptime    string           `json:"uptime"`
	GoMetrics GoMetrics        `json:"go"`
	Database  DatabaseMetrics  `json:"database"`
	Collector CollectorMetrics `json:"collector"`
}

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

type DatabaseMetrics struct {
	ParamDBConns     int `json:"param_db_open_conns"`
	ParamDBIdleConns int `json:"param_db_idle_conns"`
	DataDBConns      int `json:"data_db_open_conns"`
	DataDBIdleConns  int `json:"data_db_idle_conns"`
	DataPointsCount  int `json:"data_points_count"`
	CacheCount       int `json:"cache_count"`
}

type CollectorMetrics struct {
	Running     bool `json:"running"`
	DeviceCount int  `json:"device_count"`
	TaskCount   int  `json:"task_count"`
}

type RuntimeMetricsSnapshot struct {
	ProcessRSSMB     float64
	SystemMemoryMB   float64
	SystemMemoryUsed float64
}

type MetricsService struct {
	startedAt func() time.Time
	readHost  func() RuntimeMetricsSnapshot
}

func NewMetricsService(startedAt func() time.Time, readHost func() RuntimeMetricsSnapshot) *MetricsService {
	return &MetricsService{
		startedAt: startedAt,
		readHost:  readHost,
	}
}

func (s *MetricsService) Load(now time.Time, collector CollectorMetrics) SystemMetrics {
	memStats := runtime.MemStats{}
	runtime.ReadMemStats(&memStats)

	host := RuntimeMetricsSnapshot{}
	if s != nil && s.readHost != nil {
		host = s.readHost()
	}

	systemUsage := 0.0
	if host.SystemMemoryMB > 0 {
		systemUsage = (host.SystemMemoryUsed / host.SystemMemoryMB) * 100
	}

	paramStats := database.ParamDB.Stats()
	dataStats := database.DataDB.Stats()

	var dataPointsCount, cacheCount int
	_ = database.DataDB.QueryRow("SELECT COUNT(*) FROM data_points").Scan(&dataPointsCount)
	_ = database.DataDB.QueryRow("SELECT COUNT(*) FROM data_cache").Scan(&cacheCount)

	uptime := ""
	if s != nil && s.startedAt != nil {
		uptime = now.Sub(s.startedAt()).String()
	}

	return SystemMetrics{
		Timestamp: now,
		Uptime:    uptime,
		GoMetrics: GoMetrics{
			Version:           runtime.Version(),
			Goroutines:        runtime.NumGoroutine(),
			MemoryAlloc:       float64(memStats.Alloc) / 1024 / 1024,
			MemoryTotal:       float64(memStats.TotalAlloc) / 1024 / 1024,
			HeapAlloc:         float64(memStats.HeapAlloc) / 1024 / 1024,
			ProcessRSS:        host.ProcessRSSMB,
			SystemMemoryTotal: host.SystemMemoryMB,
			SystemMemoryUsed:  host.SystemMemoryUsed,
			SystemMemoryUsage: systemUsage,
			NumGC:             memStats.NumGC,
			GCPause:           ReadLastGCPauseMS(&memStats),
		},
		Database: DatabaseMetrics{
			ParamDBConns:     paramStats.OpenConnections,
			ParamDBIdleConns: paramStats.Idle,
			DataDBConns:      dataStats.OpenConnections,
			DataDBIdleConns:  dataStats.Idle,
			DataPointsCount:  dataPointsCount,
			CacheCount:       cacheCount,
		},
		Collector: collector,
	}
}

func ReadLastGCPauseMS(memStats *runtime.MemStats) float64 {
	if memStats == nil || memStats.NumGC == 0 {
		return 0
	}

	lastPauseIndex := (memStats.NumGC - 1) % uint32(len(memStats.PauseNs))
	return float64(memStats.PauseNs[lastPauseIndex]) / float64(time.Millisecond)
}
