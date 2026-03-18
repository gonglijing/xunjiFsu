package collector

import (
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/database"
	"github.com/gonglijing/xunjiFsu/internal/models"
)

var stringStringMapPool = sync.Pool{
	New: func() any {
		return make(map[string]string)
	},
}

var stringFloatMapPool = sync.Pool{
	New: func() any {
		return make(map[string]float64)
	},
}

var stringSetPool = sync.Pool{
	New: func() any {
		return make(map[string]struct{})
	},
}

var pointValueMapPool = sync.Pool{
	New: func() any {
		return make(map[string]interface{})
	},
}

func acquireStringStringMap() map[string]string {
	return stringStringMapPool.Get().(map[string]string)
}

func releaseStringStringMap(m map[string]string) {
	if m == nil {
		return
	}
	for key := range m {
		delete(m, key)
	}
	stringStringMapPool.Put(m)
}

func acquireStringFloatMap() map[string]float64 {
	return stringFloatMapPool.Get().(map[string]float64)
}

func releaseStringFloatMap(m map[string]float64) {
	if m == nil {
		return
	}
	for key := range m {
		delete(m, key)
	}
	stringFloatMapPool.Put(m)
}

func acquireStringSet() map[string]struct{} {
	return stringSetPool.Get().(map[string]struct{})
}

func releaseStringSet(m map[string]struct{}) {
	if m == nil {
		return
	}
	for key := range m {
		delete(m, key)
	}
	stringSetPool.Put(m)
}

func acquirePointValueMap() map[string]interface{} {
	return pointValueMapPool.Get().(map[string]interface{})
}

func releasePointValueMap(m map[string]interface{}) {
	if m == nil {
		return
	}
	for key := range m {
		delete(m, key)
	}
	pointValueMapPool.Put(m)
}

type numericFieldLookup struct {
	raw        map[string]string
	points     []models.CollectPoint
	normalized map[string]string
	parsed     map[string]float64
	invalid    map[string]struct{}
	pointRaw   map[string]interface{}
	pointNorm  map[string]interface{}
}

func newNumericFieldLookup(fields map[string]string, points []models.CollectPoint) *numericFieldLookup {
	return &numericFieldLookup{
		raw:    fields,
		points: points,
	}
}

func (l *numericFieldLookup) release() {
	if l == nil {
		return
	}
	releaseStringStringMap(l.normalized)
	releaseStringFloatMap(l.parsed)
	releaseStringSet(l.invalid)
	releasePointValueMap(l.pointRaw)
	releasePointValueMap(l.pointNorm)
	l.normalized = nil
	l.parsed = nil
	l.invalid = nil
	l.pointRaw = nil
	l.pointNorm = nil
}

func normalizeFieldName(field string) string {
	return strings.ToLower(strings.TrimSpace(field))
}

func prepareLookupKeys(field, normalized string) (trimmed string, normalizedKey string, ok bool) {
	trimmed = strings.TrimSpace(field)
	if trimmed == "" {
		return "", "", false
	}
	if normalized == "" {
		normalized = strings.ToLower(trimmed)
	}
	return trimmed, normalized, normalized != ""
}

func (l *numericFieldLookup) getCachedFloat(primaryKey, fallbackKey string) (float64, bool) {
	if l == nil || l.parsed == nil {
		return 0, false
	}
	if primaryKey != "" {
		if value, exists := l.parsed[primaryKey]; exists {
			return value, true
		}
	}
	if fallbackKey != "" && fallbackKey != primaryKey {
		if value, exists := l.parsed[fallbackKey]; exists {
			return value, true
		}
	}
	return 0, false
}

func (l *numericFieldLookup) isKnownInvalid(primaryKey, fallbackKey string) bool {
	if l == nil || l.invalid == nil {
		return false
	}
	if primaryKey != "" {
		if _, exists := l.invalid[primaryKey]; exists {
			return true
		}
	}
	if fallbackKey != "" && fallbackKey != primaryKey {
		if _, exists := l.invalid[fallbackKey]; exists {
			return true
		}
	}
	return false
}

func (l *numericFieldLookup) cacheParsedFloat(key string, value float64) {
	if l == nil || key == "" {
		return
	}
	if l.parsed == nil {
		l.parsed = acquireStringFloatMap()
	}
	l.parsed[key] = value
}

func (l *numericFieldLookup) cacheInvalidFloat(key string) {
	if l == nil || key == "" {
		return
	}
	if l.invalid == nil {
		l.invalid = acquireStringSet()
	}
	l.invalid[key] = struct{}{}
}

func (l *numericFieldLookup) ensurePointRaw() {
	if l == nil || len(l.points) == 0 || l.pointRaw != nil {
		return
	}

	l.pointRaw = acquirePointValueMap()
	for _, point := range l.points {
		name := strings.TrimSpace(point.FieldName)
		if name == "" {
			continue
		}
		if _, exists := l.pointRaw[name]; !exists {
			l.pointRaw[name] = point.Value
		}
	}
}

func (l *numericFieldLookup) ensurePointNormalized() {
	if l == nil || len(l.points) == 0 || l.pointNorm != nil {
		return
	}

	l.ensurePointRaw()
	l.pointNorm = acquirePointValueMap()
	for name, value := range l.pointRaw {
		normalizedName := strings.ToLower(name)
		if normalizedName == "" {
			continue
		}
		if _, exists := l.pointNorm[normalizedName]; !exists {
			l.pointNorm[normalizedName] = value
		}
	}
}

func (l *numericFieldLookup) getRawValueWithNormalized(field, normalized string) (cacheKey string, value string, ok bool) {
	trimmed, normalizedKey, ok := prepareLookupKeys(field, normalized)
	if !ok {
		return "", "", false
	}

	if l != nil && len(l.raw) > 0 {
		if value, ok = l.raw[trimmed]; ok {
			return trimmed, value, true
		}
	}

	if l != nil && len(l.raw) > 0 {
		if l.normalized == nil {
			l.normalized = acquireStringStringMap()
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

		if value, ok = l.normalized[normalizedKey]; ok {
			return normalizedKey, value, true
		}
	}

	return l.getPointRawValue(trimmed, normalizedKey)
}

func (l *numericFieldLookup) getPointRawValue(field, normalized string) (cacheKey string, value string, ok bool) {
	if l == nil || len(l.points) == 0 {
		return "", "", false
	}

	l.ensurePointRaw()
	if raw, exists := l.pointRaw[field]; exists {
		return field, models.CollectPointValueString(raw), true
	}

	l.ensurePointNormalized()
	raw, exists := l.pointNorm[normalized]
	if !exists {
		return "", "", false
	}
	return normalized, models.CollectPointValueString(raw), true
}

func parseNumericPointValue(value interface{}) (float64, bool, bool) {
	switch v := value.(type) {
	case float64:
		return v, true, false
	case float32:
		return float64(v), true, false
	case int:
		return float64(v), true, false
	case int8:
		return float64(v), true, false
	case int16:
		return float64(v), true, false
	case int32:
		return float64(v), true, false
	case int64:
		return float64(v), true, false
	case uint:
		return float64(v), true, false
	case uint8:
		return float64(v), true, false
	case uint16:
		return float64(v), true, false
	case uint32:
		return float64(v), true, false
	case uint64:
		return float64(v), true, false
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
		return parsed, err == nil, true
	case []byte:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(string(v)), 64)
		return parsed, err == nil, true
	default:
		return 0, false, true
	}
}

func (l *numericFieldLookup) getFloatWithNormalized(field, normalized string) (float64, bool) {
	trimmed, normalizedKey, ok := prepareLookupKeys(field, normalized)
	if !ok {
		return 0, false
	}

	if value, cached := l.getCachedFloat(trimmed, normalizedKey); cached {
		return value, true
	}
	if l.isKnownInvalid(trimmed, normalizedKey) {
		return 0, false
	}

	pointInvalidKey := ""
	if l != nil && len(l.points) > 0 {
		l.ensurePointRaw()
		if raw, exists := l.pointRaw[trimmed]; exists {
			if value, parsed, cacheable := parseNumericPointValue(raw); parsed {
				if cacheable {
					l.cacheParsedFloat(trimmed, value)
				}
				return value, true
			}
			pointInvalidKey = trimmed
		}
		l.ensurePointNormalized()
		if raw, exists := l.pointNorm[normalizedKey]; exists {
			if value, parsed, cacheable := parseNumericPointValue(raw); parsed {
				if cacheable {
					l.cacheParsedFloat(normalizedKey, value)
				}
				return value, true
			}
			if pointInvalidKey == "" {
				pointInvalidKey = normalizedKey
			}
		}
	}

	cacheKey, valueStr, exists := l.getRawValueWithNormalized(trimmed, normalizedKey)
	if !exists {
		l.cacheInvalidFloat(pointInvalidKey)
		return 0, false
	}

	if value, cached := l.getCachedFloat(cacheKey, normalizedKey); cached {
		return value, true
	}
	if l.isKnownInvalid(cacheKey, normalizedKey) {
		return 0, false
	}

	value, err := strconv.ParseFloat(strings.TrimSpace(valueStr), 64)
	if err != nil {
		if cacheKey == "" {
			cacheKey = pointInvalidKey
		}
		l.cacheInvalidFloat(cacheKey)
		return 0, false
	}

	l.cacheParsedFloat(cacheKey, value)
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
	maybePruneAlarmStates(now, repeatInterval)
	lookup := newNumericFieldLookup(data.Fields, data.Points)
	defer lookup.release()
	for _, rule := range rules {
		threshold := rule.threshold
		if threshold == nil {
			continue
		}
		if rule.shielded {
			continue
		}

		value, ok := lookup.getFloatWithNormalized(rule.fieldName, rule.normalizedFieldName)
		if !ok {
			continue
		}

		matched := thresholdMatch(value, rule.operator, rule.thresholdValue)
		if !matched {
			continue
		}
		alarmKey := rule.alarmKey
		alarmKey.DeviceID = device.ID
		if !shouldEmitAlarmForKey(alarmKey, now, repeatInterval) {
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
