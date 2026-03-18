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

	result.ProductKey = firstNonEmptyTrimmed(result.ProductKey, result.ProductKeyAlt)
	if result.ProductKey == "" {
		result.ProductKey = resolveDriverProductKeyFromResultData(result.Data)
	}
	if result.ProductKey == "" && len(rawOutput) > 0 {
		result.ProductKey = parseDriverProductKeyFromResultOutput(rawOutput)
	}

	result.ProductKeyAlt = ""
	removeDriverProductKeyFields(result.Data)
}

func resolveDriverProductKeyFromResultData(data map[string]string) string {
	if len(data) == 0 {
		return ""
	}
	return firstNonEmptyTrimmed(data["productKey"], data["product_key"])
}

func parseDriverProductKeyFromResultOutput(rawOutput []byte) string {
	var probe driverIdentityProbe
	if err := json.Unmarshal(rawOutput, &probe); err != nil {
		return ""
	}
	if value := firstNonEmptyTrimmed(probe.ProductKey, probe.ProductKeyAlt); value != "" {
		return value
	}
	if probe.Data == nil {
		return ""
	}
	return firstNonEmptyTrimmed(probe.Data.ProductKey, probe.Data.ProductKeyAlt)
}

func removeDriverProductKeyFields(data map[string]string) {
	if data == nil {
		return
	}
	delete(data, "productKey")
	delete(data, "product_key")
}

func firstNonEmptyTrimmed(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func isDriverIdentityField(field string) bool {
	switch strings.ToLower(strings.TrimSpace(field)) {
	case "productkey", "product_key", "devicekey", "device_key":
		return true
	default:
		return false
	}
}
