package database

import (
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

	// 执行索引迁移
	if err := initDataIndexes(); err != nil {
		log.Printf("Warning: failed to create data indexes: %v", err)
	}

	// 确保 alarm_logs 表存在
	if err := ensureAlarmLogsTable(); err != nil {
		return err
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
		_, err := DataDB.Exec(stmt)
		if err != nil && !strings.Contains(err.Error(), "already exists") {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	log.Println("Data indexes initialized")
	return nil
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
