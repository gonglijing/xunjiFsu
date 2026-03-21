package service

import (
	"github.com/gonglijing/xunjiFsu/internal/collector"
	"github.com/gonglijing/xunjiFsu/internal/database"
	"github.com/gonglijing/xunjiFsu/internal/models"
)

type ThresholdService struct{}

func NewThresholdService() *ThresholdService {
	return &ThresholdService{}
}

func (s *ThresholdService) ListThresholds() ([]*models.Threshold, error) {
	return database.GetAllThresholds()
}

func (s *ThresholdService) LoadThreshold(id int64) (*models.Threshold, error) {
	return database.GetThresholdByID(id)
}

func (s *ThresholdService) CreateThreshold(threshold *models.Threshold) (*models.Threshold, error) {
	if threshold == nil {
		return nil, nil
	}
	id, err := database.CreateThreshold(threshold)
	if err != nil {
		return nil, err
	}
	InvalidateThresholdDeviceCaches(threshold, nil)
	threshold.ID = id
	return threshold, nil
}

func (s *ThresholdService) UpdateThreshold(threshold *models.Threshold) (*models.Threshold, error) {
	if threshold == nil {
		return nil, nil
	}
	oldThreshold, _ := database.GetThresholdByID(threshold.ID)
	if err := database.UpdateThreshold(threshold); err != nil {
		return nil, err
	}
	InvalidateThresholdDeviceCaches(threshold, oldThreshold)
	return threshold, nil
}

func (s *ThresholdService) DeleteThreshold(id int64) error {
	threshold, _ := database.GetThresholdByID(id)
	if err := database.DeleteThreshold(id); err != nil {
		return err
	}
	InvalidateThresholdDeviceCaches(nil, threshold)
	return nil
}

func (s *ThresholdService) LoadAlarmRepeatInterval() (int, error) {
	return database.GetAlarmRepeatIntervalSeconds()
}

func (s *ThresholdService) UpdateAlarmRepeatInterval(seconds int) error {
	if err := ValidateAlarmRepeatIntervalSeconds(seconds); err != nil {
		return err
	}
	if err := database.UpdateAlarmRepeatIntervalSeconds(seconds); err != nil {
		return err
	}
	collector.InvalidateAlarmRepeatIntervalCache()
	return nil
}
