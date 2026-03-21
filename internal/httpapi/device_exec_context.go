package httpapi

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gonglijing/xunjiFsu/internal/driver"
	"github.com/gonglijing/xunjiFsu/internal/models"
)

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
