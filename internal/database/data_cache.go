package database

import (
	"database/sql"
	"github.com/gonglijing/xunjiFsu/internal/models"
	"log"
)

// ==================== 实时数据缓存操作 (param.db) ====================

// SaveDataCache 保存实时数据缓存（内存）
func SaveDataCache(deviceID int64, deviceName, fieldName, value, valueType string) error {
	_, err := ParamDB.Exec(
		`INSERT OR REPLACE INTO data_cache (device_id, field_name, value, value_type, collected_at)
		VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)`,
		deviceID, fieldName, value, valueType,
	)
	if err != nil {
		return err
	}

	// 检查并清理过量的缓存条目
	enforceDataCacheLimit()
	return nil
}

// enforceDataCacheLimit 强制执行缓存大小限制
func enforceDataCacheLimit() {
	var count int
	if err := ParamDB.QueryRow("SELECT COUNT(*) FROM data_cache").Scan(&count); err != nil {
		log.Printf("Failed to count data cache entries: %v", err)
		return
	}
	if count > maxDataCacheLimit {
		if _, err := ParamDB.Exec("DELETE FROM data_cache WHERE id IN (SELECT id FROM data_cache ORDER BY collected_at ASC LIMIT ?)", count-maxDataCacheLimit); err != nil {
			log.Printf("Failed to cleanup data cache: %v", err)
			return
		}
		log.Printf("Cleaned up data cache, removed %d entries", count-maxDataCacheLimit)
	}
}

// GetDataCacheByDeviceID 根据设备ID获取数据缓存（从内存）
func GetDataCacheByDeviceID(deviceID int64) ([]*models.DataCache, error) {
	return queryList[*models.DataCache](ParamDB,
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
	return queryList[*models.DataCache](ParamDB,
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
