package collector

import (
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/database"
	"github.com/gonglijing/xunjiFsu/internal/models"
)

func parseFloatFieldValue(fields map[string]string, field string) (float64, bool) {
	lookup := newNumericFieldLookup(fields)
	return lookup.getFloat(field)
}

type numericFieldLookup struct {
	raw        map[string]string
	normalized map[string]string
	parsed     map[string]float64
}

func newNumericFieldLookup(fields map[string]string) *numericFieldLookup {
	return &numericFieldLookup{
		raw: fields,
	}
}

func normalizeFieldName(field string) string {
	return strings.ToLower(strings.TrimSpace(field))
}

func (l *numericFieldLookup) getRawValue(field string) (normalized string, value string, ok bool) {
	if l == nil || len(l.raw) == 0 {
		return "", "", false
	}

	trimmed := strings.TrimSpace(field)
	if trimmed == "" {
		return "", "", false
	}

	if value, ok = l.raw[trimmed]; ok {
		return trimmed, value, true
	}

	normalized = strings.ToLower(trimmed)
	if normalized == "" {
		return "", "", false
	}

	if l.normalized == nil {
		l.normalized = make(map[string]string, len(l.raw))
		for key, candidate := range l.raw {
			normalizedKey := normalizeFieldName(key)
			if normalizedKey == "" {
				continue
			}
			if _, exists := l.normalized[normalizedKey]; exists {
				continue
			}
			l.normalized[normalizedKey] = candidate
		}
	}

	value, ok = l.normalized[normalized]
	return normalized, value, ok
}

func (l *numericFieldLookup) getFloat(field string) (float64, bool) {
	cacheKey, valueStr, ok := l.getRawValue(field)
	if !ok {
		return 0, false
	}

	if l.parsed != nil {
		if cached, exists := l.parsed[cacheKey]; exists {
			return cached, true
		}
	}

	value, err := strconv.ParseFloat(strings.TrimSpace(valueStr), 64)
	if err != nil {
		return 0, false
	}

	if l.parsed == nil {
		l.parsed = make(map[string]float64, 8)
	}
	l.parsed[cacheKey] = value
	return value, true
}

// checkThresholds 检查阈值
func (c *Collector) checkThresholds(device *models.Device, data *models.CollectData) error {
	if device == nil || data == nil {
		return nil
	}

	thresholds, err := GetDeviceThresholds(device.ID)
	if err != nil {
		return err
	}

	now := time.Now()
	repeatInterval := resolveAlarmRepeatInterval()
	lookup := newNumericFieldLookup(data.Fields)
	for _, threshold := range thresholds {
		if threshold == nil {
			continue
		}
		if threshold.Shielded == 1 {
			_ = shouldEmitAlarm(device.ID, threshold, false, now, repeatInterval)
			continue
		}

		value, ok := lookup.getFloat(threshold.FieldName)
		if !ok {
			_ = shouldEmitAlarm(device.ID, threshold, false, now, repeatInterval)
			continue
		}

		matched := thresholdMatch(value, threshold.Operator, threshold.Value)
		if !shouldEmitAlarm(device.ID, threshold, matched, now, repeatInterval) {
			continue
		}

		c.handleAlarm(device, threshold, value)
	}

	return nil
}

// handleAlarm 处理报警
func (c *Collector) handleAlarm(device *models.Device, threshold *models.Threshold, actualValue float64) {
	logEntry := &models.AlarmLog{
		DeviceID:       device.ID,
		ThresholdID:    &threshold.ID,
		FieldName:      threshold.FieldName,
		ActualValue:    actualValue,
		ThresholdValue: threshold.Value,
		Operator:       threshold.Operator,
		Severity:       threshold.Severity,
		Message:        threshold.Message,
	}

	if _, err := database.CreateAlarmLog(logEntry); err != nil {
		log.Printf("Failed to create alarm log: %v", err)
	}

	if c.northboundMgr == nil {
		return
	}

	payload := &models.AlarmPayload{
		DeviceID:    device.ID,
		DeviceName:  device.Name,
		ProductKey:  device.ProductKey,
		DeviceKey:   device.DeviceKey,
		FieldName:   threshold.FieldName,
		ActualValue: actualValue,
		Threshold:   threshold.Value,
		Operator:    threshold.Operator,
		Severity:    threshold.Severity,
		Message:     threshold.Message,
	}

	c.northboundMgr.SendAlarm(payload)
}
