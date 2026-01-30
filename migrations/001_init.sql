-- migrations/001_init.sql
PRAGMA foreign_keys = ON;

-- 用户表
CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT UNIQUE NOT NULL,
    password TEXT NOT NULL,
    role TEXT DEFAULT 'admin',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- 资源表（串口/网口）
CREATE TABLE IF NOT EXISTS resources (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT UNIQUE NOT NULL,
    type TEXT NOT NULL CHECK(type IN ('serial', 'network')),
    -- 串口配置
    port TEXT,
    baud_rate INTEGER DEFAULT 9600,
    data_bits INTEGER DEFAULT 8,
    stop_bits INTEGER DEFAULT 1,
    parity TEXT DEFAULT 'N' CHECK(parity IN ('N', 'O', 'E')),
    -- 网口配置
    ip_address TEXT,
    port_num INTEGER,
    protocol TEXT DEFAULT 'tcp' CHECK(protocol IN ('tcp', 'udp')),
    enabled INTEGER DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- 驱动表
CREATE TABLE IF NOT EXISTS drivers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT UNIQUE NOT NULL,
    file_path TEXT NOT NULL,
    description TEXT,
    version TEXT,
    config_schema TEXT, -- JSON schema for driver config
    enabled INTEGER DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- 设备表
CREATE TABLE IF NOT EXISTS devices (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT UNIQUE NOT NULL,
    description TEXT,
    resource_id INTEGER,
    driver_id INTEGER,
    device_config TEXT, -- JSON config specific to device
    collect_interval INTEGER DEFAULT 5000, -- 采集周期(ms)
    upload_interval INTEGER DEFAULT 10000, -- 上传周期(ms)
    enabled INTEGER DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (resource_id) REFERENCES resources(id) ON DELETE SET NULL,
    FOREIGN KEY (driver_id) REFERENCES drivers(id) ON DELETE SET NULL
);

-- 设备驱动映射表
CREATE TABLE IF NOT EXISTS device_driver_mapping (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    device_id INTEGER NOT NULL,
    driver_id INTEGER NOT NULL,
    config TEXT, -- JSON config for this specific mapping
    priority INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE CASCADE,
    FOREIGN KEY (driver_id) REFERENCES drivers(id) ON DELETE CASCADE,
    UNIQUE(device_id, driver_id)
);

-- 北向接口配置表
CREATE TABLE IF NOT EXISTS northbound_configs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT UNIQUE NOT NULL,
    type TEXT NOT NULL, -- xunji, mqtt, http, etc.
    enabled INTEGER DEFAULT 1,
    config TEXT NOT NULL, -- JSON config
    upload_interval INTEGER DEFAULT 10000, -- 上传周期(ms)
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- 阈值配置表
CREATE TABLE IF NOT EXISTS thresholds (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    device_id INTEGER NOT NULL,
    field_name TEXT NOT NULL,
    operator TEXT NOT NULL CHECK(operator IN ('>', '<', '>=', '<=', '==', '!=')),
    value REAL NOT NULL,
    severity TEXT DEFAULT 'warning' CHECK(severity IN ('info', 'warning', 'error', 'critical')),
    enabled INTEGER DEFAULT 1,
    message TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE CASCADE
);

-- 报警日志表
CREATE TABLE IF NOT EXISTS alarm_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    device_id INTEGER NOT NULL,
    threshold_id INTEGER,
    field_name TEXT,
    actual_value REAL,
    threshold_value REAL,
    operator TEXT,
    severity TEXT,
    message TEXT,
    triggered_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    acknowledged INTEGER DEFAULT 0,
    acknowledged_by TEXT,
    acknowledged_at DATETIME,
    FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE CASCADE,
    FOREIGN KEY (threshold_id) REFERENCES thresholds(id) ON DELETE SET NULL
);

-- 采集数据缓存表（用于存储最新的采集数据）
CREATE TABLE IF NOT EXISTS data_cache (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    device_id INTEGER NOT NULL,
    field_name TEXT NOT NULL,
    value TEXT,
    value_type TEXT DEFAULT 'string',
    collected_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE CASCADE,
    UNIQUE(device_id, field_name)
);

-- 索引
CREATE INDEX IF NOT EXISTS idx_data_cache_device ON data_cache(device_id);
CREATE INDEX IF NOT EXISTS idx_alarm_logs_device ON alarm_logs(device_id);
CREATE INDEX IF NOT EXISTS idx_alarm_logs_triggered ON alarm_logs(triggered_at DESC);
CREATE INDEX IF NOT EXISTS idx_devices_resource ON devices(resource_id);
CREATE INDEX IF NOT EXISTS idx_devices_driver ON devices(driver_id);
