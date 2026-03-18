package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/database"
)

// HealthStatus 健康检查状态
type HealthStatus struct {
	Status    string           `json:"status"` // healthy, degraded, unhealthy
	Timestamp time.Time        `json:"timestamp"`
	Uptime    string           `json:"uptime"`
	Checks    map[string]Check `json:"checks"`
	System    SystemInfo       `json:"system"`
}

// Check 单个检查项
type Check struct {
	Status  string `json:"status"` // pass, fail
	Message string `json:"message,omitempty"`
}

// SystemInfo 系统信息
type SystemInfo struct {
	GoVersion  string  `json:"go_version"`
	Goroutines int     `json:"goroutines"`
	MemoryMB   float64 `json:"memory_mb"`
}

// startTime 程序启动时间
var startTime = time.Now()

// Health 健康检查接口
func Health(w http.ResponseWriter, r *http.Request) {
	status := buildHealthStatus(time.Now())
	addDatabaseHealthCheck(&status)
	addDataPointHealthCheck(&status)
	status.Status = resolveOverallHealthStatus(status.Checks, status.Status)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(healthHTTPStatus(status.Status))
	json.NewEncoder(w).Encode(status)
}

// Readiness 就绪检查接口
func Readiness(w http.ResponseWriter, r *http.Request) {
	// 检查数据库连接
	if err := checkDatabase(); err != nil {
		http.Error(w, "Not ready: "+err.Error(), http.StatusServiceUnavailable)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// Liveness 存活检查接口
func Liveness(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func buildHealthStatus(now time.Time) HealthStatus {
	return HealthStatus{
		Timestamp: now,
		Uptime:    now.Sub(startTime).String(),
		Checks:    make(map[string]Check),
		System: SystemInfo{
			GoVersion:  runtime.Version(),
			Goroutines: runtime.NumGoroutine(),
			MemoryMB:   readProcessRSSMB(),
		},
	}
}

func addDatabaseHealthCheck(status *HealthStatus) {
	if status == nil {
		return
	}
	if err := checkDatabase(); err != nil {
		status.Checks["database"] = Check{
			Status:  "fail",
			Message: err.Error(),
		}
		status.Status = "degraded"
		return
	}
	status.Checks["database"] = Check{
		Status:  "pass",
		Message: "Connected",
	}
}

func addDataPointHealthCheck(status *HealthStatus) {
	if status == nil {
		return
	}
	dataPointCount, err := getDataPointsCount()
	if err != nil {
		status.Checks["data_points"] = Check{
			Status:  "fail",
			Message: err.Error(),
		}
		return
	}
	status.Checks["data_points"] = Check{
		Status:  "pass",
		Message: dataPointCount,
	}
}

func resolveOverallHealthStatus(checks map[string]Check, currentStatus string) string {
	if currentStatus != "" {
		return currentStatus
	}
	for _, check := range checks {
		if check.Status == "fail" {
			return "degraded"
		}
	}
	return "healthy"
}

func healthHTTPStatus(status string) int {
	if status == "unhealthy" {
		return http.StatusServiceUnavailable
	}
	return http.StatusOK
}

// checkDatabase 检查数据库连接
func checkDatabase() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 检查param数据库
	if err := database.ParamDB.PingContext(ctx); err != nil {
		return fmt.Errorf("param DB: %w", err)
	}

	// 检查data数据库
	if err := database.DataDB.PingContext(ctx); err != nil {
		return fmt.Errorf("data DB: %w", err)
	}

	return nil
}

// getDataPointsCount 获取数据点数量
func getDataPointsCount() (string, error) {
	var count int
	err := database.DataDB.QueryRow("SELECT COUNT(*) FROM data_points").Scan(&count)
	if err != nil {
		return "0", err
	}
	return fmt.Sprintf("%d", count), nil
}
