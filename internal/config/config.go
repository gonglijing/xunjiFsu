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
	LogLevel  string `json:"log_level"`
	LogJSON   bool   `json:"log_json"`

	// 采集器配置
	CollectorEnabled bool          `json:"collector_enabled"`
	CollectorWorkers int           `json:"collector_workers"`
	SyncInterval     time.Duration `json:"sync_interval"`

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
		ListenAddr: ":8080",
		HTTPReadTimeout:  30 * time.Second,
		HTTPWriteTimeout: 30 * time.Second,
		HTTPIdleTimeout:  60 * time.Second,
		DBPath:           "gogw.db",
		ParamDBPath:      "param.db",
		DataDBPath:       "data.db",
		SessionSecret:    "",
		AllowedOrigins:   "",
		LogLevel:         "info",
		LogJSON:          false,
		CollectorEnabled: true,
		CollectorWorkers: 10,
		SyncInterval:     5 * time.Minute,
		ThresholdCacheEnabled: true,
		ThresholdCacheTTL:     time.Minute,
		MaxDataPoints: 100000,
		MaxDataCache:  10000,
	}
}

// Load 从环境变量加载配置
func Load() (*Config, error) {
	cfg := DefaultConfig()

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

	return cfg, nil
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
	return fmt.Sprintf("Config{ListenAddr=%s, DBPath=%s, LogLevel=%s, CollectorEnabled=%v, SyncInterval=%v}",
		c.ListenAddr, c.DBPath, c.LogLevel, c.CollectorEnabled, c.SyncInterval)
}
