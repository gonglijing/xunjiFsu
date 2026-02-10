package handlers

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/gonglijing/xunjiFsu/internal/database"
	"github.com/gonglijing/xunjiFsu/internal/driver"
	"github.com/gonglijing/xunjiFsu/internal/models"
)

func normalizeDeviceInput(device *models.Device) error {
	if device == nil {
		return sql.ErrNoRows
	}
	device.Name = strings.TrimSpace(device.Name)
	if device.Name == "" {
		return sql.ErrNoRows
	}
	if device.StorageInterval <= 0 {
		device.StorageInterval = database.DefaultStorageIntervalSeconds
	}
	if device.Enabled != 1 {
		device.Enabled = 0
	}
	return nil
}

func resolveDriverDisplayName(driverModel *models.Driver) string {
	if driverModel == nil {
		return ""
	}
	if driverModel.FilePath != "" {
		name := filepath.Base(driverModel.FilePath)
		return strings.TrimSuffix(name, ".wasm")
	}
	return strings.TrimSpace(driverModel.Name)
}

func buildDriverNameMap(drivers []*models.Driver) map[int64]string {
	nameMap := make(map[int64]string, len(drivers))
	for _, driverModel := range drivers {
		if driverModel == nil {
			continue
		}
		if name := resolveDriverDisplayName(driverModel); name != "" {
			nameMap[driverModel.ID] = name
		}
	}
	return nameMap
}

func stringifyParamValue(value interface{}) string {
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

func normalizeExecuteFunction(function string) (requestFunc string, pluginFunc string, configFunc string) {
	f := strings.TrimSpace(function)
	if f == "" || strings.EqualFold(f, "collect") {
		return "collect", "handle", "read"
	}
	return f, f, f
}

func inferDeviceResourceType(device *models.Device) string {
	if device == nil {
		return "serial"
	}
	if t := strings.TrimSpace(device.ResourceType); t != "" {
		return strings.ToLower(t)
	}
	driverType := strings.ToLower(strings.TrimSpace(device.DriverType))
	switch {
	case strings.Contains(driverType, "tcp"), strings.Contains(driverType, "udp"), strings.Contains(driverType, "net"):
		return "net"
	case strings.Contains(driverType, "serial"), strings.Contains(driverType, "rtu"):
		return "serial"
	default:
		if strings.TrimSpace(device.IPAddress) != "" || device.PortNum > 0 {
			return "net"
		}
		return "serial"
	}
}

func enrichExecuteIdentity(config map[string]string, device *models.Device) {
	if config == nil || device == nil {
		return
	}

	productKey := strings.TrimSpace(device.ProductKey)
	deviceKey := strings.TrimSpace(device.DeviceKey)

	if productKey != "" {
		config["product_key"] = productKey
		config["productKey"] = productKey
	}
	if deviceKey != "" {
		config["device_key"] = deviceKey
		config["deviceKey"] = deviceKey
	}
}

func normalizeWriteParams(config map[string]string, params map[string]interface{}) error {
	if config == nil {
		return fmt.Errorf("write params are empty")
	}

	fieldName := firstNonEmpty(
		config["field_name"],
		config["fieldName"],
		config["field"],
	)
	fieldName = strings.TrimSpace(fieldName)

	value := strings.TrimSpace(firstNonEmpty(
		config["value"],
		config["val"],
	))

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

func resolveSingleWriteCandidate(params map[string]interface{}) (field string, value string, err error) {
	if len(params) == 0 {
		return "", "", nil
	}

	if props, ok := extractWriteProperties(params); ok {
		return pickSingleWriteValue(props)
	}

	return pickSingleWriteValue(params)
}

func extractWriteProperties(params map[string]interface{}) (map[string]interface{}, bool) {
	if properties, ok := mapFromAny(params["properties"]); ok {
		return properties, true
	}

	for _, key := range []string{"sub_device", "subDevice"} {
		sub, ok := mapFromAny(params[key])
		if !ok {
			continue
		}
		if properties, ok := mapFromAny(sub["properties"]); ok {
			return properties, true
		}
	}

	for _, key := range []string{"sub_devices", "subDevices"} {
		list, ok := params[key].([]interface{})
		if !ok || len(list) != 1 {
			continue
		}
		item, ok := mapFromAny(list[0])
		if !ok {
			continue
		}
		if properties, ok := mapFromAny(item["properties"]); ok {
			return properties, true
		}
	}

	return nil, false
}

func mapFromAny(value interface{}) (map[string]interface{}, bool) {
	if value == nil {
		return nil, false
	}
	if out, ok := value.(map[string]interface{}); ok {
		return out, true
	}
	return nil, false
}

func pickSingleWriteValue(values map[string]interface{}) (field string, value string, err error) {
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
		case map[string]interface{}, []interface{}:
			continue
		}
		candidates = append(candidates, candidate{key: trimmedKey, value: stringifyParamValue(raw)})
	}

	if len(candidates) == 0 {
		return "", "", nil
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].key < candidates[j].key
	})

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

func buildExecuteDriverConfig(params map[string]interface{}, device *models.Device, configFunc string) (map[string]string, error) {
	config := make(map[string]string, len(params)+2)
	for key, value := range params {
		config[key] = stringifyParamValue(value)
	}
	if device != nil {
		config["device_address"] = device.DeviceAddress
	}
	config["func_name"] = configFunc
	enrichExecuteIdentity(config, device)

	if configFunc == "write" {
		if err := normalizeWriteParams(config, params); err != nil {
			return nil, err
		}
	}

	return config, nil
}

func buildExecuteDriverContext(device *models.Device, config map[string]string) *driver.DriverContext {
	resourceID := int64(0)
	if device != nil && device.ResourceID != nil {
		resourceID = *device.ResourceID
	}

	ctx := &driver.DriverContext{
		Config:       config,
		DeviceConfig: "",
		ResourceID:   resourceID,
		ResourceType: inferDeviceResourceType(device),
	}
	if device != nil {
		ctx.DeviceID = device.ID
		ctx.DeviceName = device.Name
	}

	return ctx
}
