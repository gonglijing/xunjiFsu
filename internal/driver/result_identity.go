package driver

import (
	"encoding/json"
	"strings"
)

func normalizeDriverResultIdentity(result *DriverResult, rawOutput []byte) {
	if result == nil {
		return
	}

	result.ProductKey = strings.TrimSpace(result.ProductKey)
	if result.ProductKey == "" {
		result.ProductKey = pickProductKeyFromMap(result.Data)
	}
	if result.ProductKey == "" && len(rawOutput) > 0 {
		result.ProductKey = pickProductKeyFromRawOutput(rawOutput)
	}

	if result.Data != nil {
		delete(result.Data, "productKey")
		delete(result.Data, "product_key")
	}
}

func pickProductKeyFromMap(data map[string]string) string {
	if len(data) == 0 {
		return ""
	}
	if value := strings.TrimSpace(data["productKey"]); value != "" {
		return value
	}
	if value := strings.TrimSpace(data["product_key"]); value != "" {
		return value
	}
	return ""
}

func pickProductKeyFromRawOutput(rawOutput []byte) string {
	obj := make(map[string]interface{})
	if err := json.Unmarshal(rawOutput, &obj); err != nil {
		return ""
	}
	return pickProductKeyFromAnyMap(obj)
}

func pickProductKeyFromAnyMap(data map[string]interface{}) string {
	if len(data) == 0 {
		return ""
	}
	if value := trimmedAnyString(data["productKey"]); value != "" {
		return value
	}
	if value := trimmedAnyString(data["product_key"]); value != "" {
		return value
	}
	if rawData, ok := data["data"].(map[string]interface{}); ok {
		if value := pickProductKeyFromAnyMap(rawData); value != "" {
			return value
		}
	}
	return ""
}

func trimmedAnyString(value interface{}) string {
	if value == nil {
		return ""
	}
	str, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(str)
}

func isDriverIdentityField(field string) bool {
	switch strings.ToLower(strings.TrimSpace(field)) {
	case "productkey", "product_key", "devicekey", "device_key":
		return true
	default:
		return false
	}
}
