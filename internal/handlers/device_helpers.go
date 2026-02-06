package handlers

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gonglijing/xunjiFsu/internal/database"
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
