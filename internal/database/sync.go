package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/models"
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
	dataSyncTicker = time.NewTicker(syncInterval)

	go func() {
		log.Printf("Data sync started (interval: %v, batch_trigger: %d)", syncInterval, syncBatchTrigger)
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

	var maxID int64
	if err := DataDB.QueryRow("SELECT IFNULL(MAX(id), 0) FROM data_points").Scan(&maxID); err != nil {
		return fmt.Errorf("failed to get max data point id: %w", err)
	}
	if maxID == 0 {
		return nil
	}

	if _, err := DataDB.Exec(
		`UPDATE data_points SET device_name = ? WHERE device_id = ? AND device_name <> ?`,
		models.SystemStatsDeviceName,
		models.SystemStatsDeviceID,
		models.SystemStatsDeviceName,
	); err != nil {
		log.Printf("Failed to normalize system device_name in memory DB: %v", err)
	}

	// 1. 打开/创建磁盘数据库
	diskDB, err := openSQLite(dataDiskRWDSN(dataDBFile), 1, 1)
	if err != nil {
		return fmt.Errorf("failed to open data database: %w", err)
	}
	defer diskDB.Close()

	if _, err := diskDB.Exec("PRAGMA foreign_keys = OFF"); err != nil {
		return fmt.Errorf("failed to set disk pragma: %w", err)
	}
	if err := ensureDiskDataSchema(diskDB); err != nil {
		return err
	}

	// 2. attach + 单条 SQL 批量搬运，避免逐行扫描带来的分配和系统调用开销
	count, err := syncDataPointsWithAttach(diskDB, maxID)
	if err != nil {
		return err
	}

	// 3. 删除已同步的内存数据
	if _, err := DataDB.Exec("DELETE FROM data_points WHERE id <= ?", maxID); err != nil {
		return fmt.Errorf("failed to cleanup synced data points: %w", err)
	}

	log.Printf("Data synced to disk: %d points", count)
	return nil
}

func dataDiskRWDSN(path string) string {
	if strings.HasPrefix(path, "file:") {
		return path
	}
	return "file:" + filepath.ToSlash(path)
}

func escapeSingleQuotes(path string) string {
	return strings.ReplaceAll(path, "'", "''")
}

func getMainDatabaseFilePath(db *sql.DB) (string, error) {
	rows, err := db.Query(`PRAGMA database_list`)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	for rows.Next() {
		var seq int
		var name, file string
		if err := rows.Scan(&seq, &name, &file); err != nil {
			return "", err
		}
		if name == "main" {
			return file, nil
		}
	}
	if err := rows.Err(); err != nil {
		return "", err
	}
	return "", nil
}

func syncDataPointsWithAttach(diskDB *sql.DB, maxID int64) (int, error) {
	if DataDB == nil {
		return 0, fmt.Errorf("data db is not initialized")
	}

	memPath, err := getMainDatabaseFilePath(DataDB)
	if err != nil {
		return 0, fmt.Errorf("failed to read memory db path: %w", err)
	}

	if strings.TrimSpace(memPath) == "" || memPath == ":memory:" {
		return syncDataPointsRowByRow(diskDB, maxID)
	}

	tx, err := diskDB.Begin()
	if err != nil {
		return 0, fmt.Errorf("failed to begin sync transaction: %w", err)
	}
	defer tx.Rollback()

	attachSQL := fmt.Sprintf("ATTACH DATABASE '%s' AS memdb", escapeSingleQuotes(memPath))
	if _, err := tx.Exec(attachSQL); err != nil {
		_ = tx.Rollback()
		return syncDataPointsRowByRow(diskDB, maxID)
	}
	defer tx.Exec("DETACH DATABASE memdb")

	if _, err := tx.Exec(
		`UPDATE memdb.data_points SET device_name = ? WHERE device_id = ? AND device_name <> ? AND id <= ?`,
		models.SystemStatsDeviceName,
		models.SystemStatsDeviceID,
		models.SystemStatsDeviceName,
		maxID,
	); err != nil {
		return 0, fmt.Errorf("failed to normalize memory data before sync: %w", err)
	}

	result, err := tx.Exec(
		`INSERT OR IGNORE INTO main.data_points (device_id, device_name, field_name, value, value_type, collected_at)
		 SELECT device_id, device_name, field_name, value, value_type, collected_at
		 FROM memdb.data_points
		 WHERE id <= ?`,
		maxID,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to bulk copy data points: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected < 0 {
		rowsAffected = 0
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit bulk sync transaction: %w", err)
	}

	return int(rowsAffected), nil
}

func syncDataPointsRowByRow(diskDB *sql.DB, maxID int64) (int, error) {
	points, err := DataDB.Query(`SELECT device_id, device_name, field_name, value, value_type, collected_at
		FROM data_points WHERE id <= ? ORDER BY id`, maxID)
	if err != nil {
		return 0, fmt.Errorf("failed to query data points: %w", err)
	}
	defer points.Close()

	tx, err := diskDB.Begin()
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`INSERT OR IGNORE INTO data_points
		(device_id, device_name, field_name, value, value_type, collected_at)
		VALUES (?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return 0, fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	count := 0
	for points.Next() {
		var deviceID int64
		var deviceName, fieldName, value, valueType string
		var collectedAt time.Time
		if err := points.Scan(&deviceID, &deviceName, &fieldName, &value, &valueType, &collectedAt); err != nil {
			return 0, err
		}
		deviceName = normalizeDeviceName(deviceID, deviceName)
		if _, err := stmt.Exec(deviceID, deviceName, fieldName, value, valueType, collectedAt); err != nil {
			return 0, err
		}
		count++
	}
	if err := points.Err(); err != nil {
		return 0, err
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return count, nil
}

func ensureDiskDataSchema(db *sql.DB) error {
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS data_points (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		device_id INTEGER NOT NULL,
		device_name TEXT NOT NULL,
		field_name TEXT NOT NULL,
		value TEXT NOT NULL,
		value_type TEXT DEFAULT 'string',
		collected_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(device_id, field_name, collected_at)
	)`); err != nil {
		return fmt.Errorf("failed to ensure data_points table: %w", err)
	}

	if _, err := db.Exec("CREATE INDEX IF NOT EXISTS idx_data_points_device_time ON data_points(device_id, collected_at DESC)"); err != nil {
		return fmt.Errorf("failed to ensure data_points index: %w", err)
	}

	if _, err := db.Exec(
		`UPDATE data_points SET device_name = ? WHERE device_id = ? AND device_name <> ?`,
		models.SystemStatsDeviceName,
		models.SystemStatsDeviceID,
		models.SystemStatsDeviceName,
	); err != nil {
		return fmt.Errorf("failed to normalize system device_name: %w", err)
	}

	return nil
}

// restoreDataFromFile 尝试从备份文件恢复数据
// 注意：由于使用内存数据库，完整恢复比较复杂
// 如果备份文件存在，记录日志但不影响主程序运行
// 实时数据会在系统运行后自动重新采集
func restoreDataFromFile(filename string) error {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return nil
	}

	log.Printf("Backup file exists (%s), real-time data will be collected on startup", filename)
	return nil
}
