// =============================================================================
// 配置模块单元测试
// =============================================================================
package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.ListenAddr != ":8080" {
		t.Errorf("ListenAddr = %s, want :8080", cfg.ListenAddr)
	}
	if cfg.TLSCertFile != "" {
		t.Errorf("TLSCertFile = %s, want empty", cfg.TLSCertFile)
	}
	if cfg.TLSKeyFile != "" {
		t.Errorf("TLSKeyFile = %s, want empty", cfg.TLSKeyFile)
	}
	if cfg.TLSAuto != false {
		t.Errorf("TLSAuto = %v, want false", cfg.TLSAuto)
	}
	if cfg.HTTPReadTimeout != 30*time.Second {
		t.Errorf("HTTPReadTimeout = %v, want 30s", cfg.HTTPReadTimeout)
	}
	if cfg.HTTPWriteTimeout != 30*time.Second {
		t.Errorf("HTTPWriteTimeout = %v, want 30s", cfg.HTTPWriteTimeout)
	}
	if cfg.HTTPIdleTimeout != 60*time.Second {
		t.Errorf("HTTPIdleTimeout = %v, want 60s", cfg.HTTPIdleTimeout)
	}
	if cfg.DBPath != "gogw.db" {
		t.Errorf("DBPath = %s, want gogw.db", cfg.DBPath)
	}
	if cfg.ParamDBPath != "param.db" {
		t.Errorf("ParamDBPath = %s, want param.db", cfg.ParamDBPath)
	}
	if cfg.DataDBPath != "data.db" {
		t.Errorf("DataDBPath = %s, want data.db", cfg.DataDBPath)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel = %s, want info", cfg.LogLevel)
	}
	if cfg.LogJSON != false {
		t.Errorf("LogJSON = %v, want false", cfg.LogJSON)
	}
	if cfg.CollectorEnabled != true {
		t.Errorf("CollectorEnabled = %v, want true", cfg.CollectorEnabled)
	}
	if cfg.CollectorWorkers != 4 {
		t.Errorf("CollectorWorkers = %d, want 4", cfg.CollectorWorkers)
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
	if cfg.DriversDir != "drivers" {
		t.Errorf("DriversDir = %s, want drivers", cfg.DriversDir)
	}
	if cfg.NorthboundPluginsDir != "plugin_north" {
		t.Errorf("NorthboundPluginsDir = %s, want plugin_north", cfg.NorthboundPluginsDir)
	}
	if cfg.NorthboundMQTTReconnectInterval != 5*time.Second {
		t.Errorf("NorthboundMQTTReconnectInterval = %v, want 5s", cfg.NorthboundMQTTReconnectInterval)
	}
	if cfg.ThresholdCacheEnabled != true {
		t.Errorf("ThresholdCacheEnabled = %v, want true", cfg.ThresholdCacheEnabled)
	}
	if cfg.ThresholdCacheTTL != time.Minute {
		t.Errorf("ThresholdCacheTTL = %v, want 1m", cfg.ThresholdCacheTTL)
	}
	if cfg.MaxDataPoints != 100000 {
		t.Errorf("MaxDataPoints = %d, want 100000", cfg.MaxDataPoints)
	}
	if cfg.MaxDataCache != 100000 {
		t.Errorf("MaxDataCache = %d, want 100000", cfg.MaxDataCache)
	}
}

func TestGetAllowedOrigins_Default(t *testing.T) {
	cfg := &Config{AllowedOrigins: ""}
	origins := cfg.GetAllowedOrigins()
	if len(origins) != 2 {
		t.Errorf("GetAllowedOrigins() returned %d origins, want 2", len(origins))
	}
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
	cfg := &Config{AllowedOrigins: "http://example.com,https://test.com"}
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
	cfg := &Config{AllowedOrigins: "http://single.com"}
	origins := cfg.GetAllowedOrigins()
	if len(origins) != 1 {
		t.Errorf("GetAllowedOrigins() returned %d origins, want 1", len(origins))
	}
}

func TestLoadFromEnv_ListenAddr(t *testing.T) {
	os.Setenv("LISTEN_ADDR", ":9090")
	defer os.Unsetenv("LISTEN_ADDR")

	cfg := &Config{}
	applyEnvConfig(cfg)

	if cfg.ListenAddr != ":9090" {
		t.Errorf("ListenAddr = %s, want :9090", cfg.ListenAddr)
	}
}

func TestLoadFromEnv_HTTPTimeout(t *testing.T) {
	os.Setenv("HTTP_READ_TIMEOUT", "60s")
	os.Setenv("HTTP_WRITE_TIMEOUT", "120s")
	defer os.Unsetenv("HTTP_READ_TIMEOUT")
	defer os.Unsetenv("HTTP_WRITE_TIMEOUT")

	cfg := &Config{}
	applyEnvConfig(cfg)

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
	applyEnvConfig(cfg)

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
	applyEnvConfig(cfg)

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
	applyEnvConfig(cfg)

	if cfg.TLSAuto != true {
		t.Errorf("TLSAuto = %v, want true", cfg.TLSAuto)
	}
}

func TestParseBoolAcceptingOne_TrueInputs(t *testing.T) {
	testCases := []string{"1", "true", " TRUE "}
	for _, input := range testCases {
		if !parseBoolAcceptingOne(input) {
			t.Fatalf("parseBoolAcceptingOne(%q) = false, want true", input)
		}
	}
}

func TestApplyEnvDurationWithFallback_RequiresPositiveDuration(t *testing.T) {
	os.Setenv("COLLECTOR_DEVICE_SYNC_INTERVAL", "0s")
	defer os.Unsetenv("COLLECTOR_DEVICE_SYNC_INTERVAL")

	var got time.Duration
	applyEnvDurationWithFallback(&got, "COLLECTOR_DEVICE_SYNC_INTERVAL", 10*time.Second, true)

	if got != 10*time.Second {
		t.Fatalf("applyEnvDurationWithFallback() = %v, want 10s", got)
	}
}

func TestLoadFromEnv_Collector(t *testing.T) {
	os.Setenv("COLLECTOR_ENABLED", "false")
	os.Setenv("COLLECTOR_WORKERS", "20")
	defer os.Unsetenv("COLLECTOR_ENABLED")
	defer os.Unsetenv("COLLECTOR_WORKERS")

	cfg := &Config{}
	applyEnvConfig(cfg)

	if cfg.CollectorEnabled != false {
		t.Errorf("CollectorEnabled = %v, want false", cfg.CollectorEnabled)
	}
	if cfg.CollectorWorkers != 20 {
		t.Errorf("CollectorWorkers = %d, want 20", cfg.CollectorWorkers)
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
	applyEnvConfig(cfg)

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
	applyEnvConfig(cfg)

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

func TestLoadFromEnv_ThresholdCache(t *testing.T) {
	os.Setenv("THRESHOLD_CACHE_ENABLED", "false")
	os.Setenv("THRESHOLD_CACHE_TTL", "5m")
	defer os.Unsetenv("THRESHOLD_CACHE_ENABLED")
	defer os.Unsetenv("THRESHOLD_CACHE_TTL")

	cfg := &Config{}
	applyEnvConfig(cfg)

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
	applyEnvConfig(cfg)

	if cfg.MaxDataPoints != 200000 {
		t.Errorf("MaxDataPoints = %d, want 200000", cfg.MaxDataPoints)
	}
	if cfg.MaxDataCache != 50000 {
		t.Errorf("MaxDataCache = %d, want 50000", cfg.MaxDataCache)
	}
}

func TestParseFlatYAML(t *testing.T) {
	data := []byte(`
server:
  addr: ":9090"
  read_timeout: 40s

drivers:
  serial_open_retries: 3 # inline comment
  tcp_dial_timeout: '9s'
`)

	flat, err := parseFlatYAML(data)
	if err != nil {
		t.Fatalf("parseFlatYAML error: %v", err)
	}

	if flat["server.addr"] != ":9090" {
		t.Fatalf("server.addr=%q, want :9090", flat["server.addr"])
	}
	if flat["server.read_timeout"] != "40s" {
		t.Fatalf("server.read_timeout=%q, want 40s", flat["server.read_timeout"])
	}
	if flat["drivers.serial_open_retries"] != "3" {
		t.Fatalf("drivers.serial_open_retries=%q, want 3", flat["drivers.serial_open_retries"])
	}
	if flat["drivers.tcp_dial_timeout"] != "9s" {
		t.Fatalf("drivers.tcp_dial_timeout=%q, want 9s", flat["drivers.tcp_dial_timeout"])
	}
}

func TestLoadFromFile_ParsesConfigYAML(t *testing.T) {
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, "config")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}

	content := `
server:
  addr: ":7777"
  read_timeout: 35s
  write_timeout: 45s

drivers:
  dir: "drivers-custom"
  call_timeout: 2s
  serial_read_timeout: 1s
  serial_open_retries: 4
  serial_open_backoff: 500ms
  tcp_dial_timeout: 3s
  tcp_dial_retries: 5
  tcp_dial_backoff: 700ms
  tcp_read_timeout: 8s

northbound:
  plugins_dir: "plugin_custom"
  mqtt_reconnect_interval: 9s

collector:
  workers: 6
  device_sync_interval: 12s
  command_poll_interval: 600ms

data:
  max_data_points: 300000
  max_data_cache: 120000
`
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(content), 0o644); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir temp: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWD) })

	cfg := DefaultConfig()
	if err := applyFileConfig(cfg); err != nil {
		t.Fatalf("applyFileConfig: %v", err)
	}

	if cfg.ListenAddr != ":7777" {
		t.Fatalf("ListenAddr=%s, want :7777", cfg.ListenAddr)
	}
	if cfg.HTTPReadTimeout != 35*time.Second {
		t.Fatalf("HTTPReadTimeout=%v, want 35s", cfg.HTTPReadTimeout)
	}
	if cfg.HTTPWriteTimeout != 45*time.Second {
		t.Fatalf("HTTPWriteTimeout=%v, want 45s", cfg.HTTPWriteTimeout)
	}
	if cfg.DriversDir != "drivers-custom" {
		t.Fatalf("DriversDir=%s, want drivers-custom", cfg.DriversDir)
	}
	if cfg.DriverCallTimeout != 2*time.Second {
		t.Fatalf("DriverCallTimeout=%v, want 2s", cfg.DriverCallTimeout)
	}
	if cfg.CollectorWorkers != 6 {
		t.Fatalf("CollectorWorkers=%d, want 6", cfg.CollectorWorkers)
	}
	if cfg.MaxDataPoints != 300000 {
		t.Fatalf("MaxDataPoints=%d, want 300000", cfg.MaxDataPoints)
	}
	if cfg.MaxDataCache != 120000 {
		t.Fatalf("MaxDataCache=%d, want 120000", cfg.MaxDataCache)
	}
}
