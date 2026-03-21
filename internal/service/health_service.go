package service

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/database"
)

type HealthStatus struct {
	Status    string           `json:"status"`
	Timestamp time.Time        `json:"timestamp"`
	Uptime    string           `json:"uptime"`
	Checks    map[string]Check `json:"checks"`
	System    SystemInfo       `json:"system"`
}

type Check struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

type SystemInfo struct {
	GoVersion  string  `json:"go_version"`
	Goroutines int     `json:"goroutines"`
	MemoryMB   float64 `json:"memory_mb"`
}

type HealthSystemSnapshot struct {
	ProcessRSSMB float64
}

type HealthService struct {
	startedAt   func() time.Time
	readProcess func() HealthSystemSnapshot
}

func NewHealthService(startedAt func() time.Time, readProcess func() HealthSystemSnapshot) *HealthService {
	return &HealthService{
		startedAt:   startedAt,
		readProcess: readProcess,
	}
}

func (s *HealthService) BuildStatus(now time.Time) HealthStatus {
	system := HealthSystemSnapshot{}
	if s != nil && s.readProcess != nil {
		system = s.readProcess()
	}

	uptime := ""
	if s != nil && s.startedAt != nil {
		uptime = now.Sub(s.startedAt()).String()
	}

	return HealthStatus{
		Timestamp: now,
		Uptime:    uptime,
		Checks:    make(map[string]Check),
		System: SystemInfo{
			GoVersion:  runtime.Version(),
			Goroutines: runtime.NumGoroutine(),
			MemoryMB:   system.ProcessRSSMB,
		},
	}
}

func (s *HealthService) Load(now time.Time) HealthStatus {
	status := s.BuildStatus(now)
	AddDatabaseHealthCheck(&status)
	AddDataPointHealthCheck(&status)
	status.Status = ResolveOverallHealthStatus(status.Checks, status.Status)
	return status
}

func (s *HealthService) CheckDatabase() error {
	return CheckDatabase()
}

func AddDatabaseHealthCheck(status *HealthStatus) {
	if status == nil {
		return
	}
	if err := CheckDatabase(); err != nil {
		status.Checks["database"] = Check{Status: "fail", Message: err.Error()}
		status.Status = "degraded"
		return
	}
	status.Checks["database"] = Check{Status: "pass", Message: "Connected"}
}

func AddDataPointHealthCheck(status *HealthStatus) {
	if status == nil {
		return
	}
	dataPointCount, err := DataPointsCount()
	if err != nil {
		status.Checks["data_points"] = Check{Status: "fail", Message: err.Error()}
		return
	}
	status.Checks["data_points"] = Check{Status: "pass", Message: dataPointCount}
}

func ResolveOverallHealthStatus(checks map[string]Check, currentStatus string) string {
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

func HealthHTTPStatus(status string) int {
	if status == "unhealthy" {
		return 503
	}
	return 200
}

func CheckDatabase() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := database.ParamDB.PingContext(ctx); err != nil {
		return fmt.Errorf("param DB: %w", err)
	}
	if err := database.DataDB.PingContext(ctx); err != nil {
		return fmt.Errorf("data DB: %w", err)
	}
	return nil
}

func DataPointsCount() (string, error) {
	var count int
	if err := database.DataDB.QueryRow("SELECT COUNT(*) FROM data_points").Scan(&count); err != nil {
		return "0", err
	}
	return fmt.Sprintf("%d", count), nil
}
