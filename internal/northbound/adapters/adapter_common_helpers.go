package adapters

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/models"
)

func cloneCollectData(data *models.CollectData) *models.CollectData {
	if data == nil {
		return nil
	}
	out := *data
	if len(data.Fields) > 0 {
		out.Fields = make(map[string]string, len(data.Fields))
		for key, value := range data.Fields {
			out.Fields[key] = value
		}
	} else {
		out.Fields = nil
	}
	return &out
}

func cloneAlarmPayload(alarm *models.AlarmPayload) *models.AlarmPayload {
	if alarm == nil {
		return nil
	}
	out := *alarm
	return &out
}

func mapFromAnyByKey2(values map[string]interface{}, key1, key2 string) (map[string]interface{}, bool) {
	if out, ok := mapFromAny(values[key1]); ok {
		return out, true
	}
	return mapFromAny(values[key2])
}

func interfaceSliceByKey2(values map[string]interface{}, key1, key2 string) ([]interface{}, bool) {
	if list, ok := values[key1].([]interface{}); ok {
		return list, true
	}
	list, ok := values[key2].([]interface{})
	return list, ok
}

func mapFromAny(value interface{}) (map[string]interface{}, bool) {
	out, ok := value.(map[string]interface{})
	if !ok || out == nil {
		return nil, false
	}
	return out, true
}

func stringifyAny(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	case bool:
		return strconv.FormatBool(v)
	case int:
		return strconv.Itoa(v)
	case int8:
		return strconv.FormatInt(int64(v), 10)
	case int16:
		return strconv.FormatInt(int64(v), 10)
	case int32:
		return strconv.FormatInt(int64(v), 10)
	case int64:
		return strconv.FormatInt(v, 10)
	case uint:
		return strconv.FormatUint(uint64(v), 10)
	case uint8:
		return strconv.FormatUint(uint64(v), 10)
	case uint16:
		return strconv.FormatUint(uint64(v), 10)
	case uint32:
		return strconv.FormatUint(uint64(v), 10)
	case uint64:
		return strconv.FormatUint(v, 10)
	case float32:
		return strconv.FormatFloat(float64(v), 'f', -1, 32)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	default:
		return fmt.Sprintf("%v", value)
	}
}

func convertFieldValue(value string) interface{} {
	v := strings.TrimSpace(value)
	if v == "" {
		return ""
	}
	if b, err := strconv.ParseBool(v); err == nil {
		return b
	}
	if i, err := strconv.ParseInt(v, 10, 64); err == nil {
		return i
	}
	if f, err := strconv.ParseFloat(v, 64); err == nil {
		return f
	}
	return value
}

func pickFirstNonEmpty(values ...string) string {
	for _, value := range values {
		v := strings.TrimSpace(value)
		if v != "" {
			return v
		}
	}
	return ""
}

func pickFirstNonEmpty2(left, right string) string {
	if v := strings.TrimSpace(left); v != "" {
		return v
	}
	return strings.TrimSpace(right)
}

func pickFirstNonEmpty3(first, second, third string) string {
	if v := pickFirstNonEmpty2(first, second); v != "" {
		return v
	}
	return strings.TrimSpace(third)
}

func resolveInterval(ms int, fallback time.Duration) time.Duration {
	if ms <= 0 {
		return fallback
	}
	interval := time.Duration(ms) * time.Millisecond
	if interval < 200*time.Millisecond {
		return 200 * time.Millisecond
	}
	return interval
}

func resolvePositive(v int, fallback int) int {
	if v <= 0 {
		return fallback
	}
	return v
}

func normalizeBroker(broker string) string {
	broker = strings.TrimSpace(broker)
	if broker == "" {
		return ""
	}
	if strings.Contains(broker, "://") {
		return broker
	}
	return "tcp://" + broker
}

func clampQOS(qos int) byte {
	if qos < 0 {
		return 0
	}
	if qos > 2 {
		return 2
	}
	return byte(qos)
}
