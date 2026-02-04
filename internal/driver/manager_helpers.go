package driver

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/database"
	"github.com/gonglijing/xunjiFsu/internal/logger"
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

func validI64Ptr(ptr uint64) bool {
	return ptr != 0 && ptr <= uint64(^uint32(0))
}

func validI64PtrSize(ptr uint64, size int) bool {
	return validI64Ptr(ptr) && size > 0
}

func readWithTimeout(port SerialPort, buf []byte, expect int, timeout time.Duration) (int, error) {
	deadline := time.Now().Add(timeout)
	read := 0
	tmp := make([]byte, expect)
	for read < expect && time.Now().Before(deadline) {
		n, err := port.Read(tmp)
		if n > 0 {
			copy(buf[read:], tmp[:n])
			read += n
		}
		if err != nil {
			return read, err
		}
		if read >= expect {
			break
		}
		time.Sleep(2 * time.Millisecond)
	}
	if read < expect {
		return read, fmt.Errorf("timeout")
	}
	return read, nil
}

func resolveResource(device *models.Device) (int64, string) {
	resourceID := int64(0)
	if device.ResourceID != nil {
		resourceID = *device.ResourceID
	}

	resourceType := device.ResourceType
	if resourceType == "" {
		resourceType = device.DriverType
		if resourceType == "" {
			resourceType = "modbus_rtu"
		}
	}

	return resourceID, resourceType
}

func buildDeviceConfig(device *models.Device) map[string]string {
	deviceConfig := make(map[string]string)
	if device.DriverType == "modbus_rtu" {
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

var ErrPluginEmptyOutput = errors.New("plugin returned empty output")

func callPlugin(ctx context.Context, driver *WasmDriver, function string, input []byte) (uint32, []byte, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	rc, output, err := driver.plugin.CallWithContext(ctx, function, input)
	if err != nil {
		return rc, nil, err
	}

	if len(output) == 0 {
		if alt, err2 := driver.plugin.GetOutput(); err2 == nil && len(alt) > 0 {
			output = alt
		}
	}

	if len(output) == 0 {
		errMsg := driver.plugin.GetError()
		if errMsg != "" {
			return rc, nil, fmt.Errorf("%w: %s", ErrPluginEmptyOutput, errMsg)
		}
		return rc, nil, ErrPluginEmptyOutput
	}

	return rc, output, nil
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
	if e.GetResourcePath(resourceID) == "" && device.IPAddress != "" {
		path := fmt.Sprintf("%s:%d", device.IPAddress, device.PortNum)
		e.SetResourcePath(resourceID, path)
	}
}

func (e *DriverExecutor) ensureSerialResource(resourceID int64, resourceType string, device *models.Device) error {
	if resourceID <= 0 {
		return nil
	}
	if err := e.ensureSerialPort(resourceID, device); err != nil && resourceType == "serial" {
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
			if resourceID > 0 && loaded.resourceID != resourceID {
				logger.Warn("Reloading driver with new resource", "driver_id", loaded.ID, "old_resource_id", loaded.resourceID, "new_resource_id", resourceID)
				_ = e.manager.UnloadDriver(loaded.ID)
			}
		}
	}

	if e.manager.IsLoaded(driverID) {
		return nil
	}

	drv, err := database.GetDriverByID(driverID)
	if err != nil || drv == nil || drv.FilePath == "" {
		return ErrDriverNotFound
	}
	wasmData, err := readWasmFile(drv.FilePath)
	if err != nil {
		return fmt.Errorf("read driver wasm failed: %w", err)
	}
	if err := e.manager.LoadDriver(drv, wasmData, resourceID); err != nil {
		return fmt.Errorf("load driver %d failed: %w", drv.ID, err)
	}
	if version, err := e.manager.GetDriverVersion(driverID); err == nil && version != "" {
		if err := database.UpdateDriverVersion(driverID, version); err != nil {
			logger.Warn("Update driver version failed", "driver_id", driverID, "error", err)
		}
	}
	return nil
}
