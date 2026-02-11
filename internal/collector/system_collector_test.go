package collector

import (
	"database/sql"
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
