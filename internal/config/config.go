package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
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

// loadFromFile 从 YAML 文件加载配置
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

	// 读取并解析 YAML 文件
	data, err := os.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// 临时结构体用于解析 YAML
	type yamlConfig struct {
		Server struct {
			Addr         string `yaml:"addr"`
			ReadTimeout  string `yaml:"read_timeout"`
			WriteTimeout string `yaml:"write_timeout"`
		} `yaml:"server"`
		Drivers struct {
			Dir               string `yaml:"dir"`
			CallTimeout       string `yaml:"call_timeout"`
			SerialReadTimeout string `yaml:"serial_read_timeout"`
			SerialOpenRetries int    `yaml:"serial_open_retries"`
			SerialOpenBackoff string `yaml:"serial_open_backoff"`
			TCPDialTimeout    string `yaml:"tcp_dial_timeout"`
			TCPDialRetries    int    `yaml:"tcp_dial_retries"`
			TCPDialBackoff    string `yaml:"tcp_dial_backoff"`
			TCPReadTimeout    string `yaml:"tcp_read_timeout"`
		} `yaml:"drivers"`
		Northbound struct {
			PluginsDir            string `yaml:"plugins_dir"`
			MQTTReconnectInterval string `yaml:"mqtt_reconnect_interval"`
		} `yaml:"northbound"`
		Auth struct {
			SessionMaxAge int `yaml:"session_max_age"`
		} `yaml:"auth"`
		Collector struct {
			DefaultInterval       int    `yaml:"default_interval"`
			DefaultUploadInterval int    `yaml:"default_upload_interval"`
			DeviceSyncInterval    string `yaml:"device_sync_interval"`
			CommandPollInterval   string `yaml:"command_poll_interval"`
		} `yaml:"collector"`
		Logging struct {
			Level  string `yaml:"level"`
			Format string `yaml:"format"`
		} `yaml:"logging"`
	}

	var yamlCfg yamlConfig
	if err := yaml.Unmarshal(data, &yamlCfg); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	// 应用服务器配置
	setStringIfNotEmpty(&cfg.ListenAddr, yamlCfg.Server.Addr)
	setDurationFromText(&cfg.HTTPReadTimeout, yamlCfg.Server.ReadTimeout)
	setDurationFromText(&cfg.HTTPWriteTimeout, yamlCfg.Server.WriteTimeout)

	setStringIfNotEmpty(&cfg.DriversDir, yamlCfg.Drivers.Dir)
	setStringIfNotEmpty(&cfg.NorthboundPluginsDir, yamlCfg.Northbound.PluginsDir)
	setDurationFromText(&cfg.NorthboundMQTTReconnectInterval, yamlCfg.Northbound.MQTTReconnectInterval)

	setDurationFromText(&cfg.DriverCallTimeout, yamlCfg.Drivers.CallTimeout)
	setDurationFromText(&cfg.DriverSerialReadTimeout, yamlCfg.Drivers.SerialReadTimeout)
	setPositiveInt(&cfg.DriverSerialOpenRetries, yamlCfg.Drivers.SerialOpenRetries)
	setDurationFromText(&cfg.DriverSerialOpenBackoff, yamlCfg.Drivers.SerialOpenBackoff)
	setDurationFromText(&cfg.DriverTCPDialTimeout, yamlCfg.Drivers.TCPDialTimeout)
	setPositiveInt(&cfg.DriverTCPDialRetries, yamlCfg.Drivers.TCPDialRetries)
	setDurationFromText(&cfg.DriverTCPDialBackoff, yamlCfg.Drivers.TCPDialBackoff)
	setDurationFromText(&cfg.DriverTCPReadTimeout, yamlCfg.Drivers.TCPReadTimeout)

	setDurationFromText(&cfg.CollectorDeviceSyncInterval, yamlCfg.Collector.DeviceSyncInterval)
	setDurationFromText(&cfg.CollectorCommandPollInterval, yamlCfg.Collector.CommandPollInterval)

	return nil
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
