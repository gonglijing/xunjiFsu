package driver

import (
	"encoding/json"
	"strings"
)

type driverIdentityProbe struct {
	ProductKeyAlt string                   `json:"product_key"`
	ProductKey    string                   `json:"productKey"`
	Data          *driverIdentityProbeData `json:"data"`
}

type driverIdentityProbeData struct {
	ProductKeyAlt string `json:"product_key"`
	ProductKey    string `json:"productKey"`
}

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
	var probe driverIdentityProbe
	if err := json.Unmarshal(rawOutput, &probe); err != nil {
		return ""
	}
	if value := strings.TrimSpace(probe.ProductKey); value != "" {
		return value
	}
	if value := strings.TrimSpace(probe.ProductKeyAlt); value != "" {
		return value
	}
	if probe.Data == nil {
		return ""
	}
	if value := strings.TrimSpace(probe.Data.ProductKey); value != "" {
		return value
	}
	return strings.TrimSpace(probe.Data.ProductKeyAlt)
}

func isDriverIdentityField(field string) bool {
	switch strings.ToLower(strings.TrimSpace(field)) {
	case "productkey", "product_key", "devicekey", "device_key":
		return true
	default:
		return false
	}
}
