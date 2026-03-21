package service

import (
	"github.com/gonglijing/xunjiFsu/internal/database"
	"github.com/gonglijing/xunjiFsu/internal/models"
	"github.com/gonglijing/xunjiFsu/internal/northbound"
)

type NorthboundRuntimeHooks struct {
	Rebuild func(*models.NorthboundConfig) error
}

type NorthboundService struct {
	manager *northbound.NorthboundManager
	hooks   NorthboundRuntimeHooks
}

func NewNorthboundService(manager *northbound.NorthboundManager, hooks NorthboundRuntimeHooks) *NorthboundService {
	return &NorthboundService{manager: manager, hooks: hooks}
}

func (s *NorthboundService) ListConfigs() ([]*models.NorthboundConfig, error) {
	return database.ListNorthboundConfigs()
}

func (s *NorthboundService) LoadConfig(id int64) (*models.NorthboundConfig, error) {
	return database.LoadNorthboundConfig(id)
}

func (s *NorthboundService) CreateConfig(config *models.NorthboundConfig) (*models.NorthboundConfig, error) {
	if config == nil {
		return nil, nil
	}
	if err := s.rebuildRuntime(config); err != nil {
		return nil, err
	}
	id, err := database.CreateNorthboundConfig(config)
	if err != nil {
		s.removeRuntime(config.Name)
		return nil, err
	}
	config.ID = id
	return config, nil
}

func (s *NorthboundService) UpdateConfig(oldConfig, newConfig *models.NorthboundConfig) (*models.NorthboundConfig, error) {
	if newConfig == nil {
		return nil, nil
	}
	newConfig.ID = oldConfig.ID
	if err := s.replaceRuntime(oldConfig, newConfig); err != nil {
		return nil, err
	}
	if err := database.UpdateNorthboundConfig(newConfig); err != nil {
		s.removeRuntime(newConfig.Name)
		s.rollbackRuntime(oldConfig)
		return nil, err
	}
	return newConfig, nil
}

func (s *NorthboundService) DeleteConfig(config *models.NorthboundConfig) error {
	if err := database.DeleteNorthboundConfig(config.ID); err != nil {
		return err
	}
	s.removeRuntime(config.Name)
	return nil
}
