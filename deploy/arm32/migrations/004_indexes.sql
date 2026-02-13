-- 索引迁移脚本：添加高频查询字段的索引
-- 执行时间: 在应用启动时或通过 migrate 命令执行

-- 数据点表索引
CREATE INDEX IF NOT EXISTS idx_data_points_device_time ON data_points(device_id, collected_at DESC);
CREATE INDEX IF NOT EXISTS idx_data_points_device_field ON data_points(device_id, field_name);

-- 报警日志表索引
CREATE INDEX IF NOT EXISTS idx_alarm_logs_device_time ON alarm_logs(device_id, triggered_at DESC);
CREATE INDEX IF NOT EXISTS idx_alarm_logs_unacked ON alarm_logs(acknowledged, triggered_at DESC);

-- 阈值表索引
CREATE INDEX IF NOT EXISTS idx_thresholds_device ON thresholds(device_id);

-- 设备表索引
CREATE INDEX IF NOT EXISTS idx_devices_enabled ON devices(enabled);
CREATE INDEX IF NOT EXISTS idx_devices_resource ON devices(resource_id);
CREATE INDEX IF NOT EXISTS idx_devices_driver ON devices(driver_id);

-- 驱动表索引
CREATE INDEX IF NOT EXISTS idx_drivers_enabled ON drivers(enabled);

-- 北向配置表索引
CREATE INDEX IF NOT EXISTS idx_northbound_enabled ON northbound_configs(enabled);

-- 资源表索引
CREATE INDEX IF NOT EXISTS idx_resources_enabled ON resources(enabled);
