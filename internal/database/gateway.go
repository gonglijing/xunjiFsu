package database

import "strings"

// GatewayConfig 网关配置
type GatewayConfig struct {
	ID                int64  `json:"id" db:"id"`
	ProductKey        string `json:"product_key" db:"product_key"`
	DeviceKey         string `json:"device_key" db:"device_key"`
	GatewayName       string `json:"gateway_name" db:"gateway_name"`
	DataRetentionDays int    `json:"data_retention_days" db:"data_retention_days"`
	UpdatedAt         string `json:"updated_at" db:"updated_at"`
}

var gatewayColumnsEnsured bool

// InitGatewayConfigTable 创建网关配置表
func InitGatewayConfigTable() error {
	_, err := ParamDB.Exec(`CREATE TABLE IF NOT EXISTS gateway_config (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		product_key TEXT NOT NULL,
		device_key TEXT NOT NULL,
		gateway_name TEXT DEFAULT 'HuShu智能网关',
		data_retention_days INTEGER DEFAULT 30,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		return err
	}

	if err := ensureGatewayConfigColumns(); err != nil {
		return err
	}

	// 确保有默认配置
	var count int
	err = ParamDB.QueryRow("SELECT COUNT(*) FROM gateway_config").Scan(&count)
	if err != nil {
		return err
	}
	if count == 0 {
		_, err = ParamDB.Exec(`INSERT INTO gateway_config (product_key, device_key, gateway_name, data_retention_days) VALUES ('', '', 'HuShu智能网关', ?)`, DefaultRetentionDays)
		if err != nil {
			return err
		}
	}

	return nil
}

func ensureGatewayConfigColumns() error {
	if gatewayColumnsEnsured {
		return nil
	}

	hasDataRetentionDays, err := columnExists(ParamDB, "gateway_config", "data_retention_days")
	if err != nil {
		return err
	}
	if !hasDataRetentionDays {
		if _, err := ParamDB.Exec(`ALTER TABLE gateway_config ADD COLUMN data_retention_days INTEGER DEFAULT 30`); err != nil {
			return err
		}
	}

	if _, err := ParamDB.Exec(`UPDATE gateway_config SET data_retention_days = ? WHERE data_retention_days IS NULL OR data_retention_days <= 0`, DefaultRetentionDays); err != nil {
		return err
	}

	gatewayColumnsEnsured = true
	return nil
}

// GetGatewayConfig 获取网关配置
func GetGatewayConfig() (*GatewayConfig, error) {
	if err := ensureGatewayConfigColumns(); err != nil {
		return nil, err
	}

	cfg := &GatewayConfig{}
	err := ParamDB.QueryRow(`SELECT id, COALESCE(product_key, ''), COALESCE(device_key, ''), COALESCE(gateway_name, 'HuShu智能网关'), COALESCE(data_retention_days, ?), updated_at FROM gateway_config ORDER BY id LIMIT 1`, DefaultRetentionDays).Scan(
		&cfg.ID, &cfg.ProductKey, &cfg.DeviceKey, &cfg.GatewayName, &cfg.DataRetentionDays, &cfg.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if cfg.DataRetentionDays <= 0 {
		cfg.DataRetentionDays = DefaultRetentionDays
	}
	return cfg, nil
}

// UpdateGatewayConfig 更新网关配置
func UpdateGatewayConfig(cfg *GatewayConfig) error {
	if cfg == nil {
		return nil
	}
	if err := ensureGatewayConfigColumns(); err != nil {
		return err
	}

	if strings.TrimSpace(cfg.GatewayName) == "" {
		cfg.GatewayName = "HuShu智能网关"
	}
	if cfg.DataRetentionDays <= 0 {
		cfg.DataRetentionDays = DefaultRetentionDays
	}

	targetID := cfg.ID
	if targetID <= 0 {
		current, err := GetGatewayConfig()
		if err != nil {
			return err
		}
		targetID = current.ID
	}

	_, err := ParamDB.Exec(`UPDATE gateway_config SET product_key = ?, device_key = ?, gateway_name = ?, data_retention_days = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		cfg.ProductKey, cfg.DeviceKey, cfg.GatewayName, cfg.DataRetentionDays, targetID)
	return err
}

// GetGatewayDataRetentionDays 获取网关全局历史保留天数
func GetGatewayDataRetentionDays() int {
	cfg, err := GetGatewayConfig()
	if err != nil || cfg == nil || cfg.DataRetentionDays <= 0 {
		return DefaultRetentionDays
	}
	return cfg.DataRetentionDays
}

// GetGatewayIdentity 获取网关身份信息
func GetGatewayIdentity() (string, string) {
	cfg, err := GetGatewayConfig()
	if err != nil || cfg == nil {
		return "", ""
	}
	return strings.TrimSpace(cfg.ProductKey), strings.TrimSpace(cfg.DeviceKey)
}

// GetGatewayProductKey 获取产品密钥
func GetGatewayProductKey() string {
	cfg, err := GetGatewayConfig()
	if err != nil {
		return ""
	}
	return cfg.ProductKey
}

// GetGatewayDeviceKey 获取设备密钥
func GetGatewayDeviceKey() string {
	cfg, err := GetGatewayConfig()
	if err != nil {
		return ""
	}
	return cfg.DeviceKey
}

// GetGatewayName 获取网关名称
func GetGatewayName() string {
	cfg, err := GetGatewayConfig()
	if err != nil {
		return "HuShu智能网关"
	}
	if cfg.GatewayName == "" {
		return "HuShu智能网关"
	}
	return cfg.GatewayName
}
