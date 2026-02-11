package database

import (
	"fmt"
	"testing"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/models"
)

func prepareDataPointsBenchmarkDB(b *testing.B) {
	b.Helper()

	if DataDB != nil {
		_ = DataDB.Close()
	}

	var err error
	DataDB, err = openSQLite(":memory:", 1, 1)
	if err != nil {
		b.Fatalf("open data db: %v", err)
	}
	b.Cleanup(func() {
		_ = DataDB.Close()
	})

	_, err = DataDB.Exec(`CREATE TABLE data_points (
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
		b.Fatalf("create data_points: %v", err)
	}

	_, err = DataDB.Exec(`CREATE TABLE data_cache (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		device_id INTEGER NOT NULL,
		field_name TEXT NOT NULL,
		value TEXT,
		value_type TEXT DEFAULT 'string',
		collected_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(device_id, field_name)
	)`)
	if err != nil {
		b.Fatalf("create data_cache: %v", err)
	}
}

func makeBenchmarkCollectData(fieldCount int) *models.CollectData {
	fields := make(map[string]string, fieldCount)
	for i := 0; i < fieldCount; i++ {
		fields[fmt.Sprintf("f_%d", i)] = fmt.Sprintf("%d", i)
	}

	return &models.CollectData{
		DeviceID:   1,
		DeviceName: "bench-device",
		Timestamp:  time.Now(),
		Fields:     fields,
	}
}

func benchmarkInsertCollectDataWithOptions(b *testing.B, fieldCount int, storeHistory bool) {
	prepareDataPointsBenchmarkDB(b)

	oldSyncBatchTrigger := syncBatchTrigger
	oldMaxDataPointsLimit := maxDataPointsLimit
	oldMaxDataCacheLimit := maxDataCacheLimit
	b.Cleanup(func() {
		syncBatchTrigger = oldSyncBatchTrigger
		maxDataPointsLimit = oldMaxDataPointsLimit
		maxDataCacheLimit = oldMaxDataCacheLimit
	})

	// 基准中避免触发额外清理/落盘逻辑干扰主路径测量。
	syncBatchTrigger = int(^uint(0) >> 1)
	maxDataPointsLimit = int(^uint(0) >> 1)
	maxDataCacheLimit = int(^uint(0) >> 1)

	data := makeBenchmarkCollectData(fieldCount)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := InsertCollectDataWithOptions(data, storeHistory); err != nil {
			b.Fatalf("InsertCollectDataWithOptions error: %v", err)
		}
	}
}

func BenchmarkInsertCollectDataWithOptions_CacheOnly_8Fields(b *testing.B) {
	benchmarkInsertCollectDataWithOptions(b, 8, false)
}

func BenchmarkInsertCollectDataWithOptions_WithHistory_8Fields(b *testing.B) {
	benchmarkInsertCollectDataWithOptions(b, 8, true)
}

func BenchmarkInsertCollectDataWithOptions_CacheOnly_32Fields(b *testing.B) {
	benchmarkInsertCollectDataWithOptions(b, 32, false)
}

func BenchmarkInsertCollectDataWithOptions_WithHistory_32Fields(b *testing.B) {
	benchmarkInsertCollectDataWithOptions(b, 32, true)
}
