package driver

import (
	"context"
	"encoding/json"
	"fmt"

	extism "github.com/extism/go-sdk"
)

type driverVersionPayload struct {
	Success bool                   `json:"success"`
	Version string                 `json:"version"`
	Data    map[string]interface{} `json:"data"`
	Error   string                 `json:"error"`
}

// ExtractDriverVersion reads the internal driver version from a wasm binary if exported.
func ExtractDriverVersion(wasmData []byte) (string, error) {
	if len(wasmData) == 0 {
		return "", fmt.Errorf("empty wasm data")
	}
	plugin, err := newWasmPlugin("driver_version", wasmData, nil, nil)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = plugin.Close(context.Background())
	}()

	return extractVersionFromPlugin(plugin)
}

func extractVersionFromPlugin(plugin *extism.Plugin) (string, error) {
	if plugin == nil {
		return "", fmt.Errorf("nil plugin")
	}
	if !plugin.FunctionExists("version") {
		return "", nil
	}

	rc, output, err := plugin.CallWithContext(context.Background(), "version", []byte("{}"))
	if err != nil {
		return "", err
	}
	if rc != 0 {
		// keep parsing output but note non-zero rc
	}
	if len(output) == 0 {
		if alt, err2 := plugin.GetOutput(); err2 == nil && len(alt) > 0 {
			output = alt
		}
	}
	if len(output) == 0 {
		if msg := plugin.GetError(); msg != "" {
			return "", fmt.Errorf("driver version error: %s", msg)
		}
		return "", fmt.Errorf("driver version output empty")
	}

	return parseDriverVersionOutput(output)
}

func parseDriverVersionOutput(output []byte) (string, error) {
	var payload driverVersionPayload
	if err := json.Unmarshal(output, &payload); err != nil {
		return "", err
	}
	if !payload.Success {
		if payload.Error != "" {
			return "", fmt.Errorf("%s", payload.Error)
		}
		return "", fmt.Errorf("version response not success")
	}
	if payload.Version != "" {
		return payload.Version, nil
	}
	if payload.Data != nil {
		if v, ok := payload.Data["version"]; ok {
			if s, ok := v.(string); ok {
				return s, nil
			}
			return fmt.Sprint(v), nil
		}
	}
	return "", nil
}

// GetDriverVersion fetches version from a loaded driver plugin if exported.
func (m *DriverManager) GetDriverVersion(id int64) (string, error) {
	m.mu.RLock()
	driver, exists := m.drivers[id]
	m.mu.RUnlock()
	if !exists {
		return "", ErrDriverNotFound
	}

	return extractVersionFromPlugin(driver.plugin)
}
