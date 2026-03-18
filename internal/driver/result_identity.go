package driver

import (
	"encoding/json"
	"strings"
	"unicode/utf8"
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
	return isNormalizedDriverIdentityField(trimDriverFieldName(field))
}

func isNormalizedDriverIdentityField(field string) bool {
	switch len(field) {
	case 9:
		return strings.EqualFold(field, "devicekey")
	case 10:
		return strings.EqualFold(field, "productkey") || strings.EqualFold(field, "device_key")
	case 11:
		return strings.EqualFold(field, "product_key")
	default:
		return false
	}
}

func trimDriverFieldName(s string) string {
	if s == "" {
		return ""
	}
	start := 0
	end := len(s)
	for start < end && isASCIIDriverSpace(s[start]) {
		start++
	}
	if start == end {
		return ""
	}
	for end > start && isASCIIDriverSpace(s[end-1]) {
		end--
	}
	if start == 0 && end == len(s) {
		if s[0] < utf8.RuneSelf && s[len(s)-1] < utf8.RuneSelf {
			return s
		}
		return strings.TrimSpace(s)
	}
	if s[start] < utf8.RuneSelf && s[end-1] < utf8.RuneSelf {
		return s[start:end]
	}
	return strings.TrimSpace(s)
}

func isASCIIDriverSpace(c byte) bool {
	switch c {
	case ' ', '\t', '\n', '\r', '\f', '\v':
		return true
	default:
		return false
	}
}
