package service

import (
	"fmt"

	"github.com/gonglijing/xunjiFsu/internal/database"
	"github.com/gonglijing/xunjiFsu/internal/models"
)

func (s *NorthboundService) ToggleEnabled(config *models.NorthboundConfig) (int, error) {
	prevState := config.Enabled
	nextState := 0
	if prevState == 0 {
		nextState = 1
	}

	if err := database.UpdateNorthboundEnabled(config.ID, nextState); err != nil {
		return 0, err
	}

	config.Enabled = nextState
	if err := s.rebuildRuntime(config); err != nil {
		_ = database.UpdateNorthboundEnabled(config.ID, prevState)
		config.Enabled = prevState
		_ = s.rebuildRuntime(config)
		return 0, err
	}
	return nextState, nil
}

func (s *NorthboundService) ReloadConfig(config *models.NorthboundConfig) error {
	return s.rebuildRuntime(config)
}

func (s *NorthboundService) RuntimeAdapterForConfig(config *models.NorthboundConfig) (any, error) {
	if s.manager == nil {
		return nil, fmt.Errorf("northbound manager is nil")
	}
	adapter, err := s.manager.GetAdapter(config.Name)
	if err == nil {
		return adapter, nil
	}
	if rebuildErr := s.rebuildRuntime(config); rebuildErr != nil {
		return nil, fmt.Errorf("rebuild runtime failed: %w", rebuildErr)
	}
	adapter, err = s.manager.GetAdapter(config.Name)
	if err != nil {
		return nil, fmt.Errorf("get adapter failed: %w", err)
	}
	return adapter, nil
}

func (s *NorthboundService) rebuildRuntime(config *models.NorthboundConfig) error {
	if s == nil || s.hooks.Rebuild == nil {
		return nil
	}
	return s.hooks.Rebuild(config)
}

func (s *NorthboundService) rollbackRuntime(oldConfig *models.NorthboundConfig) {
	if oldConfig != nil {
		_ = s.rebuildRuntime(oldConfig)
	}
}

func (s *NorthboundService) replaceRuntime(oldConfig, newConfig *models.NorthboundConfig) error {
	if oldConfig != nil && oldConfig.Name != newConfig.Name {
		s.removeRuntime(oldConfig.Name)
	}
	if err := s.rebuildRuntime(newConfig); err != nil {
		s.rollbackRuntime(oldConfig)
		return err
	}
	return nil
}

func (s *NorthboundService) removeRuntime(name string) {
	if s.manager != nil {
		s.manager.RemoveAdapter(name)
	}
}
