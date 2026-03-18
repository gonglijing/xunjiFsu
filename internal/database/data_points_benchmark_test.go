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

func makeBenchmarkDataPointEntries(fieldCount int) []DataPointEntry {
	entries := make([]DataPointEntry, 0, fieldCount)
	now := time.Now()
	for i := 0; i < fieldCount; i++ {
		entries = append(entries, DataPointEntry{
			DeviceID:    1,
			DeviceName:  "bench-device",
			FieldName:   fmt.Sprintf("f_%d", i),
			Value:       fmt.Sprintf("%d", i),
			ValueType:   collectDataValueTypeString,
			CollectedAt: now,
		})
	}
	return entries
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

func BenchmarkInsertCollectDataWithOptions_CacheOnly_1Field(b *testing.B) {
	benchmarkInsertCollectDataWithOptions(b, 1, false)
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

func BenchmarkInsertCollectDataWithOptions_CacheOnly_1000Fields(b *testing.B) {
	benchmarkInsertCollectDataWithOptions(b, 1000, false)
}

func BenchmarkInsertCollectDataWithOptions_WithHistory_1000Fields(b *testing.B) {
	benchmarkInsertCollectDataWithOptions(b, 1000, true)
}

func BenchmarkInsertCollectDataWithOptions_CacheOnly_10000Fields(b *testing.B) {
	benchmarkInsertCollectDataWithOptions(b, 10000, false)
}

func BenchmarkInsertCollectDataWithOptions_WithHistory_10000Fields(b *testing.B) {
	benchmarkInsertCollectDataWithOptions(b, 10000, true)
}

func BenchmarkWriteCollectDataBatch_SingleItem_CacheOnly_8Fields(b *testing.B) {
	prepareDataPointsBenchmarkDB(b)

	oldSyncBatchTrigger := syncBatchTrigger
	oldMaxDataPointsLimit := maxDataPointsLimit
	oldMaxDataCacheLimit := maxDataCacheLimit
	b.Cleanup(func() {
		syncBatchTrigger = oldSyncBatchTrigger
		maxDataPointsLimit = oldMaxDataPointsLimit
		maxDataCacheLimit = oldMaxDataCacheLimit
	})

	syncBatchTrigger = int(^uint(0) >> 1)
	maxDataPointsLimit = int(^uint(0) >> 1)
	maxDataCacheLimit = int(^uint(0) >> 1)

	items := []collectWriteRequest{{data: makeBenchmarkCollectData(8), storeHistory: false}}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := writeCollectDataBatch(items); err != nil {
			b.Fatalf("writeCollectDataBatch error: %v", err)
		}
	}
}

func BenchmarkWriteCollectDataBatch_TwoItems_CacheOnly_8Fields(b *testing.B) {
	prepareDataPointsBenchmarkDB(b)

	oldSyncBatchTrigger := syncBatchTrigger
	oldMaxDataPointsLimit := maxDataPointsLimit
	oldMaxDataCacheLimit := maxDataCacheLimit
	b.Cleanup(func() {
		syncBatchTrigger = oldSyncBatchTrigger
		maxDataPointsLimit = oldMaxDataPointsLimit
		maxDataCacheLimit = oldMaxDataCacheLimit
	})

	syncBatchTrigger = int(^uint(0) >> 1)
	maxDataPointsLimit = int(^uint(0) >> 1)
	maxDataCacheLimit = int(^uint(0) >> 1)

	items := []collectWriteRequest{
		{data: makeBenchmarkCollectData(8), storeHistory: false},
		{data: makeBenchmarkCollectData(8), storeHistory: false},
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := writeCollectDataBatch(items); err != nil {
			b.Fatalf("writeCollectDataBatch error: %v", err)
		}
	}
}

func BenchmarkBatchSaveDataPoints_32Fields(b *testing.B) {
	prepareDataPointsBenchmarkDB(b)

	oldSyncBatchTrigger := syncBatchTrigger
	oldMaxDataPointsLimit := maxDataPointsLimit
	b.Cleanup(func() {
		syncBatchTrigger = oldSyncBatchTrigger
		maxDataPointsLimit = oldMaxDataPointsLimit
	})

	syncBatchTrigger = int(^uint(0) >> 1)
	maxDataPointsLimit = int(^uint(0) >> 1)

	entries := makeBenchmarkDataPointEntries(32)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		deviceID := int64(i + 1)
		for j := range entries {
			entries[j].DeviceID = deviceID
		}
		if err := BatchSaveDataPoints(entries); err != nil {
			b.Fatalf("BatchSaveDataPoints error: %v", err)
		}
	}
}

func BenchmarkBatchSaveLatestDataPoints_32Fields(b *testing.B) {
	prepareDataPointsBenchmarkDB(b)

	entries := makeBenchmarkDataPointEntries(32)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		deviceID := int64(i + 1)
		for j := range entries {
			entries[j].DeviceID = deviceID
		}
		if err := BatchSaveLatestDataPoints(entries); err != nil {
			b.Fatalf("BatchSaveLatestDataPoints error: %v", err)
		}
	}
}

func BenchmarkBatchSaveDataCacheEntries_32Fields(b *testing.B) {
	prepareDataPointsBenchmarkDB(b)

	entries := makeBenchmarkDataPointEntries(32)
	for i := range entries {
		entries[i].CollectedAt = time.Time{}
		entries[i].DeviceName = ""
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		deviceID := int64(i + 1)
		for j := range entries {
			entries[j].DeviceID = deviceID
		}
		if err := BatchSaveDataCacheEntries(entries); err != nil {
			b.Fatalf("BatchSaveDataCacheEntries error: %v", err)
		}
	}
}
