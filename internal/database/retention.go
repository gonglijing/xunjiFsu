package database

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
)

// 数据清理相关
var retentionControlMu sync.Mutex
var retentionTicker *time.Ticker
var retentionStop chan struct{}

// StartRetentionCleanup 启动定期历史数据清理
func StartRetentionCleanup(interval time.Duration) {
	retentionControlMu.Lock()
	defer retentionControlMu.Unlock()
	if retentionTicker != nil {
		return
	}
	if interval <= 0 {
		interval = 6 * time.Hour
	}
	retentionStop = make(chan struct{})
	retentionTicker = time.NewTicker(interval)

	go func() {
		log.Printf("Retention cleanup started (interval: %v)", interval)
		for {
			select {
			case <-retentionTicker.C:
				if _, err := CleanupOldDataByConfig(); err != nil {
					log.Printf("Retention cleanup error: %v", err)
				}
			case <-retentionStop:
				log.Println("Retention cleanup stopped")
				return
			}
		}
	}()
}

// StopRetentionCleanup 停止历史数据清理任务
func StopRetentionCleanup() {
	retentionControlMu.Lock()
	defer retentionControlMu.Unlock()
	if retentionTicker != nil {
		retentionTicker.Stop()
		retentionTicker = nil
	}
	if retentionStop != nil {
		close(retentionStop)
		retentionStop = nil
	}
}

// CleanupOldData 清理过期数据（从内存和磁盘）
func CleanupOldData() (int64, error) {
	// 清理内存中的数据，默认 30 天，实际以策略为准
	result, err := DataDB.Exec(
		`DELETE FROM data_points WHERE collected_at < datetime('now', '-30 days')`,
	)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// CleanupOldDataByConfig 根据存储配置清理过期数据
func CleanupOldDataByConfig() (int64, error) {
	configs, err := GetAllStorageConfigs()
	if err != nil {
		return 0, err
	}

	var (
		totalDeleted int64
		globalDays   = retentionDaysDefault
		protectedIDs = make(map[int64]struct{}) // 具备专属策略的设备，避免全局策略覆盖
	)

	for _, config := range configs {
		if config.Enabled != 1 {
			continue
		}

		// 设备专属策略：按 product_key/device_key 过滤
		if config.ProductKey != "" || config.DeviceKey != "" {
			ids, err := findDeviceIDs(config.ProductKey, config.DeviceKey)
			if err != nil {
				return totalDeleted, err
			}
			if len(ids) == 0 {
				continue
			}
			for _, id := range ids {
				protectedIDs[id] = struct{}{}
			}
			days := config.StorageDays
			if days <= 0 {
				days = DefaultRetentionDays
			}
			placeholders := strings.TrimRight(strings.Repeat("?,", len(ids)), ",")
			args := make([]interface{}, 0, len(ids)+1)
			args = append(args, fmt.Sprintf("-%d days", days))
			for _, id := range ids {
				args = append(args, id)
			}
			query := fmt.Sprintf(`DELETE FROM data_points WHERE collected_at < datetime('now', ?) AND device_id IN (%s)`, placeholders)
			result, err := DataDB.Exec(query, args...)
			if err != nil {
				return totalDeleted, err
			}
			deleted, _ := result.RowsAffected()
			totalDeleted += deleted
			continue
		}

		// 全局策略
		if config.StorageDays > 0 {
			globalDays = config.StorageDays
		}
	}

	// 全局策略（或默认值），排除已受专属策略保护的设备
	idsToExclude := make([]int64, 0, len(protectedIDs))
	for id := range protectedIDs {
		idsToExclude = append(idsToExclude, id)
	}

	args := []interface{}{fmt.Sprintf("-%d days", globalDays)}
	query := `DELETE FROM data_points WHERE collected_at < datetime('now', ?)`
	if len(idsToExclude) > 0 {
		placeholders := strings.TrimRight(strings.Repeat("?,", len(idsToExclude)), ",")
		query += fmt.Sprintf(" AND device_id NOT IN (%s)", placeholders)
		for _, id := range idsToExclude {
			args = append(args, id)
		}
	}

	result, err := DataDB.Exec(query, args...)
	if err != nil {
		return totalDeleted, err
	}
	deleted, _ := result.RowsAffected()
	totalDeleted += deleted

	return totalDeleted, nil
}

// findDeviceIDs 按 product_key/device_key 查找设备ID
func findDeviceIDs(productKey, deviceKey string) ([]int64, error) {
	query := "SELECT id FROM devices WHERE 1=1"
	args := []interface{}{}
	if productKey != "" {
		query += " AND product_key = ?"
		args = append(args, productKey)
	}
	if deviceKey != "" {
		query += " AND device_key = ?"
		args = append(args, deviceKey)
	}

	return queryList[int64](ParamDB, query, args,
		func(rows *sql.Rows) (int64, error) {
			var id int64
			if err := rows.Scan(&id); err != nil {
				return 0, err
			}
			return id, nil
		},
	)
}

// CleanupData 清理过期数据（根据时间戳）
func CleanupData(before string) (int64, error) {
	result, err := DataDB.Exec(
		"DELETE FROM data_points WHERE collected_at < ?",
		before,
	)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
