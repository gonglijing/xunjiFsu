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

const (
	defaultGatewayConfigName  = "HuShu智能网关"
	selectGatewayConfigFields = `SELECT id, COALESCE(product_key, ''), COALESCE(device_key, ''), COALESCE(gateway_name, 'HuShu智能网关'),
		COALESCE(data_retention_days, ?), updated_at FROM gateway_config`
)

var gatewayColumnsEnsured bool

// InitGatewayConfigTable 创建网关配置表
func InitGatewayConfigTable() error {
	if err := createGatewayConfigTable(); err != nil {
		return err
	}
	if err := ensureGatewayConfigColumns(); err != nil {
		return err
	}
	return ensureDefaultGatewayConfig()
}

func createGatewayConfigTable() error {
	_, err := ParamDB.Exec(`CREATE TABLE IF NOT EXISTS gateway_config (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		product_key TEXT NOT NULL,
		device_key TEXT NOT NULL,
		gateway_name TEXT DEFAULT 'HuShu智能网关',
		data_retention_days INTEGER DEFAULT 30,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`)
	return err
}

func ensureDefaultGatewayConfig() error {
	var count int
	err := ParamDB.QueryRow("SELECT COUNT(*) FROM gateway_config").Scan(&count)
	if err != nil {
		return err
	}
	if count == 0 {
		_, err = ParamDB.Exec(
			`INSERT INTO gateway_config (product_key, device_key, gateway_name, data_retention_days) VALUES ('', '', ?, ?)`,
			defaultGatewayConfigName,
			DefaultRetentionDays,
		)
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

	cfg, err := loadGatewayConfig(selectGatewayConfigFields+" ORDER BY id LIMIT 1", DefaultRetentionDays)
	if err != nil {
		return nil, err
	}
	normalizeGatewayConfig(cfg)
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

	normalizeGatewayConfig(cfg)

	targetID, err := resolveGatewayConfigID(cfg.ID)
	if err != nil {
		return err
	}

	_, err = ParamDB.Exec(`UPDATE gateway_config SET product_key = ?, device_key = ?, gateway_name = ?, data_retention_days = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		cfg.ProductKey, cfg.DeviceKey, cfg.GatewayName, cfg.DataRetentionDays, targetID)
	return err
}

type gatewayConfigScanner interface {
	Scan(dest ...any) error
}

func loadGatewayConfig(query string, args ...any) (*GatewayConfig, error) {
	cfg := &GatewayConfig{}
	err := scanGatewayConfig(ParamDB.QueryRow(query, args...), cfg)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

func scanGatewayConfig(scanner gatewayConfigScanner, cfg *GatewayConfig) error {
	return scanner.Scan(
		&cfg.ID,
		&cfg.ProductKey,
		&cfg.DeviceKey,
		&cfg.GatewayName,
		&cfg.DataRetentionDays,
		&cfg.UpdatedAt,
	)
}

func normalizeGatewayConfig(cfg *GatewayConfig) {
	if cfg == nil {
		return
	}
	cfg.GatewayName = strings.TrimSpace(cfg.GatewayName)
	if cfg.GatewayName == "" {
		cfg.GatewayName = defaultGatewayConfigName
	}
	if cfg.DataRetentionDays <= 0 {
		cfg.DataRetentionDays = DefaultRetentionDays
	}
}

func resolveGatewayConfigID(id int64) (int64, error) {
	if id > 0 {
		return id, nil
	}

	current, err := GetGatewayConfig()
	if err != nil {
		return 0, err
	}
	return current.ID, nil
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
		return defaultGatewayConfigName
	}
	if cfg.GatewayName == "" {
		return defaultGatewayConfigName
	}
	return cfg.GatewayName
}
