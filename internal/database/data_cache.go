package database

import (
	"database/sql"
	"log"
	"sync/atomic"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/models"
)

// ==================== 实时数据缓存操作 (data.db - 内存缓存) ====================

var (
	dataCacheCleanupCounter uint64
	dataCacheLastCleanupNS  int64

	// 允许在测试中覆盖
	dataCacheCleanupEveryWrites uint64        = 128
	dataCacheCleanupMinInterval time.Duration = 2 * time.Second
)

// SaveDataCache 保存实时数据缓存（内存）
func SaveDataCache(deviceID int64, deviceName, fieldName, value, valueType string) error {
	_, err := DataDB.Exec(
		`INSERT INTO data_cache (device_id, field_name, value, value_type, collected_at)
		VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(device_id, field_name) DO UPDATE SET
			value = excluded.value,
			value_type = excluded.value_type,
			collected_at = CURRENT_TIMESTAMP`,
		deviceID, fieldName, value, valueType,
	)
	if err != nil {
		return err
	}

	// 节流检查并清理过量缓存，避免每次写入都 count(*)
	maybeEnforceDataCacheLimit()
	return nil
}

func maybeEnforceDataCacheLimit() {
	if maxDataCacheLimit <= 0 {
		return
	}

	writes := atomic.AddUint64(&dataCacheCleanupCounter, 1)
	now := time.Now().UnixNano()
	last := atomic.LoadInt64(&dataCacheLastCleanupNS)
	minIntervalNS := int64(dataCacheCleanupMinInterval)
	if minIntervalNS < 0 {
		minIntervalNS = 0
	}

	if dataCacheCleanupEveryWrites > 0 && writes%dataCacheCleanupEveryWrites != 0 {
		if now-last < minIntervalNS {
			return
		}
	}

	if !atomic.CompareAndSwapInt64(&dataCacheLastCleanupNS, last, now) {
		return
	}

	enforceDataCacheLimit()
}

// enforceDataCacheLimit 强制执行缓存大小限制
func enforceDataCacheLimit() {
	var count int
	if err := DataDB.QueryRow("SELECT COUNT(*) FROM data_cache").Scan(&count); err != nil {
		log.Printf("Failed to count data cache entries: %v", err)
		return
	}
	if count > maxDataCacheLimit {
		if _, err := DataDB.Exec("DELETE FROM data_cache WHERE id IN (SELECT id FROM data_cache ORDER BY collected_at ASC LIMIT ?)", count-maxDataCacheLimit); err != nil {
			log.Printf("Failed to cleanup data cache: %v", err)
			return
		}
		log.Printf("Cleaned up data cache, removed %d entries", count-maxDataCacheLimit)
	}
}

// GetDataCacheByDeviceID 根据设备ID获取数据缓存（从内存）
func GetDataCacheByDeviceID(deviceID int64) ([]*models.DataCache, error) {
	return queryList[*models.DataCache](DataDB,
		"SELECT id, device_id, field_name, value, value_type, collected_at FROM data_cache WHERE device_id = ?",
		[]any{deviceID},
		func(rows *sql.Rows) (*models.DataCache, error) {
			item := &models.DataCache{}
			if err := rows.Scan(&item.ID, &item.DeviceID, &item.FieldName, &item.Value, &item.ValueType, &item.CollectedAt); err != nil {
				return nil, err
			}
			return item, nil
		},
	)
}

// GetAllDataCache 获取所有数据缓存（从内存）
func GetAllDataCache() ([]*models.DataCache, error) {
	return queryList[*models.DataCache](DataDB,
		"SELECT id, device_id, field_name, value, value_type, collected_at FROM data_cache",
		nil,
		func(rows *sql.Rows) (*models.DataCache, error) {
			item := &models.DataCache{}
			if err := rows.Scan(&item.ID, &item.DeviceID, &item.FieldName, &item.Value, &item.ValueType, &item.CollectedAt); err != nil {
				return nil, err
			}
			return item, nil
		},
	)
}
