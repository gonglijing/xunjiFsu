package adapters

import (
	"fmt"
	"strconv"
	"strings"
)

func pickConfigString(data map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		value, ok := data[key]
		if !ok || value == nil {
			continue
		}
		if text, ok := value.(string); ok {
			trimmed := strings.TrimSpace(text)
			if trimmed != "" {
				return trimmed
			}
			continue
		}
		text := strings.TrimSpace(fmt.Sprintf("%v", value))
		if text != "" {
			return text
		}
	}
	return ""
}

func pickConfigInt(data map[string]interface{}, fallback int, keys ...string) int {
	for _, key := range keys {
		value, ok := data[key]
		if !ok || value == nil {
			continue
		}
		switch typed := value.(type) {
		case int:
			return typed
		case int8:
			return int(typed)
		case int16:
			return int(typed)
		case int32:
			return int(typed)
		case int64:
			return int(typed)
		case float32:
			return int(typed)
		case float64:
			return int(typed)
		case string:
			trimmed := strings.TrimSpace(typed)
			if trimmed == "" {
				continue
			}
			number, err := strconv.Atoi(trimmed)
			if err == nil {
				return number
			}
		}
	}
	return fallback
}

func pickConfigBool(data map[string]interface{}, fallback bool, keys ...string) bool {
	for _, key := range keys {
		value, ok := data[key]
		if !ok || value == nil {
			continue
		}
		switch typed := value.(type) {
		case bool:
			return typed
		case int:
			return typed != 0
		case int64:
			return typed != 0
		case float64:
			return typed != 0
		case string:
			trimmed := strings.TrimSpace(strings.ToLower(typed))
			if trimmed == "" {
				continue
			}
			if trimmed == "true" || trimmed == "1" || trimmed == "yes" {
				return true
			}
			if trimmed == "false" || trimmed == "0" || trimmed == "no" {
				return false
			}
		}
	}
	return fallback
}
