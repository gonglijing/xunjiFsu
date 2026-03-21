package service

import (
	"fmt"
	"strings"

	"github.com/gonglijing/xunjiFsu/internal/models"
)

func ValidateAlarmRepeatIntervalSeconds(seconds int) error {
	if seconds <= 0 {
		return fmt.Errorf("alarm repeat interval must be > 0")
	}
	return nil
}

func NormalizeThresholdInput(threshold *models.Threshold) {
	if threshold == nil {
		return
	}

	threshold.FieldName = strings.TrimSpace(threshold.FieldName)
	threshold.Operator = strings.TrimSpace(threshold.Operator)
	threshold.Severity = strings.TrimSpace(threshold.Severity)
	threshold.Message = strings.TrimSpace(threshold.Message)
	if threshold.Shielded != 1 {
		threshold.Shielded = 0
	}
}
