package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

// 数据同步相关
var dataSyncMu sync.Mutex
var dataSyncControlMu sync.Mutex
var dataSyncTicker *time.Ticker
var dataSyncStop chan struct{}

// StartDataSync 启动数据同步任务（内存 -> 磁盘批量写入）
func StartDataSync() {
	dataSyncControlMu.Lock()
	defer dataSyncControlMu.Unlock()
	if dataSyncTicker != nil {
		return
	}
	dataSyncStop = make(chan struct{})
	dataSyncTicker = time.NewTicker(SyncInterval)

	go func() {
		log.Printf("Data sync started (interval: %v, batch_trigger: %d)", SyncInterval, SyncBatchTrigger)
		for {
			select {
			case <-dataSyncTicker.C:
				if err := syncDataToDisk(); err != nil {
					log.Printf("Failed to sync data to disk: %v", err)
				}
			case <-dataSyncStop:
				log.Println("Data sync stopped")
				return
			}
		}
	}()
}

// TriggerSyncIfNeeded 检查是否需要触发同步
// 返回true表示已触发同步
func TriggerSyncIfNeeded() bool {
	var count int
	if err := DataDB.QueryRow("SELECT COUNT(*) FROM data_points").Scan(&count); err != nil {
		log.Printf("Failed to count data points for sync trigger: %v", err)
		return false
	}
	if count >= syncBatchTrigger {
		log.Printf("Triggering sync due to data count: %d", count)
		go func() {
			if err := syncDataToDiskFn(); err != nil {
				log.Printf("Failed to sync data to disk: %v", err)
			}
		}()
		return true
	}
	return false
}

// SyncDataToDisk 手动触发数据同步（公开函数，供优雅关闭调用）
func SyncDataToDisk() error {
	return syncDataToDisk()
}

// StopDataSync 停止数据同步任务
func StopDataSync() {
	dataSyncControlMu.Lock()
	defer dataSyncControlMu.Unlock()
	if dataSyncTicker != nil {
		dataSyncTicker.Stop()
		dataSyncTicker = nil
	}
	if dataSyncStop != nil {
		close(dataSyncStop)
		dataSyncStop = nil
	}
}

// syncDataToDisk 将内存数据批量同步到磁盘
func syncDataToDisk() error {
	dataSyncMu.Lock()
	defer dataSyncMu.Unlock()

	log.Println("Syncing data to disk...")

	// 1. 创建临时数据库文件
	tempFile := dataDBFile + ".tmp"
	diskDB, err := sql.Open("sqlite", tempFile)
	if err != nil {
		return fmt.Errorf("failed to open temp database: %w", err)
	}
	defer diskDB.Close()

	// 2. 复制 schema
	schemaRows, err := DataDB.Query("SELECT sql FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'")
	if err != nil {
		return fmt.Errorf("failed to get schema: %w", err)
	}
	defer schemaRows.Close()

	for schemaRows.Next() {
		var sql string
		if err := schemaRows.Scan(&sql); err != nil {
			return err
		}
		if _, err := diskDB.Exec(sql); err != nil {
			return fmt.Errorf("failed to create table: %w", err)
		}
	}
	if err := schemaRows.Err(); err != nil {
		return err
	}

	// 3. 批量复制数据点
	points, err := DataDB.Query("SELECT device_id, device_name, field_name, value, value_type, collected_at FROM data_points ORDER BY collected_at")
	if err != nil {
		return fmt.Errorf("failed to query data points: %w", err)
	}
	defer points.Close()

	// 开启事务批量插入
	tx, err := diskDB.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	stmt, err := tx.Prepare(`INSERT OR REPLACE INTO data_points 
		(id, device_id, device_name, field_name, value, value_type, collected_at) 
		VALUES (NULL, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	count := 0
	for points.Next() {
		var deviceID int64
		var deviceName, fieldName, value, valueType string
		var collectedAt time.Time
		if err := points.Scan(&deviceID, &deviceName, &fieldName, &value, &valueType, &collectedAt); err != nil {
			tx.Rollback()
			return err
		}
		if _, err := stmt.Exec(deviceID, deviceName, fieldName, value, valueType, collectedAt); err != nil {
			tx.Rollback()
			return err
		}
		count++
	}
	if err := points.Err(); err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// 4. 原子替换文件
	if err := os.Rename(tempFile, dataDBFile); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	log.Printf("Data synced to disk: %d points", count)
	return nil
}

// restoreDataFromFile 尝试从备份文件恢复数据
// 注意：由于使用内存数据库，完整恢复比较复杂
// 如果备份文件存在，记录日志但不影响主程序运行
// 实时数据会在系统运行后自动重新采集
func restoreDataFromFile(filename string) error {
	// 检查文件是否存在
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return nil
	}

	// 记录有备份文件存在，但不尝试恢复
	// 数据会在运行时自动重新采集
	log.Printf("Backup file exists (%s), real-time data will be collected on startup", filename)
	return nil
}
