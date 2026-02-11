package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
)

// InitParamSchema 初始化配置数据库schema
func InitParamSchema() error {
	migration, err := os.ReadFile("migrations/002_param_schema.sql")
	if err != nil {
		return fmt.Errorf("failed to read param migration: %w", err)
	}

	_, err = ParamDB.Exec(string(migration))
	if err != nil {
		return fmt.Errorf("failed to execute param migration: %w", err)
	}

	if err := cleanupUnusedParamTables(); err != nil {
		return err
	}

	return nil
}

// InitDataSchema 初始化历史数据数据库schema
func InitDataSchema() error {
	migration, err := os.ReadFile("migrations/003_data_schema.sql")
	if err != nil {
		return fmt.Errorf("failed to read data migration: %w", err)
	}

	_, err = DataDB.Exec(string(migration))
	if err != nil {
		return fmt.Errorf("failed to execute data migration: %w", err)
	}

	if err := cleanupUnusedDataTables(); err != nil {
		return err
	}

	// 先确保 alarm_logs 表存在，再创建索引，避免索引脚本里出现 no such table
	if err := ensureAlarmLogsTable(); err != nil {
		return err
	}

	// 执行索引迁移
	if err := initDataIndexes(); err != nil {
		log.Printf("Warning: failed to create data indexes: %v", err)
	}

	return nil
}

// initDataIndexes 创建数据表索引
func initDataIndexes() error {
	indexes, err := os.ReadFile("migrations/004_indexes.sql")
	if err != nil {
		return fmt.Errorf("failed to read index migration: %w", err)
	}

	// 分割并执行每个索引创建语句
	indexSQL := string(indexes)
	// 移除注释
	lines := strings.Split(indexSQL, "\n")
	var statements []string
	var current strings.Builder
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "--") || trimmed == "" {
			continue
		}
		current.WriteString(line)
		current.WriteString("\n")
		if strings.HasSuffix(trimmed, ";") {
			statements = append(statements, current.String())
			current.Reset()
		}
	}

	for _, stmt := range statements {
		if err := executeIndexStatement(stmt); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	log.Println("Data indexes initialized")
	return nil
}

func executeIndexStatement(stmt string) error {
	tableName := extractIndexTableName(stmt)
	if tableName == "" {
		if err := execIndexOnDB(DataDB, stmt); err != nil {
			return err
		}
		return nil
	}

	executed := false

	dataHasTable, err := tableExists(DataDB, tableName)
	if err != nil {
		return err
	}
	if dataHasTable {
		if err := execIndexOnDB(DataDB, stmt); err != nil {
			return err
		}
		executed = true
	}

	paramHasTable, err := tableExists(ParamDB, tableName)
	if err != nil {
		return err
	}
	if paramHasTable && ParamDB != DataDB {
		if err := execIndexOnDB(ParamDB, stmt); err != nil {
			return err
		}
		executed = true
	}

	if !executed {
		// 兜底：当 sqlite_master 查询不到时，按历史行为尝试 DataDB -> ParamDB
		if err := execIndexOnDB(DataDB, stmt); err == nil {
			return nil
		} else if !isNoSuchTableError(err) {
			return err
		}
		if err := execIndexOnDB(ParamDB, stmt); err == nil {
			return nil
		} else if !isNoSuchTableError(err) {
			return err
		}
	}

	return nil
}

func execIndexOnDB(db *sql.DB, stmt string) error {
	if db == nil {
		return fmt.Errorf("database is nil")
	}
	_, err := db.Exec(stmt)
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		return err
	}
	return nil
}

func tableExists(db *sql.DB, tableName string) (bool, error) {
	if db == nil {
		return false, nil
	}
	if strings.TrimSpace(tableName) == "" {
		return false, nil
	}

	var count int
	err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name = ?`, tableName).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to query table %s: %w", tableName, err)
	}
	return count > 0, nil
}

func isNoSuchTableError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "no such table")
}

func extractIndexTableName(stmt string) string {
	upper := strings.ToUpper(stmt)
	onPos := strings.Index(upper, " ON ")
	if onPos < 0 {
		return ""
	}

	rest := strings.TrimSpace(stmt[onPos+4:])
	if rest == "" {
		return ""
	}

	if idx := strings.Index(rest, "("); idx >= 0 {
		rest = rest[:idx]
	}

	rest = strings.TrimSpace(rest)
	if rest == "" {
		return ""
	}

	parts := strings.Fields(rest)
	if len(parts) == 0 {
		return ""
	}

	tableName := strings.Trim(parts[0], "`\"[]")
	if dot := strings.LastIndex(tableName, "."); dot >= 0 && dot < len(tableName)-1 {
		tableName = tableName[dot+1:]
	}

	return tableName
}

// ensureAlarmLogsTable 创建 alarm_logs 表（若缺失）
func ensureAlarmLogsTable() error {
	_, err := ParamDB.Exec(`CREATE TABLE IF NOT EXISTS alarm_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		device_id INTEGER,
		threshold_id INTEGER,
		field_name TEXT,
		actual_value REAL,
		threshold_value REAL,
		operator TEXT,
		severity TEXT,
		message TEXT,
		triggered_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		acknowledged INTEGER DEFAULT 0,
		acknowledged_by TEXT,
		acknowledged_at TIMESTAMP
	)`)
	if err != nil {
		return fmt.Errorf("failed to ensure alarm_logs table: %w", err)
	}
	return nil
}

func cleanupUnusedParamTables() error {
	if _, err := ParamDB.Exec(`DROP TABLE IF EXISTS storage_policies`); err != nil {
		return fmt.Errorf("failed to drop unused table storage_policies: %w", err)
	}
	return nil
}

func cleanupUnusedDataTables() error {
	if _, err := DataDB.Exec(`DROP TABLE IF EXISTS storage_config`); err != nil {
		return fmt.Errorf("failed to drop unused table storage_config: %w", err)
	}
	return nil
}
