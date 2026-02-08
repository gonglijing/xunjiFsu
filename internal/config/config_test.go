// =============================================================================
// 配置模块单元测试
// =============================================================================
package config

import (
	"os"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	// 验证服务器默认配置
	if cfg.ListenAddr != ":8080" {
		t.Errorf("ListenAddr = %s, want :8080", cfg.ListenAddr)
	}

	// 验证TLS默认配置
	if cfg.TLSCertFile != "" {
		t.Errorf("TLSCertFile = %s, want empty", cfg.TLSCertFile)
	}
	if cfg.TLSKeyFile != "" {
		t.Errorf("TLSKeyFile = %s, want empty", cfg.TLSKeyFile)
	}
	if cfg.TLSAuto != false {
		t.Errorf("TLSAuto = %v, want false", cfg.TLSAuto)
	}

	// 验证HTTP超时默认配置
	if cfg.HTTPReadTimeout != 30*time.Second {
		t.Errorf("HTTPReadTimeout = %v, want 30s", cfg.HTTPReadTimeout)
	}
	if cfg.HTTPWriteTimeout != 30*time.Second {
		t.Errorf("HTTPWriteTimeout = %v, want 30s", cfg.HTTPWriteTimeout)
	}
	if cfg.HTTPIdleTimeout != 60*time.Second {
		t.Errorf("HTTPIdleTimeout = %v, want 60s", cfg.HTTPIdleTimeout)
	}

	// 验证数据库默认配置
	if cfg.DBPath != "gogw.db" {
		t.Errorf("DBPath = %s, want gogw.db", cfg.DBPath)
	}
	if cfg.ParamDBPath != "param.db" {
		t.Errorf("ParamDBPath = %s, want param.db", cfg.ParamDBPath)
	}
	if cfg.DataDBPath != "data.db" {
		t.Errorf("DataDBPath = %s, want data.db", cfg.DataDBPath)
	}

	// 验证日志默认配置
	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel = %s, want info", cfg.LogLevel)
	}
	if cfg.LogJSON != false {
		t.Errorf("LogJSON = %v, want false", cfg.LogJSON)
	}

	// 验证采集器默认配置
	if cfg.CollectorEnabled != true {
		t.Errorf("CollectorEnabled = %v, want true", cfg.CollectorEnabled)
	}
	if cfg.CollectorWorkers != 10 {
		t.Errorf("CollectorWorkers = %d, want 10", cfg.CollectorWorkers)
	}
	if cfg.SyncInterval != 5*time.Minute {
		t.Errorf("SyncInterval = %v, want 5m", cfg.SyncInterval)
	}
	if cfg.CollectorDeviceSyncInterval != 10*time.Second {
		t.Errorf("CollectorDeviceSyncInterval = %v, want 10s", cfg.CollectorDeviceSyncInterval)
	}
	if cfg.CollectorCommandPollInterval != 500*time.Millisecond {
		t.Errorf("CollectorCommandPollInterval = %v, want 500ms", cfg.CollectorCommandPollInterval)
	}

	// 验证驱动目录默认配置
	if cfg.DriversDir != "drivers" {
		t.Errorf("DriversDir = %s, want drivers", cfg.DriversDir)
	}
	if cfg.NorthboundPluginsDir != "plugin_north" {
		t.Errorf("NorthboundPluginsDir = %s, want plugin_north", cfg.NorthboundPluginsDir)
	}
	if cfg.NorthboundMQTTReconnectInterval != 5*time.Second {
		t.Errorf("NorthboundMQTTReconnectInterval = %v, want 5s", cfg.NorthboundMQTTReconnectInterval)
	}

	// 验证阈值缓存默认配置
	if cfg.ThresholdCacheEnabled != true {
		t.Errorf("ThresholdCacheEnabled = %v, want true", cfg.ThresholdCacheEnabled)
	}
	if cfg.ThresholdCacheTTL != time.Minute {
		t.Errorf("ThresholdCacheTTL = %v, want 1m", cfg.ThresholdCacheTTL)
	}

	// 验证内存数据库限制默认配置
	if cfg.MaxDataPoints != 100000 {
		t.Errorf("MaxDataPoints = %d, want 100000", cfg.MaxDataPoints)
	}
	if cfg.MaxDataCache != 10000 {
		t.Errorf("MaxDataCache = %d, want 10000", cfg.MaxDataCache)
	}
}

func TestConfig_String(t *testing.T) {
	cfg := DefaultConfig()
	str := cfg.String()

	// 验证字符串包含关键配置
	if str == "" {
		t.Error("String() returned empty string")
	}
}

func TestGetAllowedOrigins_Default(t *testing.T) {
	cfg := &Config{
		AllowedOrigins: "",
	}

	origins := cfg.GetAllowedOrigins()

	if len(origins) != 2 {
		t.Errorf("GetAllowedOrigins() returned %d origins, want 2", len(origins))
	}

	// 验证默认来源
	found := false
	for _, o := range origins {
		if o == "http://localhost:8080" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Default origin http://localhost:8080 not found")
	}
}

func TestGetAllowedOrigins_Custom(t *testing.T) {
	cfg := &Config{
		AllowedOrigins: "http://example.com,https://test.com",
	}

	origins := cfg.GetAllowedOrigins()

	if len(origins) != 2 {
		t.Errorf("GetAllowedOrigins() returned %d origins, want 2", len(origins))
	}

	if origins[0] != "http://example.com" {
		t.Errorf("Origin[0] = %s, want http://example.com", origins[0])
	}
	if origins[1] != "https://test.com" {
		t.Errorf("Origin[1] = %s, want https://test.com", origins[1])
	}
}

func TestGetAllowedOrigins_Single(t *testing.T) {
	cfg := &Config{
		AllowedOrigins: "http://single.com",
	}

	origins := cfg.GetAllowedOrigins()

	if len(origins) != 1 {
		t.Errorf("GetAllowedOrigins() returned %d origins, want 1", len(origins))
	}
}

func TestLoadFromEnv_ListenAddr(t *testing.T) {
	// 设置环境变量
	os.Setenv("LISTEN_ADDR", ":9090")
	defer os.Unsetenv("LISTEN_ADDR")

	cfg := &Config{}
	loadFromEnv(cfg)

	if cfg.ListenAddr != ":9090" {
		t.Errorf("ListenAddr = %s, want :9090", cfg.ListenAddr)
	}
}

func TestLoadFromEnv_HTTPTimeout(t *testing.T) {
	// 设置环境变量
	os.Setenv("HTTP_READ_TIMEOUT", "60s")
	os.Setenv("HTTP_WRITE_TIMEOUT", "120s")
	defer os.Unsetenv("HTTP_READ_TIMEOUT")
	defer os.Unsetenv("HTTP_WRITE_TIMEOUT")

	cfg := &Config{}
	loadFromEnv(cfg)

	if cfg.HTTPReadTimeout != 60*time.Second {
		t.Errorf("HTTPReadTimeout = %v, want 60s", cfg.HTTPReadTimeout)
	}
	if cfg.HTTPWriteTimeout != 120*time.Second {
		t.Errorf("HTTPWriteTimeout = %v, want 120s", cfg.HTTPWriteTimeout)
	}
}

func TestLoadFromEnv_DBPath(t *testing.T) {
	os.Setenv("DB_PATH", "/custom/path/db.sqlite")
	defer os.Unsetenv("DB_PATH")

	cfg := &Config{}
	loadFromEnv(cfg)

	if cfg.DBPath != "/custom/path/db.sqlite" {
		t.Errorf("DBPath = %s, want /custom/path/db.sqlite", cfg.DBPath)
	}
}

func TestLoadFromEnv_TLS(t *testing.T) {
	os.Setenv("TLS_AUTO", "true")
	os.Setenv("TLS_DOMAIN", "example.com")
	os.Setenv("TLS_CACHE_DIR", "/tmp/certs")
	defer os.Unsetenv("TLS_AUTO")
	defer os.Unsetenv("TLS_DOMAIN")
	defer os.Unsetenv("TLS_CACHE_DIR")

	cfg := &Config{}
	loadFromEnv(cfg)

	if cfg.TLSAuto != true {
		t.Errorf("TLSAuto = %v, want true", cfg.TLSAuto)
	}
	if cfg.TLSDomain != "example.com" {
		t.Errorf("TLSDomain = %s, want example.com", cfg.TLSDomain)
	}
	if cfg.TLSCacheDir != "/tmp/certs" {
		t.Errorf("TLSCacheDir = %s, want /tmp/certs", cfg.TLSCacheDir)
	}
}

func TestLoadFromEnv_TLS_AutoNumber(t *testing.T) {
	os.Setenv("TLS_AUTO", "1")
	defer os.Unsetenv("TLS_AUTO")

	cfg := &Config{}
	loadFromEnv(cfg)

	if cfg.TLSAuto != true {
		t.Errorf("TLSAuto = %v, want true", cfg.TLSAuto)
	}
}

func TestLoadFromEnv_Collector(t *testing.T) {
	os.Setenv("COLLECTOR_ENABLED", "false")
	os.Setenv("COLLECTOR_WORKERS", "20")
	defer os.Unsetenv("COLLECTOR_ENABLED")
	defer os.Unsetenv("COLLECTOR_WORKERS")

	cfg := &Config{}
	loadFromEnv(cfg)

	if cfg.CollectorEnabled != false {
		t.Errorf("CollectorEnabled = %v, want false", cfg.CollectorEnabled)
	}
	if cfg.CollectorWorkers != 20 {
		t.Errorf("CollectorWorkers = %d, want 20", cfg.CollectorWorkers)
	}
}

func TestLoadFromEnv_SyncInterval(t *testing.T) {
	os.Setenv("SYNC_INTERVAL", "10m")
	defer os.Unsetenv("SYNC_INTERVAL")

	cfg := &Config{}
	loadFromEnv(cfg)

	if cfg.SyncInterval != 10*time.Minute {
		t.Errorf("SyncInterval = %v, want 10m", cfg.SyncInterval)
	}
}

func TestLoadFromEnv_CollectorRuntimeIntervals(t *testing.T) {
	os.Setenv("COLLECTOR_DEVICE_SYNC_INTERVAL", "15s")
	os.Setenv("COLLECTOR_COMMAND_POLL_INTERVAL", "800ms")
	defer os.Unsetenv("COLLECTOR_DEVICE_SYNC_INTERVAL")
	defer os.Unsetenv("COLLECTOR_COMMAND_POLL_INTERVAL")

	cfg := &Config{}
	loadFromEnv(cfg)

	if cfg.CollectorDeviceSyncInterval != 15*time.Second {
		t.Errorf("CollectorDeviceSyncInterval = %v, want 15s", cfg.CollectorDeviceSyncInterval)
	}
	if cfg.CollectorCommandPollInterval != 800*time.Millisecond {
		t.Errorf("CollectorCommandPollInterval = %v, want 800ms", cfg.CollectorCommandPollInterval)
	}
}

func TestLoadFromEnv_NorthboundMQTTReconnectInterval(t *testing.T) {
	os.Setenv("NORTHBOUND_MQTT_RECONNECT_INTERVAL", "7s")
	defer os.Unsetenv("NORTHBOUND_MQTT_RECONNECT_INTERVAL")

	cfg := &Config{}
	loadFromEnv(cfg)

	if cfg.NorthboundMQTTReconnectInterval != 7*time.Second {
		t.Errorf("NorthboundMQTTReconnectInterval = %v, want 7s", cfg.NorthboundMQTTReconnectInterval)
	}
}

func TestLoadFromEnv_DriversDir(t *testing.T) {
	os.Setenv("DRIVERS_DIR", "/custom/drivers")
	os.Setenv("NORTHBOUND_PLUGINS_DIR", "/custom/plugins")
	defer os.Unsetenv("DRIVERS_DIR")
	defer os.Unsetenv("NORTHBOUND_PLUGINS_DIR")

	cfg := &Config{}
	loadFromEnv(cfg)

	if cfg.DriversDir != "/custom/drivers" {
		t.Errorf("DriversDir = %s, want /custom/drivers", cfg.DriversDir)
	}
	if cfg.NorthboundPluginsDir != "/custom/plugins" {
		t.Errorf("NorthboundPluginsDir = %s, want /custom/plugins", cfg.NorthboundPluginsDir)
	}
}

func TestLoadFromEnv_LogLevel(t *testing.T) {
	tests := []struct {
		envValue string
		expected string
	}{
		{"debug", "debug"},
		{"DEBUG", "DEBUG"},
		{"info", "info"},
		{"warn", "warn"},
		{"error", "error"},
	}

	for _, tt := range tests {
		t.Run(tt.envValue, func(t *testing.T) {
			os.Setenv("LOG_LEVEL", tt.envValue)
			defer os.Unsetenv("LOG_LEVEL")

			cfg := &Config{}
			loadFromEnv(cfg)

			if cfg.LogLevel != tt.expected {
				t.Errorf("LogLevel = %s, want %s", cfg.LogLevel, tt.expected)
			}
		})
	}
}

func TestLoadFromEnv_LogJSON(t *testing.T) {
	tests := []struct {
		envValue string
		expected bool
	}{
		{"true", true},
		{"TRUE", true},
		{"True", true},
		{"false", false},
		{"FALSE", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.envValue, func(t *testing.T) {
			os.Setenv("LOG_JSON", tt.envValue)
			defer os.Unsetenv("LOG_JSON")

			cfg := &Config{}
			loadFromEnv(cfg)

			if cfg.LogJSON != tt.expected {
				t.Errorf("LogJSON = %v, want %v for env=%s", cfg.LogJSON, tt.expected, tt.envValue)
			}
		})
	}
}

func TestLoadFromEnv_ThresholdCache(t *testing.T) {
	os.Setenv("THRESHOLD_CACHE_ENABLED", "false")
	os.Setenv("THRESHOLD_CACHE_TTL", "5m")
	defer os.Unsetenv("THRESHOLD_CACHE_ENABLED")
	defer os.Unsetenv("THRESHOLD_CACHE_TTL")

	cfg := &Config{}
	loadFromEnv(cfg)

	if cfg.ThresholdCacheEnabled != false {
		t.Errorf("ThresholdCacheEnabled = %v, want false", cfg.ThresholdCacheEnabled)
	}
	if cfg.ThresholdCacheTTL != 5*time.Minute {
		t.Errorf("ThresholdCacheTTL = %v, want 5m", cfg.ThresholdCacheTTL)
	}
}

func TestLoadFromEnv_MaxDataLimits(t *testing.T) {
	os.Setenv("MAX_DATA_POINTS", "200000")
	os.Setenv("MAX_DATA_CACHE", "50000")
	defer os.Unsetenv("MAX_DATA_POINTS")
	defer os.Unsetenv("MAX_DATA_CACHE")

	cfg := &Config{}
	loadFromEnv(cfg)

	if cfg.MaxDataPoints != 200000 {
		t.Errorf("MaxDataPoints = %d, want 200000", cfg.MaxDataPoints)
	}
	if cfg.MaxDataCache != 50000 {
		t.Errorf("MaxDataCache = %d, want 50000", cfg.MaxDataCache)
	}
}

func TestLoadFromEnv_DriverSerialConfig(t *testing.T) {
	os.Setenv("DRIVER_SERIAL_READ_TIMEOUT", "5s")
	os.Setenv("DRIVER_SERIAL_OPEN_RETRIES", "3")
	os.Setenv("DRIVER_SERIAL_OPEN_BACKOFF", "500ms")
	defer os.Unsetenv("DRIVER_SERIAL_READ_TIMEOUT")
	defer os.Unsetenv("DRIVER_SERIAL_OPEN_RETRIES")
	defer os.Unsetenv("DRIVER_SERIAL_OPEN_BACKOFF")

	cfg := &Config{}
	loadFromEnv(cfg)

	if cfg.DriverSerialReadTimeout != 5*time.Second {
		t.Errorf("DriverSerialReadTimeout = %v, want 5s", cfg.DriverSerialReadTimeout)
	}
	if cfg.DriverSerialOpenRetries != 3 {
		t.Errorf("DriverSerialOpenRetries = %d, want 3", cfg.DriverSerialOpenRetries)
	}
	if cfg.DriverSerialOpenBackoff != 500*time.Millisecond {
		t.Errorf("DriverSerialOpenBackoff = %v, want 500ms", cfg.DriverSerialOpenBackoff)
	}
}

func TestLoadFromEnv_DriverTCPConfig(t *testing.T) {
	os.Setenv("DRIVER_TCP_DIAL_TIMEOUT", "10s")
	os.Setenv("DRIVER_TCP_DIAL_RETRIES", "5")
	os.Setenv("DRIVER_TCP_DIAL_BACKOFF", "1s")
	os.Setenv("DRIVER_TCP_READ_TIMEOUT", "30s")
	defer os.Unsetenv("DRIVER_TCP_DIAL_TIMEOUT")
	defer os.Unsetenv("DRIVER_TCP_DIAL_RETRIES")
	defer os.Unsetenv("DRIVER_TCP_DIAL_BACKOFF")
	defer os.Unsetenv("DRIVER_TCP_READ_TIMEOUT")

	cfg := &Config{}
	loadFromEnv(cfg)

	if cfg.DriverTCPDialTimeout != 10*time.Second {
		t.Errorf("DriverTCPDialTimeout = %v, want 10s", cfg.DriverTCPDialTimeout)
	}
	if cfg.DriverTCPDialRetries != 5 {
		t.Errorf("DriverTCPDialRetries = %d, want 5", cfg.DriverTCPDialRetries)
	}
	if cfg.DriverTCPDialBackoff != time.Second {
		t.Errorf("DriverTCPDialBackoff = %v, want 1s", cfg.DriverTCPDialBackoff)
	}
	if cfg.DriverTCPReadTimeout != 30*time.Second {
		t.Errorf("DriverTCPReadTimeout = %v, want 30s", cfg.DriverTCPReadTimeout)
	}
}

func TestLoadFromEnv_InvalidTimeout(t *testing.T) {
	os.Setenv("HTTP_READ_TIMEOUT", "invalid")
	defer os.Unsetenv("HTTP_READ_TIMEOUT")

	cfg := &Config{}
	loadFromEnv(cfg)

	// 应该保持默认值
	if cfg.HTTPReadTimeout != 30*time.Second {
		t.Errorf("HTTPReadTimeout = %v, want 30s", cfg.HTTPReadTimeout)
	}
}

func TestLoadFromEnv_InvalidInt(t *testing.T) {
	os.Setenv("COLLECTOR_WORKERS", "not-a-number")
	defer os.Unsetenv("COLLECTOR_WORKERS")

	cfg := &Config{}
	loadFromEnv(cfg)

	// 应该保持默认值
	if cfg.CollectorWorkers != 10 {
		t.Errorf("CollectorWorkers = %d, want 10", cfg.CollectorWorkers)
	}
}

func TestLoadFromEnv_InvalidDuration(t *testing.T) {
	os.Setenv("SYNC_INTERVAL", "forever")
	defer os.Unsetenv("SYNC_INTERVAL")

	cfg := &Config{}
	loadFromEnv(cfg)

	// 应该保持默认值
	if cfg.SyncInterval != 5*time.Minute {
		t.Errorf("SyncInterval = %v, want 5m", cfg.SyncInterval)
	}
}

func TestLoadFromEnv_SessionSecret(t *testing.T) {
	os.Setenv("SESSION_SECRET", "my-secret-key")
	defer os.Unsetenv("SESSION_SECRET")

	cfg := &Config{}
	loadFromEnv(cfg)

	if cfg.SessionSecret != "my-secret-key" {
		t.Errorf("SessionSecret = %s, want my-secret-key", cfg.SessionSecret)
	}
}

func TestLoadFromEnv_CORS(t *testing.T) {
	os.Setenv("ALLOWED_ORIGINS", "http://a.com,http://b.com")
	defer os.Unsetenv("ALLOWED_ORIGINS")

	cfg := &Config{}
	loadFromEnv(cfg)

	if cfg.AllowedOrigins != "http://a.com,http://b.com" {
		t.Errorf("AllowedOrigins = %s, want http://a.com,http://b.com", cfg.AllowedOrigins)
	}
}

func TestLoadFromEnv_DBPaths(t *testing.T) {
	os.Setenv("PARAM_DB_PATH", "/custom/param.db")
	os.Setenv("DATA_DB_PATH", "/custom/data.db")
	defer os.Unsetenv("PARAM_DB_PATH")
	defer os.Unsetenv("DATA_DB_PATH")

	cfg := &Config{}
	loadFromEnv(cfg)

	if cfg.ParamDBPath != "/custom/param.db" {
		t.Errorf("ParamDBPath = %s, want /custom/param.db", cfg.ParamDBPath)
	}
	if cfg.DataDBPath != "/custom/data.db" {
		t.Errorf("DataDBPath = %s, want /custom/data.db", cfg.DataDBPath)
	}
}

func TestLoadFromEnv_TLSFiles(t *testing.T) {
	os.Setenv("TLS_CERT_FILE", "/certs/cert.pem")
	os.Setenv("TLS_KEY_FILE", "/certs/key.pem")
	defer os.Unsetenv("TLS_CERT_FILE")
	defer os.Unsetenv("TLS_KEY_FILE")

	cfg := &Config{}
	loadFromEnv(cfg)

	if cfg.TLSCertFile != "/certs/cert.pem" {
		t.Errorf("TLSCertFile = %s, want /certs/cert.pem", cfg.TLSCertFile)
	}
	if cfg.TLSKeyFile != "/certs/key.pem" {
		t.Errorf("TLSKeyFile = %s, want /certs/key.pem", cfg.TLSKeyFile)
	}
}
