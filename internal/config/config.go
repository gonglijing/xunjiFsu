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
	CollectorEnabled bool          `json:"collector_enabled"`
	CollectorWorkers int           `json:"collector_workers"`
	SyncInterval     time.Duration `json:"sync_interval"`

	// 驱动目录
	DriversDir string `json:"drivers_dir"`

	// 北向插件目录
	NorthboundPluginsDir string `json:"northbound_plugins_dir"`

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
		ListenAddr:              ":8080",
		TLSCertFile:             "",
		TLSKeyFile:              "",
		TLSAuto:                 false,
		TLSDomain:               "",
		TLSCacheDir:             "cert-cache",
		HTTPReadTimeout:         30 * time.Second,
		HTTPWriteTimeout:        30 * time.Second,
		HTTPIdleTimeout:         60 * time.Second,
		DBPath:                  "gogw.db",
		ParamDBPath:             "param.db",
		DataDBPath:              "data.db",
		SessionSecret:           "",
		AllowedOrigins:          "",
		LogLevel:                "info",
		LogJSON:                 false,
		CollectorEnabled:        true,
		CollectorWorkers:        10,
		SyncInterval:            5 * time.Minute,
		DriversDir:              "drivers",
		NorthboundPluginsDir:    "plugin_north",
		DriverCallTimeout:       0,
		DriverSerialReadTimeout: 0,
		DriverSerialOpenRetries: 0,
		DriverSerialOpenBackoff: 0,
		DriverTCPDialTimeout:    0,
		DriverTCPDialRetries:    0,
		DriverTCPDialBackoff:    0,
		DriverTCPReadTimeout:    0,
		ThresholdCacheEnabled:   true,
		ThresholdCacheTTL:       time.Minute,
		MaxDataPoints:           100000,
		MaxDataCache:            10000,
	}
}

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
			PluginsDir string `yaml:"plugins_dir"`
		} `yaml:"northbound"`
		Auth struct {
			SessionMaxAge int `yaml:"session_max_age"`
		} `yaml:"auth"`
		Collector struct {
			DefaultInterval       int `yaml:"default_interval"`
			DefaultUploadInterval int `yaml:"default_upload_interval"`
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
	if yamlCfg.Server.Addr != "" {
		cfg.ListenAddr = yamlCfg.Server.Addr
	}
	if yamlCfg.Server.ReadTimeout != "" {
		if timeout, err := time.ParseDuration(yamlCfg.Server.ReadTimeout); err == nil {
			cfg.HTTPReadTimeout = timeout
		}
	}
	if yamlCfg.Server.WriteTimeout != "" {
		if timeout, err := time.ParseDuration(yamlCfg.Server.WriteTimeout); err == nil {
			cfg.HTTPWriteTimeout = timeout
		}
	}
	if yamlCfg.Drivers.Dir != "" {
		cfg.DriversDir = yamlCfg.Drivers.Dir
	}
	if yamlCfg.Northbound.PluginsDir != "" {
		cfg.NorthboundPluginsDir = yamlCfg.Northbound.PluginsDir
	}
	if yamlCfg.Drivers.CallTimeout != "" {
		if timeout, err := time.ParseDuration(yamlCfg.Drivers.CallTimeout); err == nil {
			cfg.DriverCallTimeout = timeout
		}
	}
	if yamlCfg.Drivers.SerialReadTimeout != "" {
		if timeout, err := time.ParseDuration(yamlCfg.Drivers.SerialReadTimeout); err == nil {
			cfg.DriverSerialReadTimeout = timeout
		}
	}
	if yamlCfg.Drivers.SerialOpenRetries > 0 {
		cfg.DriverSerialOpenRetries = yamlCfg.Drivers.SerialOpenRetries
	}
	if yamlCfg.Drivers.SerialOpenBackoff != "" {
		if timeout, err := time.ParseDuration(yamlCfg.Drivers.SerialOpenBackoff); err == nil {
			cfg.DriverSerialOpenBackoff = timeout
		}
	}
	if yamlCfg.Drivers.TCPDialTimeout != "" {
		if timeout, err := time.ParseDuration(yamlCfg.Drivers.TCPDialTimeout); err == nil {
			cfg.DriverTCPDialTimeout = timeout
		}
	}
	if yamlCfg.Drivers.TCPDialRetries > 0 {
		cfg.DriverTCPDialRetries = yamlCfg.Drivers.TCPDialRetries
	}
	if yamlCfg.Drivers.TCPDialBackoff != "" {
		if timeout, err := time.ParseDuration(yamlCfg.Drivers.TCPDialBackoff); err == nil {
			cfg.DriverTCPDialBackoff = timeout
		}
	}
	if yamlCfg.Drivers.TCPReadTimeout != "" {
		if timeout, err := time.ParseDuration(yamlCfg.Drivers.TCPReadTimeout); err == nil {
			cfg.DriverTCPReadTimeout = timeout
		}
	}

	return nil
}

// loadFromEnv 从环境变量加载配置（会覆盖文件配置）
func loadFromEnv(cfg *Config) {

	// 服务器配置
	if v := os.Getenv("LISTEN_ADDR"); v != "" {
		cfg.ListenAddr = v
	}

	// HTTP超时配置
	if v := os.Getenv("HTTP_READ_TIMEOUT"); v != "" {
		if timeout, err := time.ParseDuration(v); err == nil {
			cfg.HTTPReadTimeout = timeout
		}
	}
	if v := os.Getenv("HTTP_WRITE_TIMEOUT"); v != "" {
		if timeout, err := time.ParseDuration(v); err == nil {
			cfg.HTTPWriteTimeout = timeout
		}
	}
	if v := os.Getenv("HTTP_IDLE_TIMEOUT"); v != "" {
		if timeout, err := time.ParseDuration(v); err == nil {
			cfg.HTTPIdleTimeout = timeout
		}
	}

	// 数据库配置
	if v := os.Getenv("DB_PATH"); v != "" {
		cfg.DBPath = v
	}
	if v := os.Getenv("PARAM_DB_PATH"); v != "" {
		cfg.ParamDBPath = v
	}
	if v := os.Getenv("DATA_DB_PATH"); v != "" {
		cfg.DataDBPath = v
	}

	// TLS
	if v := os.Getenv("TLS_CERT_FILE"); v != "" {
		cfg.TLSCertFile = v
	}
	if v := os.Getenv("TLS_KEY_FILE"); v != "" {
		cfg.TLSKeyFile = v
	}
	if v := os.Getenv("TLS_AUTO"); strings.ToLower(v) == "true" || v == "1" {
		cfg.TLSAuto = true
	}
	if v := os.Getenv("TLS_DOMAIN"); v != "" {
		cfg.TLSDomain = v
	}
	if v := os.Getenv("TLS_CACHE_DIR"); v != "" {
		cfg.TLSCacheDir = v
	}

	// 会话配置
	if v := os.Getenv("SESSION_SECRET"); v != "" {
		cfg.SessionSecret = v
	}

	// CORS配置
	if v := os.Getenv("ALLOWED_ORIGINS"); v != "" {
		cfg.AllowedOrigins = v
	}

	// 日志配置
	if v := os.Getenv("LOG_LEVEL"); v != "" {
		cfg.LogLevel = v
	}
	if v := os.Getenv("LOG_JSON"); v != "" {
		cfg.LogJSON = strings.ToLower(v) == "true"
	}

	// 采集器配置
	if v := os.Getenv("COLLECTOR_ENABLED"); v != "" {
		cfg.CollectorEnabled = strings.ToLower(v) == "true"
	}
	if v := os.Getenv("COLLECTOR_WORKERS"); v != "" {
		if workers, err := strconv.Atoi(v); err == nil {
			cfg.CollectorWorkers = workers
		}
	}
	if v := os.Getenv("SYNC_INTERVAL"); v != "" {
		if interval, err := time.ParseDuration(v); err == nil {
			cfg.SyncInterval = interval
		}
	}

	// 驱动目录
	if v := os.Getenv("DRIVERS_DIR"); v != "" {
		cfg.DriversDir = v
	}
	if v := os.Getenv("NORTHBOUND_PLUGINS_DIR"); v != "" {
		cfg.NorthboundPluginsDir = v
	}

	// 驱动执行配置
	if v := os.Getenv("DRIVER_CALL_TIMEOUT"); v != "" {
		if timeout, err := time.ParseDuration(v); err == nil {
			cfg.DriverCallTimeout = timeout
		}
	}
	if v := os.Getenv("DRIVER_SERIAL_READ_TIMEOUT"); v != "" {
		if timeout, err := time.ParseDuration(v); err == nil {
			cfg.DriverSerialReadTimeout = timeout
		}
	}
	if v := os.Getenv("DRIVER_SERIAL_OPEN_RETRIES"); v != "" {
		if retries, err := strconv.Atoi(v); err == nil {
			cfg.DriverSerialOpenRetries = retries
		}
	}
	if v := os.Getenv("DRIVER_SERIAL_OPEN_BACKOFF"); v != "" {
		if timeout, err := time.ParseDuration(v); err == nil {
			cfg.DriverSerialOpenBackoff = timeout
		}
	}
	if v := os.Getenv("DRIVER_TCP_DIAL_TIMEOUT"); v != "" {
		if timeout, err := time.ParseDuration(v); err == nil {
			cfg.DriverTCPDialTimeout = timeout
		}
	}
	if v := os.Getenv("DRIVER_TCP_DIAL_RETRIES"); v != "" {
		if retries, err := strconv.Atoi(v); err == nil {
			cfg.DriverTCPDialRetries = retries
		}
	}
	if v := os.Getenv("DRIVER_TCP_DIAL_BACKOFF"); v != "" {
		if timeout, err := time.ParseDuration(v); err == nil {
			cfg.DriverTCPDialBackoff = timeout
		}
	}
	if v := os.Getenv("DRIVER_TCP_READ_TIMEOUT"); v != "" {
		if timeout, err := time.ParseDuration(v); err == nil {
			cfg.DriverTCPReadTimeout = timeout
		}
	}

	// 阈值缓存配置
	if v := os.Getenv("THRESHOLD_CACHE_ENABLED"); v != "" {
		cfg.ThresholdCacheEnabled = strings.ToLower(v) == "true"
	}
	if v := os.Getenv("THRESHOLD_CACHE_TTL"); v != "" {
		if ttl, err := time.ParseDuration(v); err == nil {
			cfg.ThresholdCacheTTL = ttl
		}
	}

	// 内存数据库限制
	if v := os.Getenv("MAX_DATA_POINTS"); v != "" {
		if max, err := strconv.Atoi(v); err == nil {
			cfg.MaxDataPoints = max
		}
	}
	if v := os.Getenv("MAX_DATA_CACHE"); v != "" {
		if max, err := strconv.Atoi(v); err == nil {
			cfg.MaxDataCache = max
		}
	}
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
