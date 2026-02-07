-- migrations/006_northbound_connection.sql
-- 北向接口连接配置迁移

PRAGMA foreign_keys = ON;

-- 备份旧表数据
CREATE TABLE IF NOT EXISTS northbound_configs_backup AS
SELECT id, name, type, enabled, config, upload_interval, created_at, updated_at
FROM northbound_configs;

-- 删除旧表
DROP TABLE IF EXISTS northbound_configs;

-- 创建新的北向配置表
CREATE TABLE IF NOT EXISTS northbound_configs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT UNIQUE NOT NULL,
    type TEXT NOT NULL CHECK(type IN ('xunji', 'pandax', 'ithings', 'mqtt', 'http')),
    enabled INTEGER DEFAULT 1,
    upload_interval INTEGER DEFAULT 10000,

    -- 连接配置
    server_url TEXT,
    port INTEGER DEFAULT 0,
    path TEXT,
    username TEXT,
    password TEXT,
    client_id TEXT,

    -- 主题配置
    topic TEXT,
    alarm_topic TEXT,

    -- 协议配置
    qos INTEGER DEFAULT 0,
    retain INTEGER DEFAULT 0,
    keep_alive INTEGER DEFAULT 60,
    timeout INTEGER DEFAULT 30,

    -- XunJi 特定配置
    product_key TEXT,
    device_key TEXT,

    -- 扩展配置
    ext_config TEXT,

    -- Schema 配置（JSON）- 用于前端 schema 方式
    config TEXT,

    -- 状态字段
    connected INTEGER DEFAULT 0,
    last_connected_at DATETIME,

    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- 尝试从旧配置迁移数据
INSERT INTO northbound_configs (
    id, name, type, enabled, upload_interval,
    server_url, port, username, password, client_id,
    topic, alarm_topic, qos, retain, keep_alive, timeout,
    product_key, device_key,
    config,
    created_at, updated_at
)
SELECT
    id, name, type, enabled, COALESCE(upload_interval, 10000),
    -- 解析旧 config JSON
    CASE
        WHEN type = 'http' THEN json_extract(config, '$.url')
        WHEN type = 'mqtt' THEN json_extract(config, '$.broker')
        WHEN type = 'xunji' THEN json_extract(config, '$.serverUrl')
        WHEN type = 'ithings' THEN json_extract(config, '$.serverUrl')
        ELSE NULL
    END,
    CASE
        WHEN type = 'http' THEN COALESCE(json_extract(config, '$.port'), 80)
        WHEN type = 'mqtt' THEN COALESCE(json_extract(config, '$.port'), 1883)
        WHEN type = 'xunji' THEN COALESCE(json_extract(config, '$.port'), 1883)
        WHEN type = 'ithings' THEN COALESCE(json_extract(config, '$.port'), 1883)
        ELSE NULL
    END,
    COALESCE(json_extract(config, '$.username'), ''),
    COALESCE(json_extract(config, '$.password'), ''),
    COALESCE(json_extract(config, '$.client_id'), ''),
    COALESCE(json_extract(config, '$.topic'), ''),
    COALESCE(json_extract(config, '$.alarm_topic'), ''),
    COALESCE(json_extract(config, '$.qos'), 0),
    CASE WHEN json_extract(config, '$.retain') = 1 THEN 1 ELSE 0 END,
    COALESCE(json_extract(config, '$.keepAlive'), 60),
    COALESCE(json_extract(config, '$.connectTimeout'), 30),
    COALESCE(json_extract(config, '$.productKey'), ''),
    COALESCE(json_extract(config, '$.deviceKey'), ''),
    -- 保留旧的 config 字段
    config,
    created_at,
    updated_at
FROM northbound_configs_backup;

-- 删除备份表
DROP TABLE IF EXISTS northbound_configs_backup;

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_northbound_type ON northbound_configs(type);
CREATE INDEX IF NOT EXISTS idx_northbound_enabled ON northbound_configs(enabled);
