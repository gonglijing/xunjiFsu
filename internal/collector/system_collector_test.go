package collector

import (
	"database/sql"
	"math"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"github.com/gonglijing/xunjiFsu/internal/database"
	"github.com/gonglijing/xunjiFsu/internal/models"
)

func prepareSystemCollectorTestDB(t *testing.T) {
	t.Helper()

	if database.DataDB != nil {
		_ = database.DataDB.Close()
	}

	var err error
	database.DataDB, err = sql.Open("sqlite", "file:system_collector_test?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open data db: %v", err)
	}
	database.DataDB.SetMaxOpenConns(1)
	database.DataDB.SetMaxIdleConns(1)

	if err := database.DataDB.Ping(); err != nil {
		t.Fatalf("ping data db: %v", err)
	}
	t.Cleanup(func() {
		_ = database.DataDB.Close()
	})

	_, err = database.DataDB.Exec(`CREATE TABLE data_points (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		device_id INTEGER NOT NULL,
		device_name TEXT NOT NULL,
		field_name TEXT NOT NULL,
		value TEXT NOT NULL,
		value_type TEXT DEFAULT 'string',
		collected_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(device_id, field_name)
	)`)
	if err != nil {
		t.Fatalf("create data_points: %v", err)
	}

	_, err = database.DataDB.Exec(`CREATE TABLE data_cache (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		device_id INTEGER NOT NULL,
		field_name TEXT NOT NULL,
		value TEXT,
		value_type TEXT DEFAULT 'string',
		collected_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(device_id, field_name)
	)`)
	if err != nil {
		t.Fatalf("create data_cache: %v", err)
	}
}

func TestSystemStatsCollector_RunCollectsImmediately(t *testing.T) {
	prepareSystemCollectorTestDB(t)

	c := NewSystemStatsCollector(24 * time.Hour)
	c.stopChan = make(chan struct{})
	c.wg.Add(1)

	go c.run()

	deadline := time.Now().Add(1200 * time.Millisecond)
	for time.Now().Before(deadline) {
		var count int
		err := database.DataDB.QueryRow(`SELECT COUNT(*) FROM data_points WHERE device_id = ?`, models.SystemStatsDeviceID).Scan(&count)
		if err != nil {
			t.Fatalf("query data_points: %v", err)
		}
		if count > 0 {
			close(c.stopChan)
			c.wg.Wait()
			return
		}
		time.Sleep(50 * time.Millisecond)
	}

	close(c.stopChan)
	c.wg.Wait()
	t.Fatal("expected system stats to be collected immediately")
}

func TestStatsToCollectData_FormatAndFieldCount(t *testing.T) {
	collector := NewSystemStatsCollector()
	stats := &models.SystemStats{
		Timestamp:    1700000000,
		CpuUsage:     12.3456,
		MemTotal:     2048.4,
		MemUsed:      1024.2,
		MemUsage:     50.01,
		MemAvailable: 1024.2,
		DiskTotal:    100.0,
		DiskUsed:     33.3,
		DiskUsage:    33.3,
		DiskFree:     66.7,
		Uptime:       12345,
		Load1:        0.12,
		Load5:        0.34,
		Load15:       0.56,
	}

	data := collector.statsToCollectData(stats)
	if data == nil {
		t.Fatal("expected non-nil collect data")
	}
	if len(data.Fields) != 13 {
		t.Fatalf("field count = %d, want 13", len(data.Fields))
	}
	if got := data.Fields["cpu_usage"]; got != "12.35" {
		t.Fatalf("cpu_usage = %q, want 12.35", got)
	}
	if got := data.Fields["uptime"]; got != "12345" {
		t.Fatalf("uptime = %q, want 12345", got)
	}
}

func TestFormatSystemMetricValue_TwoDecimalPlaces(t *testing.T) {
	if got := formatSystemMetricValue(1.234); got != "1.23" {
		t.Fatalf("formatSystemMetricValue() = %q, want 1.23", got)
	}
	if got := formatSystemMetricValue(1); got != "1.00" {
		t.Fatalf("formatSystemMetricValue() = %q, want 1.00", got)
	}
}

func TestNewSystemStatsCollector_SetsDiskPath(t *testing.T) {
	collector := NewSystemStatsCollector()
	if collector.diskPath == "" {
		t.Fatal("expected non-empty diskPath")
	}
}

func TestCollectSystemStatsOnce_UsesCachedSnapshot(t *testing.T) {
	collector := NewSystemStatsCollector()
	collector.setLastStats(&models.SystemStats{
		Timestamp: 123,
		CpuUsage:  45.6,
	})

	stats := collector.CollectSystemStatsOnce()
	if stats == nil {
		t.Fatal("expected cached stats")
	}
	if stats.Timestamp != 123 || stats.CpuUsage != 45.6 {
		t.Fatalf("unexpected cached stats: %+v", stats)
	}

	stats.CpuUsage = 0
	cached := collector.getLastStats()
	if cached == nil || cached.CpuUsage != 45.6 {
		t.Fatalf("cached stats should be cloned, got %+v", cached)
	}
}

func TestCollectSystemStatsOnce_PopulatesCacheWhenMissing(t *testing.T) {
	collector := NewSystemStatsCollector()

	stats := collector.CollectSystemStatsOnce()
	if stats == nil {
		t.Fatal("expected collected stats")
	}
	if stats.Timestamp == 0 {
		t.Fatalf("expected non-zero timestamp, got %+v", stats)
	}

	cached := collector.getLastStats()
	if cached == nil {
		t.Fatal("expected cache populated")
	}
	if cached.Timestamp != stats.Timestamp {
		t.Fatalf("cached timestamp = %d, want %d", cached.Timestamp, stats.Timestamp)
	}
}

func TestParseProcStatCPUTotalIdleBytes(t *testing.T) {
	stat, ok := parseProcStatCPUTotalIdleBytes([]byte("cpu  100 20 30 40 5 6 7 8 0 0\ncpu0 1 2 3 4\n"))
	if !ok {
		t.Fatal("expected ok=true")
	}
	if stat.total != 216 {
		t.Fatalf("total = %d, want 216", stat.total)
	}
	if stat.idle != 40 {
		t.Fatalf("idle = %d, want 40", stat.idle)
	}
}

func TestParseProcUptimeSeconds(t *testing.T) {
	if got := parseProcUptimeSeconds([]byte("12345.67 54321.00\n")); got != 12345 {
		t.Fatalf("uptime = %d, want 12345", got)
	}
}

func TestParseProcLoadAverage(t *testing.T) {
	stats := &models.SystemStats{}
	parseProcLoadAverage([]byte("0.12 0.34 0.56 1/100 123\n"), stats)
	if math.Abs(stats.Load1-0.12) > 1e-9 || math.Abs(stats.Load5-0.34) > 1e-9 || math.Abs(stats.Load15-0.56) > 1e-9 {
		t.Fatalf("unexpected load averages: %+v", stats)
	}
}

func BenchmarkParseProcStatCPUTotalIdleBytes(b *testing.B) {
	data := []byte("cpu  100 20 30 40 5 6 7 8 0 0\ncpu0 1 2 3 4\n")
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = parseProcStatCPUTotalIdleBytes(data)
	}
}

func BenchmarkParseProcUptimeSeconds(b *testing.B) {
	data := []byte("12345.67 54321.00\n")
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = parseProcUptimeSeconds(data)
	}
}

func BenchmarkParseProcLoadAverage(b *testing.B) {
	data := []byte("0.12 0.34 0.56 1/100 123\n")
	stats := &models.SystemStats{}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		parseProcLoadAverage(data, stats)
	}
}

func BenchmarkStatsToCollectData(b *testing.B) {
	collector := NewSystemStatsCollector()
	stats := &models.SystemStats{
		Timestamp:    1700000000,
		CpuUsage:     12.3456,
		MemTotal:     2048.4,
		MemUsed:      1024.2,
		MemUsage:     50.01,
		MemAvailable: 1024.2,
		DiskTotal:    100.0,
		DiskUsed:     33.3,
		DiskUsage:    33.3,
		DiskFree:     66.7,
		Uptime:       12345,
		Load1:        0.12,
		Load5:        0.34,
		Load15:       0.56,
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = collector.statsToCollectData(stats)
	}
}

func BenchmarkCollectSystemStatsOnceCached(b *testing.B) {
	collector := NewSystemStatsCollector()
	collector.setLastStats(&models.SystemStats{
		Timestamp:    1700000000,
		CpuUsage:     12.3456,
		MemTotal:     2048.4,
		MemUsed:      1024.2,
		MemUsage:     50.01,
		MemAvailable: 1024.2,
		DiskTotal:    100.0,
		DiskUsed:     33.3,
		DiskUsage:    33.3,
		DiskFree:     66.7,
		Uptime:       12345,
		Load1:        0.12,
		Load5:        0.34,
		Load15:       0.56,
	})

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = collector.CollectSystemStatsOnce()
	}
}
