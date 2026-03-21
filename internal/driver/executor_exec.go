package driver

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/gonglijing/xunjiFsu/internal/models"
)

// Execute 执行驱动读取（带资源访问锁）
func (e *DriverExecutor) Execute(device *models.Device) (*DriverResult, error) {
	return e.ExecuteWithContext(context.Background(), device)
}

// ExecuteWithContext 执行驱动读取（支持超时/取消）
func (e *DriverExecutor) ExecuteWithContext(ctx context.Context, device *models.Device) (*DriverResult, error) {
	return e.executePreparedWithContextAndConfig(ctx, device, defaultDriverFunction, NewPreparedExecution(device), nil)
}

// ExecuteCommand 执行指定函数（用于写入等主动命令）
func (e *DriverExecutor) ExecuteCommand(device *models.Device, function string, config map[string]string) (*DriverResult, error) {
	return e.ExecuteCommandWithContext(context.Background(), device, function, config)
}

// ExecuteCommandWithContext 执行指定函数（支持超时/取消）
func (e *DriverExecutor) ExecuteCommandWithContext(ctx context.Context, device *models.Device, function string, config map[string]string) (*DriverResult, error) {
	return e.executeWithContextAndConfig(ctx, device, function, config)
}

// ExecutePrepared executes a driver call with a cached device execution context.
func (e *DriverExecutor) ExecutePrepared(device *models.Device, function string, prepared *PreparedExecution, overrides map[string]string) (*DriverResult, error) {
	return e.ExecutePreparedWithContext(context.Background(), device, function, prepared, overrides)
}

// ExecutePreparedWithContext executes a driver call with a cached device execution context.
func (e *DriverExecutor) ExecutePreparedWithContext(ctx context.Context, device *models.Device, function string, prepared *PreparedExecution, overrides map[string]string) (*DriverResult, error) {
	return e.executePreparedWithContextAndConfig(ctx, device, function, prepared, overrides)
}

func (e *DriverExecutor) executeWithContextAndConfig(ctx context.Context, device *models.Device, function string, overrides map[string]string) (*DriverResult, error) {
	return e.executePreparedWithContextAndConfig(ctx, device, function, NewPreparedExecution(device), overrides)
}

func (e *DriverExecutor) executePreparedWithContextAndConfig(ctx context.Context, device *models.Device, function string, prepared *PreparedExecution, overrides map[string]string) (*DriverResult, error) {
	if device.DriverID == nil {
		return nil, fmt.Errorf("device %s has no driver", device.Name)
	}
	done, err := e.startExecution(device)
	if err != nil {
		return nil, err
	}
	defer done()

	prepared = normalizePreparedExecution(device, prepared)
	resourceID := prepared.ResourceID
	resourceType := prepared.ResourceType
	e.ensureResourcePath(resourceID, resourceType, device)

	pluginFunc := resolveExecutionFunction(function)
	driverCtx := prepared.DriverContext
	inputJSON := prepared.InputJSON
	if len(overrides) > 0 {
		deviceConfig := cloneDeviceConfig(prepared.Config, len(overrides))
		mergeDeviceConfig(deviceConfig, overrides)
		driverCtx = cloneDriverContext(prepared.DriverContext, deviceConfig)
		inputJSON, err = marshalDriverInvocationInput(driverCtx)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal input: %w", err)
		}
	}

	unlock := e.lockResource(resourceID)
	if unlock != nil {
		defer unlock()
	}

	if err := e.ensureSerialResource(resourceID, resourceType, device); err != nil {
		return nil, err
	}

	if err := e.ensureDriverLoaded(device, resourceID); err != nil {
		return nil, err
	}

	return e.manager.executeDriverWithInput(ctx, *device.DriverID, pluginFunc, driverCtx, inputJSON)
}

func normalizePreparedExecution(device *models.Device, prepared *PreparedExecution) *PreparedExecution {
	if prepared == nil || prepared.DriverContext == nil || prepared.Config == nil || prepared.InputJSON == nil {
		return NewPreparedExecution(device)
	}
	return prepared
}

func cloneDeviceConfig(base map[string]string, extra int) map[string]string {
	if len(base) == 0 {
		if extra <= 0 {
			return nil
		}
		return make(map[string]string, extra)
	}
	cloned := make(map[string]string, len(base)+extra)
	for key, value := range base {
		cloned[key] = value
	}
	return cloned
}

func cloneDriverContext(base *DriverContext, deviceConfig map[string]string) *DriverContext {
	if base == nil {
		return nil
	}
	cloned := *base
	cloned.Config = deviceConfig
	return &cloned
}

func mergeDeviceConfig(base, overrides map[string]string) {
	if base == nil || len(overrides) == 0 {
		return
	}
	for key, value := range overrides {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			continue
		}
		base[trimmedKey] = value
	}
}

func resolveExecutionFunction(function string) string {
	resolved := strings.TrimSpace(function)
	if resolved == "" {
		return defaultDriverFunction
	}
	return resolved
}

// CollectData 采集数据
func (e *DriverExecutor) CollectData(device *models.Device) (*models.CollectData, error) {
	return e.CollectDataWithContext(context.Background(), device)
}

// CollectDataWithContext 采集数据（支持超时/取消）
func (e *DriverExecutor) CollectDataWithContext(ctx context.Context, device *models.Device) (*models.CollectData, error) {
	result, err := e.ExecuteWithContext(ctx, device)
	if err != nil {
		return nil, err
	}

	if !result.Success {
		return nil, errors.New(result.Error)
	}

	// 解析返回数据
	fields := ResultFields(result)

	return &models.CollectData{
		DeviceID:   device.ID,
		DeviceName: device.Name,
		Timestamp:  result.Timestamp,
		Fields:     fields,
	}, nil
}

// ResultFields converts a driver result into field data.
// When the legacy data map is already clean, it may be returned directly.
func ResultFields(result *DriverResult) map[string]string {
	return mapResultFields(result)
}

func mapResultFields(result *DriverResult) map[string]string {
	if result == nil {
		return nil
	}

	if len(result.Points) > 0 {
		var fields map[string]string
		for _, point := range result.Points {
			name := trimDriverFieldName(point.FieldName)
			if name == "" {
				continue
			}
			if isNormalizedDriverIdentityField(name) {
				continue
			}
			if fields == nil {
				fields = make(map[string]string, len(result.Points))
			}
			fields[name] = formatDriverValue(point.Value)
		}
		return fields
	}

	if len(result.Data) == 0 {
		return nil
	}
	if canReuseResultData(result.Data) {
		return result.Data
	}

	var fields map[string]string
	for key, value := range result.Data {
		name := trimDriverFieldName(key)
		if name == "" {
			continue
		}
		if isNormalizedDriverIdentityField(name) {
			continue
		}
		if fields == nil {
			fields = make(map[string]string, len(result.Data))
		}
		fields[name] = value
	}
	return fields
}

func canReuseResultData(data map[string]string) bool {
	for key := range data {
		name := trimDriverFieldName(key)
		if name == "" {
			return false
		}
		if key != name {
			return false
		}
		if isNormalizedDriverIdentityField(name) {
			return false
		}
	}
	return true
}

func formatDriverValue(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return v
	case []byte:
		return string(v)
	case bool:
		return strconv.FormatBool(v)
	case int:
		return strconv.FormatInt(int64(v), 10)
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
		return strconv.FormatFloat(float64(v), 'f', 6, 32)
	case float64:
		return strconv.FormatFloat(v, 'f', 6, 64)
	default:
		return fmt.Sprintf("%v", v)
	}
}
