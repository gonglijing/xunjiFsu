package service

import (
	"time"

	"github.com/gonglijing/xunjiFsu/internal/database"
	"github.com/gonglijing/xunjiFsu/internal/models"
)

type HistoryDataQuery struct {
	DeviceID  *int64
	FieldName string
	StartTime time.Time
	EndTime   time.Time
}

type DataService struct{}

func NewDataService() *DataService {
	return &DataService{}
}

func (s *DataService) LoadAllDataCache() ([]*models.DataCache, error) {
	return database.GetAllDataCache()
}

func (s *DataService) LoadDeviceDataCache(deviceID int64) ([]*models.DataCache, error) {
	return database.GetDataCacheByDeviceID(deviceID)
}

func (s *DataService) QueryHistoryData(query HistoryDataQuery) ([]*database.DataPoint, error) {
	if query.DeviceID != nil {
		if query.FieldName != "" {
			return database.GetDataPointsByDeviceFieldAndTime(*query.DeviceID, query.FieldName, query.StartTime, query.EndTime, 2000)
		}
		if !query.StartTime.IsZero() || !query.EndTime.IsZero() {
			return database.GetDataPointsByDeviceAndTimeLimit(*query.DeviceID, query.StartTime, query.EndTime, 2000)
		}
		return database.GetDataPointsByDevice(*query.DeviceID, 1000)
	}

	return database.GetLatestDataPoints(1000)
}

func (s *DataService) ClearHistoryPoint(deviceID int64, fieldName string) (int64, error) {
	return database.DeleteHistoryDataByPoint(deviceID, fieldName)
}
