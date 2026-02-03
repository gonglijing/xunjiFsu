-- migrations/005_fix_resource_type.sql
-- 修复资源类型约束，允许 'net' 和 'tcp' 类型

-- 删除旧的 CHECK 约束（SQLite 不支持直接删除，需要重新创建表）

-- 1. 重命名旧表
ALTER TABLE resources RENAME TO resources_old;

-- 2. 创建新表，包含新的类型约束
CREATE TABLE resources (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT UNIQUE NOT NULL,
    type TEXT NOT NULL CHECK(type IN ('serial', 'net', 'network', 'tcp', 'di', 'do')),
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

-- 3. 复制数据
INSERT INTO resources (id, name, type, port, baud_rate, data_bits, stop_bits, parity, ip_address, port_num, protocol, enabled, created_at, updated_at)
SELECT id, name, 
    CASE WHEN type = 'network' THEN 'net' ELSE type END,
    port, baud_rate, data_bits, stop_bits, parity, ip_address, port_num, protocol, enabled, created_at, updated_at
FROM resources_old;

-- 4. 删除旧表
DROP TABLE resources_old;
