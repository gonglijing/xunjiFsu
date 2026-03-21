package database

import (
	"database/sql"
	"strings"
)

func openSQLite(dsn string, maxOpen, maxIdle int) (*sql.DB, error) {
	db, err := sql.Open(sqliteDriverName, dsn)
	if err != nil {
		return nil, err
	}

	if isMemorySQLiteDSN(dsn) {
		if maxOpen <= 0 || maxOpen > 1 {
			maxOpen = 1
		}
		if maxIdle <= 0 {
			maxIdle = 1
		}
	}

	db.SetMaxOpenConns(maxOpen)
	db.SetMaxIdleConns(maxIdle)

	// 内存数据库不设置生命周期，避免连接回收导致内存库丢失。
	if isMemorySQLiteDSN(dsn) {
		db.SetConnMaxLifetime(0)
	} else {
		db.SetConnMaxLifetime(ConnMaxLifetime)
	}

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func isMemorySQLiteDSN(dsn string) bool {
	dsn = strings.TrimSpace(strings.ToLower(dsn))
	if dsn == ":memory:" {
		return true
	}
	return strings.Contains(dsn, "mode=memory")
}
