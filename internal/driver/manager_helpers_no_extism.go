//go:build no_extism

package driver

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/database"
	"github.com/gonglijing/xunjiFsu/internal/models"
)

const defaultDriverFunction = "handle"
const (
	defaultSerialReadTimeout = 200 * time.Millisecond
	defaultSerialOpenBackoff = 200 * time.Millisecond
	defaultTCPDialTimeout    = 5 * time.Second
	defaultTCPDialBackoff    = 200 * time.Millisecond
	defaultTCPReadTimeout    = 500 * time.Millisecond
)

func resolveResource(device *models.Device) (int64, string) {
	resourceID := int64(0)
	if device.ResourceID != nil {
		resourceID = *device.ResourceID
	}

	resourceType := device.ResourceType
	if resourceType == "" {
		resourceType = inferResourceType(device)
	}

	return resourceID, resourceType
}

func inferResourceType(device *models.Device) string {
	if device == nil {
		return "serial"
	}
	if device.ResourceType != "" {
		return strings.ToLower(strings.TrimSpace(device.ResourceType))
	}
	driverType := strings.ToLower(strings.TrimSpace(device.DriverType))
	switch {
	case strings.Contains(driverType, "tcp"), strings.Contains(driverType, "udp"), strings.Contains(driverType, "net"):
		return "net"
	case strings.Contains(driverType, "serial"), strings.Contains(driverType, "rtu"):
		return "serial"
	default:
		if device.IPAddress != "" || device.PortNum > 0 {
			return "net"
		}
		return "serial"
	}
}

func buildDeviceConfig(device *models.Device) map[string]string {
	deviceConfig := make(map[string]string)
	resourceType := inferResourceType(device)
	if resourceType == "serial" {
		deviceConfig["serial_port"] = device.SerialPort
		deviceConfig["baud_rate"] = fmt.Sprintf("%d", device.BaudRate)
		deviceConfig["data_bits"] = fmt.Sprintf("%d", device.DataBits)
		deviceConfig["stop_bits"] = fmt.Sprintf("%d", device.StopBits)
		deviceConfig["parity"] = device.Parity
	} else {
		deviceConfig["ip_address"] = device.IPAddress
		deviceConfig["port_num"] = fmt.Sprintf("%d", device.PortNum)
	}
	if device.DeviceAddress != "" {
		deviceConfig["device_address"] = device.DeviceAddress
	}
	deviceConfig["func_name"] = "read"
	return deviceConfig
}

func parseDriverResourceID(configSchema string) int64 {
	if strings.TrimSpace(configSchema) == "" {
		return 0
	}
	var cfg struct {
		ResourceID int64 `json:"resource_id"`
	}
	if err := json.Unmarshal([]byte(configSchema), &cfg); err != nil {
		return 0
	}
	return cfg.ResourceID
}

func buildDriverContext(device *models.Device, resourceID int64, resourceType string, deviceConfig map[string]string) *DriverContext {
	return &DriverContext{
		DeviceID:     device.ID,
		DeviceName:   device.Name,
		ResourceID:   resourceID,
		ResourceType: resourceType,
		Config:       deviceConfig,
		DeviceConfig: "",
	}
}

func (e *DriverExecutor) startExecution(device *models.Device) (func(), error) {
	e.mu.Lock()
	if e.executing[device.ID] {
		e.mu.Unlock()
		return nil, fmt.Errorf("device %s is already being read", device.Name)
	}
	e.executing[device.ID] = true
	e.mu.Unlock()

	return func() {
		e.mu.Lock()
		delete(e.executing, device.ID)
		e.mu.Unlock()
	}, nil
}

func (e *DriverExecutor) lockResource(resourceID int64) func() {
	if resourceID <= 0 {
		return nil
	}
	lock := e.getResourceLock(resourceID)
	lock.Lock()
	return lock.Unlock
}

func (e *DriverExecutor) ensureResourcePath(resourceID int64, resourceType string, device *models.Device) {
	if resourceType != "net" || resourceID <= 0 {
		return
	}
	if res, err := database.GetResourceByID(resourceID); err == nil && res != nil {
		path := strings.TrimSpace(res.Path)
		if path != "" {
			e.SetResourcePath(resourceID, path)
			return
		}
	}
	if e.GetResourcePath(resourceID) == "" && device.IPAddress != "" {
		path := fmt.Sprintf("%s:%d", device.IPAddress, device.PortNum)
		e.SetResourcePath(resourceID, path)
	}
}

func (e *DriverExecutor) ensureSerialResource(resourceID int64, resourceType string, device *models.Device) error {
	if resourceID <= 0 {
		return nil
	}
	if resourceType != "serial" {
		return nil
	}
	if err := e.ensureSerialPort(resourceID, device); err != nil {
		return fmt.Errorf("open serial resource %d failed: %w", resourceID, err)
	}
	return nil
}

func (e *DriverExecutor) serialReadTimeout() time.Duration {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.serialTimeout > 0 {
		return e.serialTimeout
	}
	return defaultSerialReadTimeout
}

func (e *DriverExecutor) serialOpenAttempts() int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.serialOpenRetries < 0 {
		return 1
	}
	return e.serialOpenRetries + 1
}

func (e *DriverExecutor) serialOpenBackoff() time.Duration {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.serialOpenBackoffOverride > 0 {
		return e.serialOpenBackoffOverride
	}
	return defaultSerialOpenBackoff
}

func (e *DriverExecutor) tcpDialTimeout() time.Duration {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.tcpDialTimeoutOverride > 0 {
		return e.tcpDialTimeoutOverride
	}
	return defaultTCPDialTimeout
}

func (e *DriverExecutor) tcpDialAttempts() int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.tcpDialRetries < 0 {
		return 1
	}
	return e.tcpDialRetries + 1
}

func (e *DriverExecutor) tcpDialBackoff() time.Duration {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.tcpDialBackoffOverride > 0 {
		return e.tcpDialBackoffOverride
	}
	return defaultTCPDialBackoff
}

func (e *DriverExecutor) tcpReadTimeout() time.Duration {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.tcpReadTimeoutOverride > 0 {
		return e.tcpReadTimeoutOverride
	}
	return defaultTCPReadTimeout
}

func (e *DriverExecutor) ensureDriverLoaded(device *models.Device, resourceID int64) error {
	driverID := *device.DriverID

	if e.manager.IsLoaded(driverID) {
		if loaded, err := e.manager.GetDriver(driverID); err == nil && loaded != nil {
			if resourceID <= 0 || loaded.resourceID == resourceID {
				return nil
			}
		}
	}

	drv, err := database.GetDriverByID(driverID)
	if err != nil || drv == nil || drv.FilePath == "" {
		return ErrDriverNotFound
	}
	if err := e.manager.LoadDriverFromModel(drv, resourceID); err != nil {
		return fmt.Errorf("load driver %d failed: %w", drv.ID, err)
	}
	return nil
}

// ReloadDeviceDriver 强制重载设备对应驱动
func (e *DriverExecutor) ReloadDeviceDriver(device *models.Device) error {
	if device == nil || device.DriverID == nil {
		return fmt.Errorf("device has no driver")
	}
	resourceID, _ := resolveResource(device)
	return e.ensureDriverLoaded(device, resourceID)
}
