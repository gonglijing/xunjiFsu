package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config 应用配置
type Config struct {
	// 服务器配置
	ListenAddr string `json:"listen_addr"`
	// TLS/证书配置
	TLSCertFile string `json:"tls_cert_file"`
	TLSKeyFile  string `json:"tls_key_file"`
	TLSAuto     bool   `json:"tls_auto"`      // 是否启用自动申请（Let's Encrypt）
	TLSDomain   string `json:"tls_domain"`    // 自动证书域名
	TLSCacheDir string `json:"tls_cache_dir"` // 自动证书缓存目录

	// HTTP超时配置
	HTTPReadTimeout  time.Duration `json:"http_read_timeout"`
	HTTPWriteTimeout time.Duration `json:"http_write_timeout"`
	HTTPIdleTimeout  time.Duration `json:"http_idle_timeout"`

	// 数据库配置
	DBPath      string `json:"db_path"`
	ParamDBPath string `json:"param_db_path"`
	DataDBPath  string `json:"data_db_path"`

	// 会话配置
	SessionSecret string `json:"session_secret"`

	// CORS配置
	AllowedOrigins string `json:"allowed_origins"`

	// 日志配置
	LogLevel string `json:"log_level"`
	LogJSON  bool   `json:"log_json"`

	// 采集器配置
	CollectorEnabled             bool          `json:"collector_enabled"`
	CollectorWorkers             int           `json:"collector_workers"`
	SyncInterval                 time.Duration `json:"sync_interval"`
	CollectorDeviceSyncInterval  time.Duration `json:"collector_device_sync_interval"`
	CollectorCommandPollInterval time.Duration `json:"collector_command_poll_interval"`

	// 驱动目录
	DriversDir string `json:"drivers_dir"`

	// 北向插件目录
	NorthboundPluginsDir            string        `json:"northbound_plugins_dir"`
	NorthboundMQTTReconnectInterval time.Duration `json:"northbound_mqtt_reconnect_interval"`

	// 驱动执行配置
	DriverCallTimeout       time.Duration `json:"driver_call_timeout"`
	DriverSerialReadTimeout time.Duration `json:"driver_serial_read_timeout"`
	DriverSerialOpenRetries int           `json:"driver_serial_open_retries"`
	DriverSerialOpenBackoff time.Duration `json:"driver_serial_open_backoff"`
	DriverTCPDialTimeout    time.Duration `json:"driver_tcp_dial_timeout"`
	DriverTCPDialRetries    int           `json:"driver_tcp_dial_retries"`
	DriverTCPDialBackoff    time.Duration `json:"driver_tcp_dial_backoff"`
	DriverTCPReadTimeout    time.Duration `json:"driver_tcp_read_timeout"`

	// 阈值缓存配置
	ThresholdCacheEnabled bool          `json:"threshold_cache_enabled"`
	ThresholdCacheTTL     time.Duration `json:"threshold_cache_ttl"`

	// 内存数据库限制
	MaxDataPoints int `json:"max_data_points"`
	MaxDataCache  int `json:"max_data_cache"`
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		ListenAddr:                      ":8080",
		TLSCertFile:                     "",
		TLSKeyFile:                      "",
		TLSAuto:                         false,
		TLSDomain:                       "",
		TLSCacheDir:                     "cert-cache",
		HTTPReadTimeout:                 30 * time.Second,
		HTTPWriteTimeout:                30 * time.Second,
		HTTPIdleTimeout:                 60 * time.Second,
		DBPath:                          "gogw.db",
		ParamDBPath:                     "param.db",
		DataDBPath:                      "data.db",
		SessionSecret:                   "",
		AllowedOrigins:                  "",
		LogLevel:                        "info",
		LogJSON:                         false,
		CollectorEnabled:                true,
		CollectorWorkers:                10,
		SyncInterval:                    5 * time.Minute,
		CollectorDeviceSyncInterval:     10 * time.Second,
		CollectorCommandPollInterval:    500 * time.Millisecond,
		DriversDir:                      "drivers",
		NorthboundPluginsDir:            "plugin_north",
		NorthboundMQTTReconnectInterval: 5 * time.Second,
		DriverCallTimeout:               0,
		DriverSerialReadTimeout:         0,
		DriverSerialOpenRetries:         0,
		DriverSerialOpenBackoff:         0,
		DriverTCPDialTimeout:            0,
		DriverTCPDialRetries:            0,
		DriverTCPDialBackoff:            0,
		DriverTCPReadTimeout:            0,
		ThresholdCacheEnabled:           true,
		ThresholdCacheTTL:               time.Minute,
		MaxDataPoints:                   100000,
		MaxDataCache:                    10000,
	}
}

var defaultEnvConfig = DefaultConfig()

// Load 从配置文件和环境变量加载配置
func Load() (*Config, error) {
	cfg := DefaultConfig()

	// 1. 先从 YAML 文件加载配置
	if err := loadFromFile(cfg); err != nil {
		// 配置文件不存在或解析失败，使用默认配置（不报错）
		// fmt.Fprintf(os.Stderr, "Warning: Failed to load config file: %v\n", err)
	}

	// 2. 环境变量覆盖配置
	loadFromEnv(cfg)

	return cfg, nil
}

// loadFromFile 从 YAML 文件加载配置（仅解析本项目使用的简单层级键值）
func loadFromFile(cfg *Config) error {
	// 查找配置文件路径
	configPaths := []string{
		"config/config.yaml",
		"../config/config.yaml",
		"./config.yaml",
	}

	var configFile string
	for _, path := range configPaths {
		if _, err := os.Stat(path); err == nil {
			configFile = path
			break
		}
	}

	if configFile == "" {
		return fmt.Errorf("config file not found")
	}

	data, err := os.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	flatCfg, err := parseFlatYAML(data)
	if err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	// 应用服务器配置
	setStringIfNotEmpty(&cfg.ListenAddr, flatCfg["server.addr"])
	setDurationFromText(&cfg.HTTPReadTimeout, flatCfg["server.read_timeout"])
	setDurationFromText(&cfg.HTTPWriteTimeout, flatCfg["server.write_timeout"])

	setStringIfNotEmpty(&cfg.DriversDir, flatCfg["drivers.dir"])
	setStringIfNotEmpty(&cfg.NorthboundPluginsDir, flatCfg["northbound.plugins_dir"])
	setDurationFromText(&cfg.NorthboundMQTTReconnectInterval, flatCfg["northbound.mqtt_reconnect_interval"])

	setDurationFromText(&cfg.DriverCallTimeout, flatCfg["drivers.call_timeout"])
	setDurationFromText(&cfg.DriverSerialReadTimeout, flatCfg["drivers.serial_read_timeout"])
	setPositiveIntFromText(&cfg.DriverSerialOpenRetries, flatCfg["drivers.serial_open_retries"])
	setDurationFromText(&cfg.DriverSerialOpenBackoff, flatCfg["drivers.serial_open_backoff"])
	setDurationFromText(&cfg.DriverTCPDialTimeout, flatCfg["drivers.tcp_dial_timeout"])
	setPositiveIntFromText(&cfg.DriverTCPDialRetries, flatCfg["drivers.tcp_dial_retries"])
	setDurationFromText(&cfg.DriverTCPDialBackoff, flatCfg["drivers.tcp_dial_backoff"])
	setDurationFromText(&cfg.DriverTCPReadTimeout, flatCfg["drivers.tcp_read_timeout"])

	setDurationFromText(&cfg.CollectorDeviceSyncInterval, flatCfg["collector.device_sync_interval"])
	setDurationFromText(&cfg.CollectorCommandPollInterval, flatCfg["collector.command_poll_interval"])

	return nil
}

func parseFlatYAML(data []byte) (map[string]string, error) {
	lines := strings.Split(string(data), "\n")
	result := make(map[string]string)
	pathStack := make([]string, 0, 8)

	for index, raw := range lines {
		line := strings.TrimRight(raw, " \t\r")
		if strings.TrimSpace(line) == "" {
			continue
		}

		leadingSpaces := countLeadingSpaces(line)
		if leadingSpaces%2 != 0 {
			return nil, fmt.Errorf("invalid indentation at line %d", index+1)
		}

		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			continue
		}
		trimmed = strings.TrimSpace(stripYAMLComment(trimmed))
		if trimmed == "" {
			continue
		}

		colon := strings.IndexByte(trimmed, ':')
		if colon <= 0 {
			return nil, fmt.Errorf("invalid line %d", index+1)
		}

		key := strings.TrimSpace(trimmed[:colon])
		value := strings.TrimSpace(trimmed[colon+1:])
		if key == "" {
			return nil, fmt.Errorf("empty key at line %d", index+1)
		}

		depth := leadingSpaces / 2
		if depth > len(pathStack) {
			return nil, fmt.Errorf("invalid nesting at line %d", index+1)
		}
		if depth < len(pathStack) {
			pathStack = pathStack[:depth]
		}

		if value == "" {
			pathStack = append(pathStack, key)
			continue
		}

		path := strings.Join(append(pathStack, key), ".")
		result[path] = unquoteYAMLScalar(value)
	}

	return result, nil
}

func countLeadingSpaces(text string) int {
	count := 0
	for count < len(text) && text[count] == ' ' {
		count++
	}
	return count
}

func stripYAMLComment(text string) string {
	inSingle := false
	inDouble := false
	for i := 0; i < len(text); i++ {
		switch text[i] {
		case '\'':
			if !inDouble {
				inSingle = !inSingle
			}
		case '"':
			if !inSingle {
				inDouble = !inDouble
			}
		case '#':
			if !inSingle && !inDouble {
				return strings.TrimSpace(text[:i])
			}
		}
	}
	return text
}

func unquoteYAMLScalar(value string) string {
	if len(value) >= 2 {
		if value[0] == '"' && value[len(value)-1] == '"' {
			if unquoted, err := strconv.Unquote(value); err == nil {
				return unquoted
			}
			return value[1 : len(value)-1]
		}
		if value[0] == '\'' && value[len(value)-1] == '\'' {
			return value[1 : len(value)-1]
		}
	}
	return value
}

func setStringIfNotEmpty(dst *string, value string) {
	if dst == nil || value == "" {
		return
	}
	*dst = value
}

func setDurationFromText(dst *time.Duration, value string) {
	if dst == nil || value == "" {
		return
	}
	if parsed, err := time.ParseDuration(value); err == nil {
		*dst = parsed
	}
}

func setPositiveInt(dst *int, value int) {
	if dst == nil || value <= 0 {
		return
	}
	*dst = value
}

func setPositiveIntFromText(dst *int, value string) {
	if dst == nil || value == "" {
		return
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return
	}
	*dst = parsed
}

// loadFromEnv 从环境变量加载配置（会覆盖文件配置）
func loadFromEnv(cfg *Config) {
	if cfg == nil {
		return
	}

	defaults := defaultEnvConfig

	setStringFromEnv(&cfg.ListenAddr, "LISTEN_ADDR")

	setDurationFromEnvWithFallback(&cfg.HTTPReadTimeout, "HTTP_READ_TIMEOUT", defaults.HTTPReadTimeout, false)
	setDurationFromEnvWithFallback(&cfg.HTTPWriteTimeout, "HTTP_WRITE_TIMEOUT", defaults.HTTPWriteTimeout, false)
	setDurationFromEnv(&cfg.HTTPIdleTimeout, "HTTP_IDLE_TIMEOUT")

	setStringFromEnv(&cfg.DBPath, "DB_PATH")
	setStringFromEnv(&cfg.ParamDBPath, "PARAM_DB_PATH")
	setStringFromEnv(&cfg.DataDBPath, "DATA_DB_PATH")

	setStringFromEnv(&cfg.TLSCertFile, "TLS_CERT_FILE")
	setStringFromEnv(&cfg.TLSKeyFile, "TLS_KEY_FILE")
	setBoolFromEnvAllowOne(&cfg.TLSAuto, "TLS_AUTO")
	setStringFromEnv(&cfg.TLSDomain, "TLS_DOMAIN")
	setStringFromEnv(&cfg.TLSCacheDir, "TLS_CACHE_DIR")

	setStringFromEnv(&cfg.SessionSecret, "SESSION_SECRET")
	setStringFromEnv(&cfg.AllowedOrigins, "ALLOWED_ORIGINS")

	setStringFromEnv(&cfg.LogLevel, "LOG_LEVEL")
	setBoolFromEnv(&cfg.LogJSON, "LOG_JSON")

	setBoolFromEnv(&cfg.CollectorEnabled, "COLLECTOR_ENABLED")
	setIntFromEnvWithFallback(&cfg.CollectorWorkers, "COLLECTOR_WORKERS", defaults.CollectorWorkers)
	setDurationFromEnvWithFallback(&cfg.SyncInterval, "SYNC_INTERVAL", defaults.SyncInterval, false)
	setDurationFromEnvWithFallback(&cfg.CollectorDeviceSyncInterval, "COLLECTOR_DEVICE_SYNC_INTERVAL", defaults.CollectorDeviceSyncInterval, true)
	setDurationFromEnvWithFallback(&cfg.CollectorCommandPollInterval, "COLLECTOR_COMMAND_POLL_INTERVAL", defaults.CollectorCommandPollInterval, true)

	setStringFromEnv(&cfg.DriversDir, "DRIVERS_DIR")
	setStringFromEnv(&cfg.NorthboundPluginsDir, "NORTHBOUND_PLUGINS_DIR")
	setDurationFromEnvWithFallback(&cfg.NorthboundMQTTReconnectInterval, "NORTHBOUND_MQTT_RECONNECT_INTERVAL", defaults.NorthboundMQTTReconnectInterval, true)

	setDurationFromEnv(&cfg.DriverCallTimeout, "DRIVER_CALL_TIMEOUT")
	setDurationFromEnv(&cfg.DriverSerialReadTimeout, "DRIVER_SERIAL_READ_TIMEOUT")
	setIntFromEnv(&cfg.DriverSerialOpenRetries, "DRIVER_SERIAL_OPEN_RETRIES")
	setDurationFromEnv(&cfg.DriverSerialOpenBackoff, "DRIVER_SERIAL_OPEN_BACKOFF")
	setDurationFromEnv(&cfg.DriverTCPDialTimeout, "DRIVER_TCP_DIAL_TIMEOUT")
	setIntFromEnv(&cfg.DriverTCPDialRetries, "DRIVER_TCP_DIAL_RETRIES")
	setDurationFromEnv(&cfg.DriverTCPDialBackoff, "DRIVER_TCP_DIAL_BACKOFF")
	setDurationFromEnv(&cfg.DriverTCPReadTimeout, "DRIVER_TCP_READ_TIMEOUT")

	setBoolFromEnv(&cfg.ThresholdCacheEnabled, "THRESHOLD_CACHE_ENABLED")
	setDurationFromEnv(&cfg.ThresholdCacheTTL, "THRESHOLD_CACHE_TTL")

	setIntFromEnv(&cfg.MaxDataPoints, "MAX_DATA_POINTS")
	setIntFromEnv(&cfg.MaxDataCache, "MAX_DATA_CACHE")
}

func setStringFromEnv(dst *string, key string) {
	if dst == nil {
		return
	}
	if value, ok := envValue(key); ok {
		*dst = value
	}
}

func setBoolFromEnv(dst *bool, key string) {
	if dst == nil {
		return
	}
	if value, ok := envValue(key); ok {
		*dst = parseTrueBool(value)
	}
}

func setBoolFromEnvAllowOne(dst *bool, key string) {
	if dst == nil {
		return
	}
	if value, ok := envValue(key); ok {
		*dst = parseTrueBoolOrOne(value)
	}
}

func setIntFromEnv(dst *int, key string) {
	if dst == nil {
		return
	}
	value, ok := envValue(key)
	if !ok {
		return
	}
	if parsed, err := strconv.Atoi(value); err == nil {
		*dst = parsed
	}
}

func setIntFromEnvWithFallback(dst *int, key string, fallback int) {
	if dst == nil {
		return
	}
	value, ok := envValue(key)
	if !ok {
		return
	}
	if parsed, err := strconv.Atoi(value); err == nil {
		*dst = parsed
		return
	}
	if *dst == 0 {
		*dst = fallback
	}
}

func setDurationFromEnv(dst *time.Duration, key string) {
	if dst == nil {
		return
	}
	value, ok := envValue(key)
	if !ok {
		return
	}
	if parsed, err := time.ParseDuration(value); err == nil {
		*dst = parsed
	}
}

func setDurationFromEnvWithFallback(dst *time.Duration, key string, fallback time.Duration, mustPositive bool) {
	if dst == nil {
		return
	}
	value, ok := envValue(key)
	if !ok {
		return
	}
	if parsed, err := time.ParseDuration(value); err == nil {
		if !mustPositive || parsed > 0 {
			*dst = parsed
			return
		}
	}
	if *dst == 0 {
		*dst = fallback
	}
}

func envValue(key string) (string, bool) {
	value := os.Getenv(key)
	if value == "" {
		return "", false
	}
	return value, true
}

func parseTrueBool(value string) bool {
	return strings.EqualFold(strings.TrimSpace(value), "true")
}

func parseTrueBoolOrOne(value string) bool {
	trimmed := strings.TrimSpace(value)
	return strings.EqualFold(trimmed, "true") || trimmed == "1"
}

// GetAllowedOrigins 获取允许的跨域来源列表
func (c *Config) GetAllowedOrigins() []string {
	if c.AllowedOrigins == "" {
		return []string{"http://localhost:8080", "http://127.0.0.1:8080"}
	}
	return strings.Split(c.AllowedOrigins, ",")
}

// String 返回配置的字符串表示
func (c *Config) String() string {
	return fmt.Sprintf("Config{ListenAddr=%s, DBPath=%s, DriversDir=%s, LogLevel=%s, CollectorEnabled=%v, SyncInterval=%v}",
		c.ListenAddr, c.DBPath, c.DriversDir, c.LogLevel, c.CollectorEnabled, c.SyncInterval)
}
