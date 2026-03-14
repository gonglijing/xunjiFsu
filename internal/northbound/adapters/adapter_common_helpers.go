package adapters

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type jsonFieldValueMap map[string]string
type jsonConvertedValue string

type jsonSingleConvertedField struct {
	Key   string
	Value string
}

type jsonSingleRawField struct {
	Key   string
	Value string
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

func (m jsonFieldValueMap) MarshalJSON() ([]byte, error) {
	if len(m) == 0 {
		return []byte("{}"), nil
	}

	buf := make([]byte, 0, len(m)*24)
	buf = append(buf, '{')
	index := 0
	for key, value := range m {
		if index > 0 {
			buf = append(buf, ',')
		}
		buf = strconv.AppendQuote(buf, key)
		buf = append(buf, ':')
		buf = appendConvertedFieldJSON(buf, value)
		index++
	}
	buf = append(buf, '}')
	return buf, nil
}

func appendConvertedFieldJSON(dst []byte, value string) []byte {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return strconv.AppendQuote(dst, "")
	}
	if b, err := strconv.ParseBool(trimmed); err == nil {
		if b {
			return append(dst, "true"...)
		}
		return append(dst, "false"...)
	}
	if i, err := strconv.ParseInt(trimmed, 10, 64); err == nil {
		return strconv.AppendInt(dst, i, 10)
	}
	if f, err := strconv.ParseFloat(trimmed, 64); err == nil {
		return strconv.AppendFloat(dst, f, 'f', -1, 64)
	}
	return strconv.AppendQuote(dst, value)
}

func (v jsonConvertedValue) MarshalJSON() ([]byte, error) {
	return appendConvertedFieldJSON(nil, string(v)), nil
}

func (f jsonSingleConvertedField) MarshalJSON() ([]byte, error) {
	buf := make([]byte, 0, len(f.Key)+len(f.Value)+8)
	buf = append(buf, '{')
	buf = strconv.AppendQuote(buf, f.Key)
	buf = append(buf, ':')
	buf = appendConvertedFieldJSON(buf, f.Value)
	buf = append(buf, '}')
	return buf, nil
}

func (f jsonSingleRawField) MarshalJSON() ([]byte, error) {
	buf := make([]byte, 0, len(f.Key)+len(f.Value)+8)
	buf = append(buf, '{')
	buf = strconv.AppendQuote(buf, f.Key)
	buf = append(buf, ':')
	buf = strconv.AppendQuote(buf, f.Value)
	buf = append(buf, '}')
	return buf, nil
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
