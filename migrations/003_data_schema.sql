-- data.db Schema - 历史数据存储
-- 采集点数据存储在此数据库

PRAGMA foreign_keys = OFF;

-- 采集数据表（带时间戳的历史数据）
CREATE TABLE IF NOT EXISTS data_points (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    device_id INTEGER NOT NULL,
    device_name TEXT NOT NULL,
    field_name TEXT NOT NULL,
    value TEXT NOT NULL,
    value_type TEXT DEFAULT 'string',
    collected_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(device_id, field_name, collected_at)
);

-- 实时数据缓存表（最新值）
CREATE TABLE IF NOT EXISTS data_cache (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    device_id INTEGER NOT NULL,
    field_name TEXT NOT NULL,
    value TEXT,
    value_type TEXT DEFAULT 'string',
    collected_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(device_id, field_name)
);

-- 存储配置表
CREATE TABLE IF NOT EXISTS storage_config (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT UNIQUE NOT NULL,
    product_key TEXT NOT NULL,
    device_key TEXT NOT NULL,
    storage_days INTEGER DEFAULT 30,
    enabled INTEGER DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- 索引
CREATE INDEX IF NOT EXISTS idx_data_points_device ON data_points(device_id);
CREATE INDEX IF NOT EXISTS idx_data_points_collected ON data_points(collected_at DESC);
CREATE INDEX IF NOT EXISTS idx_data_points_device_time ON data_points(device_id, collected_at);
CREATE INDEX IF NOT EXISTS idx_data_cache_device ON data_cache(device_id);
CREATE INDEX IF NOT EXISTS idx_storage_config ON storage_config(product_key, device_key);
