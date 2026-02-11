//go:build no_extism

package driver

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/models"
)

// ErrDriverNotFound 驱动未找到
var ErrDriverNotFound = errors.New("driver not found")

// ErrDriverNotLoaded 驱动未加载
var ErrDriverNotLoaded = errors.New("driver not loaded")

// ErrDriverExecutionFailed 驱动执行失败
var ErrDriverExecutionFailed = errors.New("driver execution failed")

// ErrDriverTimeout 驱动执行超时
var ErrDriverTimeout = errors.New("driver execution timeout")

// ErrDriverCanceled 驱动执行被取消
var ErrDriverCanceled = errors.New("driver execution canceled")

// ErrDriverBadOutput 驱动输出无效
var ErrDriverBadOutput = errors.New("driver output invalid")

// DriverResult 驱动执行结果
type DriverResult struct {
	Success   bool              `json:"success"`
	Data      map[string]string `json:"data"`
	Points    []DriverPoint     `json:"points"`
	Error     string            `json:"error"`
	Timestamp time.Time         `json:"timestamp"`
}

// DriverPoint 驱动测点数据
type DriverPoint struct {
	FieldName string      `json:"field_name"`
	Value     interface{} `json:"value"`
	RW        string      `json:"rw"`
}

// DriverContext 驱动上下文
type DriverContext struct {
	DeviceID     int64             `json:"device_id"`
	DeviceName   string            `json:"device_name"`
	ResourceID   int64             `json:"resource_id"`
	ResourceType string            `json:"resource_type"`
	Config       map[string]string `json:"config"`
	DeviceConfig string            `json:"device_config"`
}

// SerialPort 串口接口
type SerialPort interface {
	Write([]byte) (int, error)
	Read([]byte) (int, error)
	Close() error
}

// WasmDriver no_extism 下的占位驱动
type WasmDriver struct {
	ID         int64
	Name       string
	lastActive time.Time
	config     string
	resourceID int64
}

// DriverManager 驱动管理器
type DriverManager struct {
	drivers     map[int64]*WasmDriver
	executor    *DriverExecutor
	callTimeout time.Duration
}

func NewDriverManager() *DriverManager {
	return &DriverManager{drivers: make(map[int64]*WasmDriver)}
}

func (m *DriverManager) SetExecutor(executor *DriverExecutor) { m.executor = executor }
func (m *DriverManager) SetCallTimeout(timeout time.Duration) { m.callTimeout = timeout }

func (m *DriverManager) LoadDriver(driver *models.Driver, wasmData []byte, resourceID int64) error {
	if driver == nil {
		return fmt.Errorf("driver is nil")
	}
	if resourceID == 0 {
		resourceID = parseDriverResourceID(driver.ConfigSchema)
	}
	m.drivers[driver.ID] = &WasmDriver{
		ID:         driver.ID,
		Name:       driver.Name,
		lastActive: time.Now(),
		config:     driver.ConfigSchema,
		resourceID: resourceID,
	}
	return nil
}

func (m *DriverManager) ReloadDriver(driver *models.Driver, wasmData []byte, resourceID int64) error {
	return m.LoadDriver(driver, wasmData, resourceID)
}

func (m *DriverManager) UnloadDriver(id int64) error {
	if _, ok := m.drivers[id]; !ok {
		return ErrDriverNotFound
	}
	delete(m.drivers, id)
	return nil
}

func (m *DriverManager) ExecuteDriver(id int64, function string, driverCtx *DriverContext) (*DriverResult, error) {
	return m.ExecuteDriverWithContext(context.Background(), id, function, driverCtx)
}

func (m *DriverManager) ExecuteDriverWithContext(ctx context.Context, id int64, function string, driverCtx *DriverContext) (*DriverResult, error) {
	if _, ok := m.drivers[id]; !ok {
		return nil, ErrDriverNotFound
	}
	return nil, fmt.Errorf("%w: extism disabled by build tag no_extism", ErrDriverExecutionFailed)
}

func (m *DriverManager) GetDriver(id int64) (*WasmDriver, error) {
	d, ok := m.drivers[id]
	if !ok {
		return nil, ErrDriverNotFound
	}
	return d, nil
}

func (m *DriverManager) ListDrivers() []*WasmDriver {
	list := make([]*WasmDriver, 0, len(m.drivers))
	for _, d := range m.drivers {
		list = append(list, d)
	}
	return list
}

func (m *DriverManager) IsLoaded(id int64) bool {
	_, ok := m.drivers[id]
	return ok
}

func (m *DriverManager) GetDriverResourceID(id int64) (int64, error) {
	d, ok := m.drivers[id]
	if !ok {
		return 0, ErrDriverNotFound
	}
	return d.resourceID, nil
}

func (m *DriverManager) LoadDriverFromModel(driver *models.Driver, resourceID int64) error {
	if driver == nil {
		return fmt.Errorf("driver is nil")
	}
	if resourceID == 0 {
		resourceID = parseDriverResourceID(driver.ConfigSchema)
	}
	if driver.FilePath == "" {
		return fmt.Errorf("driver file path is empty")
	}
	wasmData, err := readWasmFile(driver.FilePath)
	if err != nil {
		return fmt.Errorf("read driver wasm failed: %w", err)
	}
	if err := m.ReloadDriver(driver, wasmData, resourceID); err != nil {
		return fmt.Errorf("reload driver failed: %w", err)
	}
	return nil
}

// DriverRuntime 驱动运行时信息
type DriverRuntime struct {
	ID                int64     `json:"id"`
	Name              string    `json:"name"`
	Loaded            bool      `json:"loaded"`
	ResourceID        int64     `json:"resource_id"`
	LastActive        time.Time `json:"last_active"`
	ExportedFunctions []string  `json:"exported_functions,omitempty"`
}

func runtimeFromDriver(driver *WasmDriver) *DriverRuntime {
	if driver == nil {
		return nil
	}
	return &DriverRuntime{
		ID:         driver.ID,
		Name:       driver.Name,
		Loaded:     true,
		ResourceID: driver.resourceID,
		LastActive: driver.lastActive,
	}
}

func (m *DriverManager) GetRuntime(id int64) (*DriverRuntime, error) {
	d, ok := m.drivers[id]
	if !ok {
		return nil, ErrDriverNotLoaded
	}
	return runtimeFromDriver(d), nil
}

func (m *DriverManager) ListRuntimes() []*DriverRuntime {
	runtimes := make([]*DriverRuntime, 0, len(m.drivers))
	for _, d := range m.drivers {
		if rt := runtimeFromDriver(d); rt != nil {
			runtimes = append(runtimes, rt)
		}
	}
	return runtimes
}

func (m *DriverManager) GetDriverVersion(id int64) (string, error) {
	if _, ok := m.drivers[id]; !ok {
		return "", ErrDriverNotFound
	}
	return "", nil
}

func ExtractDriverVersion(wasmData []byte) (string, error) {
	if len(wasmData) == 0 {
		return "", fmt.Errorf("empty wasm data")
	}
	return "", nil
}
