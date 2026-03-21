package database

import (
	"fmt"
	"log"
	"os"
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
		interval = 24 * time.Hour
	}
	retentionStop = make(chan struct{})
	retentionTicker = time.NewTicker(interval)

	go func() {
		log.Printf("Retention cleanup started (interval: %v)", interval)
		if _, err := CleanupOldDataByGatewayRetention(); err != nil {
			log.Printf("Initial retention cleanup error: %v", err)
		}
		for {
			select {
			case <-retentionTicker.C:
				if _, err := CleanupOldDataByGatewayRetention(); err != nil {
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

// CleanupOldData 清理过期数据（按网关全局保留天数）
func CleanupOldData() (int64, error) {
	return CleanupOldDataByGatewayRetention()
}

// CleanupOldDataByGatewayRetention 根据网关全局保留天数清理过期数据
func CleanupOldDataByGatewayRetention() (int64, error) {
	days := GetGatewayDataRetentionDays()
	if days <= 0 {
		days = DefaultRetentionDays
	}

	result, err := DataDB.Exec(
		`DELETE FROM data_points WHERE collected_at < datetime('now', ?)`,
		fmt.Sprintf("-%d days", days),
	)
	if err != nil {
		return 0, err
	}
	memDeleted, _ := result.RowsAffected()

	diskDeleted, err := cleanupOldDataOnDisk(days)
	if err != nil {
		return memDeleted, err
	}

	return memDeleted + diskDeleted, nil
}

func cleanupOldDataOnDisk(days int) (int64, error) {
	if dataDBFile == "" {
		return 0, nil
	}
	if _, err := os.Stat(dataDBFile); err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}

	diskDB, err := openSQLite(dataDBFile, 1, 1)
	if err != nil {
		return 0, err
	}
	defer diskDB.Close()

	if err := ensureDiskDataSchema(diskDB); err != nil {
		return 0, err
	}

	result, err := diskDB.Exec(
		`DELETE FROM data_points WHERE collected_at < datetime('now', ?)`,
		fmt.Sprintf("-%d days", days),
	)
	if err != nil {
		return 0, err
	}

	deleted, _ := result.RowsAffected()
	return deleted, nil
}
