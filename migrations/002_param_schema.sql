-- param.db Schema - 配置数据库
-- 所有配置信息、用户、报警日志等存储在此数据库

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

-- 资源表（串口/DI/DO）
CREATE TABLE IF NOT EXISTS resources (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT UNIQUE NOT NULL,
    type TEXT NOT NULL CHECK(type IN ('serial', 'net', 'di', 'do')),
    -- 串口配置
    port TEXT,
    -- DI/DO 配置
    address INTEGER DEFAULT 1,
    -- 通用状态
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
    config_schema TEXT,
    enabled INTEGER DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- 设备表
CREATE TABLE IF NOT EXISTS devices (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT UNIQUE NOT NULL,
    description TEXT,
    product_key TEXT,
    device_key TEXT,
    driver_type TEXT DEFAULT 'modbus_rtu',
    serial_port TEXT,
    resource_id INTEGER,
    driver_id INTEGER,
    device_config TEXT,
    collect_interval INTEGER DEFAULT 5000,
    upload_interval INTEGER DEFAULT 10000,
    timeout INTEGER DEFAULT 1000,
    -- 串口参数
    baud_rate INTEGER DEFAULT 9600,
    data_bits INTEGER DEFAULT 8,
    stop_bits INTEGER DEFAULT 1,
    parity TEXT DEFAULT 'N' CHECK(parity IN ('N', 'O', 'E')),
    -- 网口参数
    ip_address TEXT,
    port_num INTEGER,
    device_address TEXT,
    protocol TEXT DEFAULT 'tcp' CHECK(protocol IN ('tcp', 'udp')),
    enabled INTEGER DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (resource_id) REFERENCES resources(id) ON DELETE SET NULL,
    FOREIGN KEY (driver_id) REFERENCES drivers(id) ON DELETE SET NULL
);

-- 北向配置表
CREATE TABLE IF NOT EXISTS northbound_configs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT UNIQUE NOT NULL,
    type TEXT NOT NULL,
    enabled INTEGER DEFAULT 1,
    config TEXT NOT NULL,
    upload_interval INTEGER DEFAULT 10000,
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

-- 索引
CREATE INDEX IF NOT EXISTS idx_alarm_logs_device ON alarm_logs(device_id);
CREATE INDEX IF NOT EXISTS idx_alarm_logs_triggered ON alarm_logs(triggered_at DESC);
CREATE INDEX IF NOT EXISTS idx_devices_resource ON devices(resource_id);
CREATE INDEX IF NOT EXISTS idx_devices_driver ON devices(driver_id);
