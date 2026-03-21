package httpapi

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	"github.com/gonglijing/xunjiFsu/internal/service"
)

func normalizeWriteParams(config map[string]string, params map[string]any) error {
	if config == nil {
		return fmt.Errorf("write params are empty")
	}
	if err := validateSingleWriteRequest(config, params); err != nil {
		return err
	}
	fieldName := strings.TrimSpace(firstNonEmpty(config["field_name"], config["fieldName"], config["field"]))
	value := strings.TrimSpace(firstNonEmpty(config["value"], config["val"]))
	if fieldName != "" && value == "" {
		if raw, ok := params[fieldName]; ok {
			value = stringifyParamValue(raw)
		}
	}
	if fieldName == "" || value == "" {
		candidateField, candidateValue, err := resolveSingleWriteCandidate(params)
		if err != nil {
			return err
		}
		if fieldName == "" {
			fieldName = candidateField
		}
		if value == "" {
			value = candidateValue
		}
	}
	fieldName = strings.TrimSpace(fieldName)
	value = strings.TrimSpace(value)
	if fieldName == "" {
		return fmt.Errorf("write params missing field_name")
	}
	if value == "" {
		return fmt.Errorf("write params missing value")
	}
	config["field_name"] = fieldName
	config["value"] = value
	delete(config, "field")
	delete(config, "fieldName")
	delete(config, "val")
	return nil
}

func validateSingleWriteRequest(config map[string]string, params map[string]any) error {
	if len(params) == 0 {
		return nil
	}
	explicitField := strings.TrimSpace(firstNonEmpty(config["field_name"], config["fieldName"], config["field"]))
	candidateFields, err := collectWriteCandidateFields(params)
	if err != nil {
		return err
	}
	if len(candidateFields) > 1 {
		return fmt.Errorf("write only supports one field per call")
	}
	if explicitField != "" && len(candidateFields) == 1 && !strings.EqualFold(explicitField, candidateFields[0]) {
		return fmt.Errorf("write params field_name %q does not match payload field %q", explicitField, candidateFields[0])
	}
	if raw, ok := firstPresentValue(params, "value", "val"); ok {
		switch raw.(type) {
		case map[string]any, []any:
			return fmt.Errorf("write params value must be a scalar")
		}
	}
	return nil
}

func collectWriteCandidateFields(params map[string]any) ([]string, error) {
	if len(params) == 0 {
		return nil, nil
	}
	fields := make(map[string]string)
	addWriteCandidateFields(fields, params)
	if properties, ok := resolveMapValue(params["properties"]); ok {
		addWriteCandidateFields(fields, properties)
	}
	for _, key := range []string{"sub_device", "subDevice"} {
		sub, ok := resolveMapValue(params[key])
		if !ok {
			continue
		}
		if properties, ok := resolveMapValue(sub["properties"]); ok {
			addWriteCandidateFields(fields, properties)
		}
	}
	for _, key := range []string{"sub_devices", "subDevices"} {
		raw, exists := params[key]
		if !exists {
			continue
		}
		list, ok := raw.([]any)
		if !ok {
			return nil, fmt.Errorf("write params %s must be an array", key)
		}
		if len(list) > 1 {
			return nil, fmt.Errorf("write only supports one sub device per call")
		}
		if len(list) == 0 {
			continue
		}
		item, ok := resolveMapValue(list[0])
		if !ok {
			return nil, fmt.Errorf("write params %s[0] must be an object", key)
		}
		if properties, ok := resolveMapValue(item["properties"]); ok {
			addWriteCandidateFields(fields, properties)
		}
	}
	names := make([]string, 0, len(fields))
	for _, field := range fields {
		names = append(names, field)
	}
	slices.Sort(names)
	return names, nil
}

func addWriteCandidateFields(dst map[string]string, values map[string]any) {
	if len(values) == 0 {
		return
	}
	for key, raw := range values {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" || isReservedWriteKey(trimmedKey) {
			continue
		}
		switch raw.(type) {
		case map[string]any, []any:
			continue
		}
		normalized := strings.ToLower(trimmedKey)
		if _, exists := dst[normalized]; !exists {
			dst[normalized] = trimmedKey
		}
	}
}

func firstPresentValue(values map[string]any, keys ...string) (any, bool) {
	if len(values) == 0 {
		return nil, false
	}
	for _, key := range keys {
		if raw, ok := values[key]; ok {
			return raw, true
		}
	}
	return nil, false
}

func resolveSingleWriteCandidate(params map[string]any) (field string, value string, err error) {
	if len(params) == 0 {
		return "", "", nil
	}
	if props, ok := resolveWriteProperties(params); ok {
		return pickSingleWriteValue(props)
	}
	return pickSingleWriteValue(params)
}

func resolveWriteProperties(params map[string]any) (map[string]any, bool) {
	if properties, ok := resolveMapValue(params["properties"]); ok {
		return properties, true
	}
	for _, key := range []string{"sub_device", "subDevice"} {
		sub, ok := resolveMapValue(params[key])
		if !ok {
			continue
		}
		if properties, ok := resolveMapValue(sub["properties"]); ok {
			return properties, true
		}
	}
	for _, key := range []string{"sub_devices", "subDevices"} {
		list, ok := params[key].([]any)
		if !ok || len(list) != 1 {
			continue
		}
		item, ok := resolveMapValue(list[0])
		if !ok {
			continue
		}
		if properties, ok := resolveMapValue(item["properties"]); ok {
			return properties, true
		}
	}
	return nil, false
}

func resolveMapValue(value any) (map[string]any, bool) {
	if value == nil {
		return nil, false
	}
	out, ok := value.(map[string]any)
	return out, ok
}

func pickSingleWriteValue(values map[string]any) (field string, value string, err error) {
	type candidate struct {
		key   string
		value string
	}
	candidates := make([]candidate, 0, len(values))
	for key, raw := range values {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" || isReservedWriteKey(trimmedKey) {
			continue
		}
		switch raw.(type) {
		case map[string]any, []any:
			continue
		}
		candidates = append(candidates, candidate{key: trimmedKey, value: stringifyParamValue(raw)})
	}
	if len(candidates) == 0 {
		return "", "", nil
	}
	slices.SortFunc(candidates, func(a, b candidate) int { return cmp.Compare(a.key, b.key) })
	if len(candidates) > 1 {
		fields := make([]string, 0, len(candidates))
		for _, item := range candidates {
			fields = append(fields, item.key)
		}
		return "", "", fmt.Errorf("write params are ambiguous, please provide field_name and value explicitly (fields: %s)", strings.Join(fields, ","))
	}
	return candidates[0].key, candidates[0].value, nil
}

func isReservedWriteKey(key string) bool {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "field_name", "fieldname", "field", "value", "val", "values",
		"product_key", "productkey", "device_key", "devicekey",
		"identity", "properties", "sub_device", "subdevice", "sub_devices", "subdevices":
		return true
	default:
		return false
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func parseDriverWritables(configSchema string) ([]any, error) {
	return service.ParseDriverWritables(configSchema)
}
