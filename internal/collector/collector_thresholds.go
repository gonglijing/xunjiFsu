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
	lookup := newNumericFieldLookup(fields, nil)
	return lookup.getFloat(field)
}

type numericFieldLookup struct {
	raw        map[string]string
	points     []models.CollectPoint
	normalized map[string]string
	parsed     map[string]float64
	pointRaw   map[string]interface{}
	pointNorm  map[string]interface{}
}

func newNumericFieldLookup(fields map[string]string, points []models.CollectPoint) *numericFieldLookup {
	return &numericFieldLookup{
		raw:    fields,
		points: points,
	}
}

func normalizeFieldName(field string) string {
	return strings.ToLower(strings.TrimSpace(field))
}

func (l *numericFieldLookup) getRawValue(field string) (normalized string, value string, ok bool) {
	return l.getRawValueWithNormalized(field, "")
}

func (l *numericFieldLookup) getRawValueWithNormalized(field, normalized string) (cacheKey string, value string, ok bool) {
	trimmed := strings.TrimSpace(field)
	if trimmed == "" {
		return "", "", false
	}

	if l != nil && len(l.raw) > 0 {
		if value, ok = l.raw[trimmed]; ok {
			return trimmed, value, true
		}
	}

	if normalized == "" {
		normalized = strings.ToLower(trimmed)
	}
	if normalized == "" {
		return "", "", false
	}

	if l != nil && len(l.raw) > 0 {
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

		if value, ok = l.normalized[normalized]; ok {
			return normalized, value, true
		}
	}

	return l.getPointRawValue(trimmed, normalized)
}

func (l *numericFieldLookup) getPointRawValue(field, normalized string) (cacheKey string, value string, ok bool) {
	if l == nil || len(l.points) == 0 {
		return "", "", false
	}

	if l.pointRaw == nil {
		l.pointRaw = make(map[string]interface{}, len(l.points))
		l.pointNorm = make(map[string]interface{}, len(l.points))
		for _, point := range l.points {
			name := strings.TrimSpace(point.FieldName)
			if name == "" {
				continue
			}
			if _, exists := l.pointRaw[name]; !exists {
				l.pointRaw[name] = point.Value
			}
			normalizedName := normalizeFieldName(name)
			if normalizedName == "" {
				continue
			}
			if _, exists := l.pointNorm[normalizedName]; !exists {
				l.pointNorm[normalizedName] = point.Value
			}
		}
	}

	if raw, exists := l.pointRaw[field]; exists {
		return field, models.CollectPointValueString(raw), true
	}
	raw, exists := l.pointNorm[normalized]
	if !exists {
		return "", "", false
	}
	return normalized, models.CollectPointValueString(raw), true
}

func parseNumericPointValue(value interface{}) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int8:
		return float64(v), true
	case int16:
		return float64(v), true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	case uint:
		return float64(v), true
	case uint8:
		return float64(v), true
	case uint16:
		return float64(v), true
	case uint32:
		return float64(v), true
	case uint64:
		return float64(v), true
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
		return parsed, err == nil
	case []byte:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(string(v)), 64)
		return parsed, err == nil
	default:
		return 0, false
	}
}

func (l *numericFieldLookup) getFloat(field string) (float64, bool) {
	return l.getFloatWithNormalized(field, "")
}

func (l *numericFieldLookup) getFloatWithNormalized(field, normalized string) (float64, bool) {
	if l != nil && len(l.points) > 0 {
		if l.pointRaw == nil {
			_, _, _ = l.getPointRawValue(strings.TrimSpace(field), normalized)
		}
		trimmed := strings.TrimSpace(field)
		if trimmed != "" {
			if raw, ok := l.pointRaw[trimmed]; ok {
				if value, parsed := parseNumericPointValue(raw); parsed {
					return value, true
				}
			}
		}
		if normalized == "" {
			normalized = strings.ToLower(trimmed)
		}
		if normalized != "" {
			if raw, ok := l.pointNorm[normalized]; ok {
				if value, parsed := parseNumericPointValue(raw); parsed {
					return value, true
				}
			}
		}
	}

	cacheKey, valueStr, ok := l.getRawValueWithNormalized(field, normalized)
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

	rules, err := getDeviceThresholdRules(device.ID)
	if err != nil {
		return err
	}

	now := time.Now()
	repeatInterval := resolveAlarmRepeatInterval()
	lookup := newNumericFieldLookup(data.Fields, data.Points)
	for _, rule := range rules {
		threshold := rule.threshold
		if threshold == nil {
			continue
		}
		if threshold.Shielded == 1 {
			_ = shouldEmitAlarm(device.ID, threshold, false, now, repeatInterval)
			continue
		}

		value, ok := lookup.getFloatWithNormalized(rule.fieldName, rule.normalizedFieldName)
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
