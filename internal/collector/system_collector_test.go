package collector

import (
	"database/sql"
	"testing"
	"time"

	_ "github.com/glebarez/go-sqlite"

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
