package service

import (
	"github.com/gonglijing/xunjiFsu/internal/database"
	"github.com/gonglijing/xunjiFsu/internal/models"
)

type AlarmService struct{}

func NewAlarmService() *AlarmService {
	return &AlarmService{}
}

func (s *AlarmService) ListRecentAlarmLogs(limit int) ([]*models.AlarmLog, error) {
	return database.ListRecentAlarmLogs(limit)
}

func (s *AlarmService) LoadAlarm(id int64) (*models.AlarmLog, error) {
	return database.LoadAlarmLog(id)
}

func (s *AlarmService) AcknowledgeAlarm(id int64, acknowledgedBy string) error {
	return database.AcknowledgeAlarmLog(id, acknowledgedBy)
}

func (s *AlarmService) DeleteAlarm(id int64) error {
	return database.DeleteAlarmLog(id)
}

func (s *AlarmService) BatchDeleteAlarms(ids []int64) (int64, error) {
	return database.DeleteAlarmLogsByIDs(ids)
}

func (s *AlarmService) ClearAlarms() (int64, error) {
	return database.ClearAlarmLogs()
}
