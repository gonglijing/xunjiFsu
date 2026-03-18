package driver

import (
	"io"
	"net"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/database"
	"github.com/gonglijing/xunjiFsu/internal/models"
)

func TestResolveDeviceResourceDefaults(t *testing.T) {
	device := &models.Device{}
	resourceID, resourceType := resolveDeviceResource(device)
	if resourceID != 0 {
		t.Fatalf("expected resourceID 0, got %d", resourceID)
	}
	if resourceType != "serial" {
		t.Fatalf("expected resourceType serial, got %s", resourceType)
	}
}

func TestResolveDeviceResourceDriverTypeFallback(t *testing.T) {
	id := int64(9)
	device := &models.Device{
		ResourceID: &id,
		DriverType: "net",
	}
	resourceID, resourceType := resolveDeviceResource(device)
	if resourceID != id {
		t.Fatalf("expected resourceID %d, got %d", id, resourceID)
	}
	if resourceType != "net" {
		t.Fatalf("expected resourceType net, got %s", resourceType)
	}
}

func TestResolveDeviceResourceExplicitType(t *testing.T) {
	device := &models.Device{
		ResourceType: "serial",
		DriverType:   "net",
	}
	_, resourceType := resolveDeviceResource(device)
	if resourceType != "serial" {
		t.Fatalf("expected resourceType serial, got %s", resourceType)
	}
}

func TestResolveDeviceResourceModbusTypeSkipsDatabaseLookup(t *testing.T) {
	id := int64(9)
	device := &models.Device{
		ResourceID: &id,
		DriverType: "modbus_tcp",
	}

	tmpDir := t.TempDir()
	paramPath := filepath.Join(tmpDir, "param.db")
	originalParamDB := database.ParamDB
	t.Cleanup(func() {
		if database.ParamDB != nil {
			_ = database.ParamDB.Close()
		}
		database.ParamDB = originalParamDB
	})
	if err := database.InitParamDBWithPath(paramPath); err != nil {
		t.Fatalf("InitParamDBWithPath failed: %v", err)
	}

	resourceID, resourceType := resolveDeviceResource(device)
	if resourceID != id {
		t.Fatalf("expected resourceID %d, got %d", id, resourceID)
	}
	if resourceType != "net" {
		t.Fatalf("expected resourceType net, got %s", resourceType)
	}
}

func TestBuildRecoverableDriverNames(t *testing.T) {
	got := buildRecoverableDriverNames(" modbus_tcp ")
	want := []string{"modbus_tcp", "th_modbustcp"}

	if len(got) != len(want) {
		t.Fatalf("len(got) = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestBuildRecoverableDriverNames_EmptyType(t *testing.T) {
	got := buildRecoverableDriverNames(" ")
	if got != nil {
		t.Fatalf("expected nil candidate names, got %#v", got)
	}
}

func TestBuildDeviceConfigModbusRTU(t *testing.T) {
	device := &models.Device{
		DriverType:    "modbus_rtu",
		SerialPort:    "/dev/ttyUSB0",
		BaudRate:      9600,
		DataBits:      8,
		StopBits:      1,
		Parity:        "N",
		DeviceAddress: "1",
	}
	config := buildDeviceConfig(device)
	assertMapEqual(t, config, map[string]string{
		"serial_port":    "/dev/ttyUSB0",
		"baud_rate":      "9600",
		"data_bits":      "8",
		"stop_bits":      "1",
		"parity":         "N",
		"device_address": "1",
		"func_name":      "read",
	})
}

func TestBuildDeviceConfigTCP(t *testing.T) {
	device := &models.Device{
		DriverType: "modbus_tcp",
		IPAddress:  "127.0.0.1",
		PortNum:    502,
	}
	config := buildDeviceConfig(device)
	assertMapEqual(t, config, map[string]string{
		"ip_address": "127.0.0.1",
		"port_num":   "502",
		"func_name":  "read",
	})
	if _, ok := config["serial_port"]; ok {
		t.Fatalf("did not expect serial_port in tcp config")
	}
}

func TestNewPreparedExecution(t *testing.T) {
	device := &models.Device{
		ID:            11,
		Name:          "dev-prepared",
		DriverType:    "modbus_rtu",
		SerialPort:    "/dev/ttyUSB1",
		BaudRate:      9600,
		DataBits:      8,
		StopBits:      1,
		Parity:        "N",
		DeviceAddress: "2",
	}

	prepared := NewPreparedExecution(device)
	if prepared == nil {
		t.Fatalf("prepared execution should not be nil")
	}
	if prepared.DriverContext == nil {
		t.Fatalf("prepared driver context should not be nil")
	}
	if prepared.DriverContext.DeviceID != device.ID {
		t.Fatalf("prepared device id = %d, want %d", prepared.DriverContext.DeviceID, device.ID)
	}
	if prepared.Config["serial_port"] != "/dev/ttyUSB1" {
		t.Fatalf("prepared config missing serial_port: %#v", prepared.Config)
	}
	if len(prepared.InputJSON) == 0 {
		t.Fatalf("prepared input json should not be empty")
	}
	if !strings.Contains(string(prepared.InputJSON), "\"device_id\":11") {
		t.Fatalf("prepared input json should contain device id, got %s", string(prepared.InputJSON))
	}
	prepared.Config["probe"] = "x"
	if prepared.DriverContext.Config["probe"] != "x" {
		t.Fatalf("driver context should share prepared config map")
	}
}

func TestMergeDeviceConfig(t *testing.T) {
	base := map[string]string{
		"func_name": "read",
	}
	overrides := map[string]string{
		" field_a ": "1",
		"field_b":   "2",
		"   ":       "ignored",
		"":          "ignored",
	}

	mergeDeviceConfig(base, overrides)

	assertMapEqual(t, base, map[string]string{
		"func_name": "read",
		"field_a":   "1",
		"field_b":   "2",
	})
}

func TestMergeDeviceConfig_BaseNilOrNoOverrides(t *testing.T) {
	mergeDeviceConfig(nil, map[string]string{"a": "1"})

	base := map[string]string{"x": "1"}
	mergeDeviceConfig(base, nil)
	if base["x"] != "1" {
		t.Fatalf("base map should keep original values when overrides nil")
	}
	mergeDeviceConfig(base, map[string]string{})
	if base["x"] != "1" {
		t.Fatalf("base map should keep original values when overrides empty")
	}
}

func TestResolveExecutionFunction(t *testing.T) {
	if got := resolveExecutionFunction(""); got != defaultDriverFunction {
		t.Fatalf("expected default function, got %q", got)
	}
	if got := resolveExecutionFunction("   "); got != defaultDriverFunction {
		t.Fatalf("expected default function for blank input, got %q", got)
	}
	if got := resolveExecutionFunction(" write "); got != "write" {
		t.Fatalf("expected trimmed function name, got %q", got)
	}
}

func TestCloneDriverContextWithOverridesDoesNotMutateBase(t *testing.T) {
	baseConfig := map[string]string{
		"func_name": "read",
		"address":   "1",
	}
	baseCtx := &DriverContext{
		DeviceID:     1,
		DeviceName:   "dev",
		ResourceID:   2,
		ResourceType: "serial",
		Config:       baseConfig,
	}

	clonedConfig := cloneDeviceConfig(baseConfig, 1)
	mergeDeviceConfig(clonedConfig, map[string]string{"func_name": "write"})
	clonedCtx := cloneDriverContext(baseCtx, clonedConfig)

	if baseConfig["func_name"] != "read" {
		t.Fatalf("base config mutated: %#v", baseConfig)
	}
	if clonedCtx == nil || clonedCtx.Config["func_name"] != "write" {
		t.Fatalf("cloned context should contain override: %#v", clonedCtx)
	}
}

func TestWasmDriverHasFunctionUsesCachedExports(t *testing.T) {
	driver := &WasmDriver{
		exportedSet: map[string]struct{}{
			"handle":  {},
			"version": {},
		},
	}

	if !driver.hasFunction("handle") {
		t.Fatal("expected cached handle export")
	}
	if driver.hasFunction("missing") {
		t.Fatal("did not expect missing export")
	}
}

func TestBuildDriverRuntimeUsesCachedExports(t *testing.T) {
	driver := &WasmDriver{
		ID:                1,
		Name:              "modbus_tcp",
		resourceID:        7,
		lastActive:        time.Unix(1700000000, 0),
		version:           "1.2.3",
		productKey:        "pk-1",
		exportedFunctions: []string{"handle", "version"},
	}

	runtime := buildDriverRuntime(driver)
	if runtime == nil {
		t.Fatal("runtime should not be nil")
	}
	if runtime.Version != "1.2.3" {
		t.Fatalf("runtime.Version = %q, want 1.2.3", runtime.Version)
	}
	if runtime.ProductKey != "pk-1" {
		t.Fatalf("runtime.ProductKey = %q, want pk-1", runtime.ProductKey)
	}
	if len(runtime.ExportedFunctions) != 2 {
		t.Fatalf("unexpected exported functions: %#v", runtime.ExportedFunctions)
	}

	runtime.ExportedFunctions[0] = "mutated"
	if driver.exportedFunctions[0] != "handle" {
		t.Fatalf("driver cached exports should not be mutated: %#v", driver.exportedFunctions)
	}
}

func assertMapEqual(t *testing.T, got, want map[string]string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("expected map size %d, got %d", len(want), len(got))
	}
	for key, wantVal := range want {
		gotVal, ok := got[key]
		if !ok {
			t.Fatalf("missing key %s", key)
		}
		if gotVal != wantVal {
			t.Fatalf("key %s expected %s got %s", key, wantVal, gotVal)
		}
	}
}

type fakeSerialPort struct {
	closed bool
}

func (p *fakeSerialPort) Write(_ []byte) (int, error) {
	return 0, nil
}

func (p *fakeSerialPort) Read(_ []byte) (int, error) {
	return 0, io.EOF
}

func (p *fakeSerialPort) Close() error {
	p.closed = true
	return nil
}

func TestStartExecution(t *testing.T) {
	manager := NewDriverManager()
	executor := NewDriverExecutor(manager)
	device := &models.Device{ID: 1, Name: "dev1"}

	done, err := executor.startExecution(device)
	if err != nil {
		t.Fatalf("startExecution error: %v", err)
	}

	if _, err := executor.startExecution(device); err == nil {
		t.Fatalf("expected concurrent startExecution to fail")
	}

	done()

	if done2, err := executor.startExecution(device); err != nil {
		t.Fatalf("expected startExecution after done to succeed: %v", err)
	} else {
		done2()
	}
}

func TestUnregisterSerialPortCloses(t *testing.T) {
	manager := NewDriverManager()
	executor := NewDriverExecutor(manager)
	port := &fakeSerialPort{}

	executor.RegisterSerialPort(1, port)
	executor.UnregisterSerialPort(1)

	if !port.closed {
		t.Fatalf("expected serial port to be closed on unregister")
	}
}

func TestSetResourcePathClosesTCPOnChange(t *testing.T) {
	manager := NewDriverManager()
	executor := NewDriverExecutor(manager)

	c1, c2 := net.Pipe()
	defer c2.Close()

	executor.RegisterTCP(1, c1)
	executor.SetResourcePath(1, "127.0.0.1:502")
	executor.SetResourcePath(1, "127.0.0.1:503")

	if _, err := c1.Write([]byte("x")); err == nil {
		t.Fatalf("expected closed conn after path change")
	}

	if _, err := c2.Write([]byte("x")); err == nil {
		t.Fatalf("expected other end to be closed")
	}
}

func TestSetTimeoutsOverridesDefaults(t *testing.T) {
	manager := NewDriverManager()
	executor := NewDriverExecutor(manager)

	executor.SetTimeouts(2*time.Second, 3*time.Second, 4*time.Second)

	if got := executor.serialReadTimeout(); got != 2*time.Second {
		t.Fatalf("expected serial timeout 2s, got %v", got)
	}
	if got := executor.tcpDialTimeout(); got != 3*time.Second {
		t.Fatalf("expected tcp dial timeout 3s, got %v", got)
	}
	if got := executor.tcpReadTimeout(); got != 4*time.Second {
		t.Fatalf("expected tcp read timeout 4s, got %v", got)
	}
}

func TestSetRetriesOverridesDefaults(t *testing.T) {
	manager := NewDriverManager()
	executor := NewDriverExecutor(manager)

	if got := executor.serialOpenAttempts(); got != 1 {
		t.Fatalf("expected default serial attempts 1, got %d", got)
	}
	if got := executor.tcpDialAttempts(); got != 1 {
		t.Fatalf("expected default tcp attempts 1, got %d", got)
	}

	executor.SetRetries(2, 3, 150*time.Millisecond, 250*time.Millisecond)

	if got := executor.serialOpenAttempts(); got != 3 {
		t.Fatalf("expected serial attempts 3, got %d", got)
	}
	if got := executor.tcpDialAttempts(); got != 4 {
		t.Fatalf("expected tcp attempts 4, got %d", got)
	}
	if got := executor.serialOpenBackoff(); got != 150*time.Millisecond {
		t.Fatalf("expected serial backoff 150ms, got %v", got)
	}
	if got := executor.tcpDialBackoff(); got != 250*time.Millisecond {
		t.Fatalf("expected tcp backoff 250ms, got %v", got)
	}
}

func TestGetTCPConnReturnsRegisteredConnectionWithoutProbeWrite(t *testing.T) {
	manager := NewDriverManager()
	executor := NewDriverExecutor(manager)

	c1, c2 := net.Pipe()
	defer c1.Close()
	defer c2.Close()

	executor.RegisterTCP(12, c1)

	got := executor.GetTCPConn(12)
	if got != c1 {
		t.Fatalf("expected existing registered connection")
	}

	readDone := make(chan struct{})
	go func() {
		buf := make([]byte, 1)
		_, _ = c2.Read(buf)
		close(readDone)
	}()

	select {
	case <-readDone:
		t.Fatalf("expected no unsolicited probe write on existing TCP connection")
	case <-time.After(50 * time.Millisecond):
	}
}

func TestGetTCPConnConcurrentDialReusesSingleConnection(t *testing.T) {
	manager := NewDriverManager()
	executor := NewDriverExecutor(manager)
	executor.SetResourcePath(21, "127.0.0.1:502")

	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()

	var dialCount int32
	start := make(chan struct{})
	executor.tcpDialFn = func(network, address string, timeout time.Duration) (net.Conn, error) {
		atomic.AddInt32(&dialCount, 1)
		<-start
		return clientConn, nil
	}

	const workers = 8
	var wg sync.WaitGroup
	results := make([]net.Conn, workers)
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func(idx int) {
			defer wg.Done()
			results[idx] = executor.GetTCPConn(21)
		}(i)
	}

	time.Sleep(20 * time.Millisecond)
	close(start)
	wg.Wait()

	if got := atomic.LoadInt32(&dialCount); got != 1 {
		t.Fatalf("dial count = %d, want 1", got)
	}
	for i, conn := range results {
		if conn != clientConn {
			t.Fatalf("results[%d] did not reuse shared connection", i)
		}
	}
}

func TestModbusFrameBufferPool(t *testing.T) {
	buf := getModbusFrameBuffer(128)
	if len(buf) != 128 {
		t.Fatalf("len(buf) = %d, want 128", len(buf))
	}
	if cap(buf) != pooledModbusFrameSize {
		t.Fatalf("cap(buf) = %d, want %d", cap(buf), pooledModbusFrameSize)
	}
	putModbusFrameBuffer(buf)

	large := getModbusFrameBuffer(pooledModbusFrameSize + 1)
	if len(large) != pooledModbusFrameSize+1 {
		t.Fatalf("len(large) = %d, want %d", len(large), pooledModbusFrameSize+1)
	}
	if cap(large) < pooledModbusFrameSize+1 {
		t.Fatalf("cap(large) = %d, want >= %d", cap(large), pooledModbusFrameSize+1)
	}
	putModbusFrameBuffer(large)
}

func TestEnsureResourcePathLoadsFromResourcePath(t *testing.T) {
	tmpDir := t.TempDir()
	paramPath := filepath.Join(tmpDir, "param.db")
	originalParamDB := database.ParamDB
	t.Cleanup(func() {
		if database.ParamDB != nil {
			_ = database.ParamDB.Close()
		}
		database.ParamDB = originalParamDB
	})

	if err := database.InitParamDBWithPath(paramPath); err != nil {
		t.Fatalf("InitParamDBWithPath failed: %v", err)
	}
	if _, err := database.ParamDB.Exec(`CREATE TABLE IF NOT EXISTS resources (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		type TEXT NOT NULL,
		path TEXT,
		enabled INTEGER DEFAULT 1,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`); err != nil {
		t.Fatalf("create resources table failed: %v", err)
	}

	resourceID, err := database.AddResource(&models.Resource{
		Name:    "net-r1",
		Type:    "net",
		Path:    "192.168.10.20:502",
		Enabled: 1,
	})
	if err != nil {
		t.Fatalf("AddResource failed: %v", err)
	}

	manager := NewDriverManager()
	executor := NewDriverExecutor(manager)
	device := &models.Device{IPAddress: "", PortNum: 0}

	executor.ensureResourcePath(resourceID, "net", device)

	got := executor.GetResourcePath(resourceID)
	if got != "192.168.10.20:502" {
		t.Fatalf("resource path = %q, want %q", got, "192.168.10.20:502")
	}
}

func TestEnsureResourcePathPrefersCachedValue(t *testing.T) {
	tmpDir := t.TempDir()
	paramPath := filepath.Join(tmpDir, "param.db")
	originalParamDB := database.ParamDB
	t.Cleanup(func() {
		if database.ParamDB != nil {
			_ = database.ParamDB.Close()
		}
		database.ParamDB = originalParamDB
	})

	if err := database.InitParamDBWithPath(paramPath); err != nil {
		t.Fatalf("InitParamDBWithPath failed: %v", err)
	}
	if _, err := database.ParamDB.Exec(`CREATE TABLE IF NOT EXISTS resources (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		type TEXT NOT NULL,
		path TEXT,
		enabled INTEGER DEFAULT 1,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`); err != nil {
		t.Fatalf("create resources table failed: %v", err)
	}

	resourceID, err := database.AddResource(&models.Resource{
		Name:    "net-r2",
		Type:    "net",
		Path:    "192.168.10.20:502",
		Enabled: 1,
	})
	if err != nil {
		t.Fatalf("AddResource failed: %v", err)
	}

	executor := NewDriverExecutor(NewDriverManager())
	device := &models.Device{}

	executor.ensureResourcePath(resourceID, "net", device)
	if got := executor.GetResourcePath(resourceID); got != "192.168.10.20:502" {
		t.Fatalf("resource path = %q, want %q", got, "192.168.10.20:502")
	}

	if err := database.UpdateResource(&models.Resource{
		ID:      resourceID,
		Name:    "net-r2",
		Type:    "net",
		Path:    "192.168.10.20:503",
		Enabled: 1,
	}); err != nil {
		t.Fatalf("UpdateResource failed: %v", err)
	}

	executor.ensureResourcePath(resourceID, "net", device)
	if got := executor.GetResourcePath(resourceID); got != "192.168.10.20:502" {
		t.Fatalf("cached resource path = %q, want %q", got, "192.168.10.20:502")
	}
}

func TestRefreshResourceClosesDisabledNetConnection(t *testing.T) {
	executor := NewDriverExecutor(NewDriverManager())

	c1, c2 := net.Pipe()
	defer c2.Close()

	executor.RegisterTCP(7, c1)
	executor.SetResourcePath(7, "127.0.0.1:502")
	executor.RefreshResource(&models.Resource{
		ID:      7,
		Type:    "net",
		Path:    "127.0.0.1:502",
		Enabled: 0,
	})

	if got := executor.GetResourcePath(7); got != "" {
		t.Fatalf("resource path = %q, want empty", got)
	}
	if _, err := c1.Write([]byte("x")); err == nil {
		t.Fatalf("expected closed conn after resource disable")
	}
}

func TestRecoverMissingDriverBindingByDriverType(t *testing.T) {
	tmpDir := t.TempDir()
	paramPath := filepath.Join(tmpDir, "param.db")
	originalParamDB := database.ParamDB
	t.Cleanup(func() {
		if database.ParamDB != nil {
			_ = database.ParamDB.Close()
		}
		database.ParamDB = originalParamDB
	})

	if err := database.InitParamDBWithPath(paramPath); err != nil {
		t.Fatalf("InitParamDBWithPath failed: %v", err)
	}
	if _, err := database.ParamDB.Exec(`CREATE TABLE drivers (
		id INTEGER PRIMARY KEY,
		name TEXT UNIQUE NOT NULL,
		file_path TEXT NOT NULL,
		description TEXT,
		version TEXT,
		config_schema TEXT,
		enabled INTEGER DEFAULT 1,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`); err != nil {
		t.Fatalf("create drivers table failed: %v", err)
	}
	if _, err := database.ParamDB.Exec(`CREATE TABLE devices (
		id INTEGER PRIMARY KEY,
		name TEXT,
		driver_id INTEGER,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`); err != nil {
		t.Fatalf("create devices table failed: %v", err)
	}

	driverID, err := database.CreateDriver(&models.Driver{
		Name:     "modbus_rtu",
		FilePath: "drivers/modbus_rtu.wasm",
		Enabled:  1,
	})
	if err != nil {
		t.Fatalf("CreateDriver failed: %v", err)
	}

	if _, err := database.ParamDB.Exec(`INSERT INTO devices(id, name, driver_id) VALUES(?,?,?)`, 101, "dev-101", 9999); err != nil {
		t.Fatalf("insert device failed: %v", err)
	}

	executor := NewDriverExecutor(NewDriverManager())
	dev := &models.Device{ID: 101, Name: "dev-101", DriverType: "modbus_rtu"}

	recovered, err := executor.recoverMissingDriverBinding(dev)
	if err != nil {
		t.Fatalf("recoverMissingDriverBinding failed: %v", err)
	}
	if recovered == nil {
		t.Fatalf("expected recovered driver, got nil")
	}
	if recovered.ID != driverID {
		t.Fatalf("recovered driver id=%d, want %d", recovered.ID, driverID)
	}
	if dev.DriverID == nil || *dev.DriverID != driverID {
		t.Fatalf("device driver id not updated in memory")
	}

	var boundID int64
	if err := database.ParamDB.QueryRow(`SELECT driver_id FROM devices WHERE id = ?`, dev.ID).Scan(&boundID); err != nil {
		t.Fatalf("query updated driver_id failed: %v", err)
	}
	if boundID != driverID {
		t.Fatalf("device driver_id in db=%d, want %d", boundID, driverID)
	}
}

func TestRecoverMissingDriverBindingNoMatch(t *testing.T) {
	tmpDir := t.TempDir()
	paramPath := filepath.Join(tmpDir, "param.db")
	originalParamDB := database.ParamDB
	t.Cleanup(func() {
		if database.ParamDB != nil {
			_ = database.ParamDB.Close()
		}
		database.ParamDB = originalParamDB
	})

	if err := database.InitParamDBWithPath(paramPath); err != nil {
		t.Fatalf("InitParamDBWithPath failed: %v", err)
	}
	if _, err := database.ParamDB.Exec(`CREATE TABLE drivers (
		id INTEGER PRIMARY KEY,
		name TEXT UNIQUE NOT NULL,
		file_path TEXT NOT NULL,
		description TEXT,
		version TEXT,
		config_schema TEXT,
		enabled INTEGER DEFAULT 1,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`); err != nil {
		t.Fatalf("create drivers table failed: %v", err)
	}
	if _, err := database.ParamDB.Exec(`CREATE TABLE devices (
		id INTEGER PRIMARY KEY,
		name TEXT,
		driver_id INTEGER,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`); err != nil {
		t.Fatalf("create devices table failed: %v", err)
	}

	executor := NewDriverExecutor(NewDriverManager())
	dev := &models.Device{ID: 102, Name: "dev-102", DriverType: "modbus_tcp"}

	_, err := executor.recoverMissingDriverBinding(dev)
	if err == nil {
		t.Fatalf("expected recoverMissingDriverBinding to fail without matching driver")
	}
	if !strings.Contains(err.Error(), "no recoverable driver") {
		t.Fatalf("unexpected error: %v", err)
	}
}
