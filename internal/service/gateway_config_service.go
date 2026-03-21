package service

import (
	"strings"

	"github.com/gonglijing/xunjiFsu/internal/database"
	"github.com/gonglijing/xunjiFsu/internal/models"
)

const DefaultGatewayName = "HuShu智能网关"

type GatewayConfigService struct{}

func NewGatewayConfigService() *GatewayConfigService {
	return &GatewayConfigService{}
}

func (s *GatewayConfigService) LoadGatewayConfig() (*database.GatewayConfig, error) {
	return database.GetGatewayConfig()
}

func (s *GatewayConfigService) UpdateGatewayConfig(cfg *models.GatewayConfig) (*database.GatewayConfig, error) {
	if err := database.UpdateGatewayConfig(BuildDatabaseGatewayConfig(cfg)); err != nil {
		return nil, err
	}
	return database.GetGatewayConfig()
}

func NormalizeGatewayConfigInput(cfg *models.GatewayConfig) {
	if cfg == nil {
		return
	}
	cfg.GatewayName = strings.TrimSpace(cfg.GatewayName)
	if cfg.GatewayName == "" {
		cfg.GatewayName = DefaultGatewayName
	}
	if cfg.DataRetentionDays <= 0 {
		cfg.DataRetentionDays = database.DefaultRetentionDays
	}
}

func BuildDatabaseGatewayConfig(cfg *models.GatewayConfig) *database.GatewayConfig {
	if cfg == nil {
		return nil
	}
	return &database.GatewayConfig{
		ID:                cfg.ID,
		GatewayName:       cfg.GatewayName,
		DataRetentionDays: cfg.DataRetentionDays,
	}
}
