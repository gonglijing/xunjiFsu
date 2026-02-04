package driver

import (
	"context"
	"fmt"

	extism "github.com/extism/go-sdk"
	"github.com/gonglijing/xunjiFsu/internal/logger"
)

func newWasmPlugin(driverName string, wasmData []byte, hostFuncs []extism.HostFunction, config map[string]string) (*extism.Plugin, error) {
	manifest := extism.Manifest{
		Wasm: []extism.Wasm{
			&extism.WasmData{
				Name: driverName,
				Data: wasmData,
			},
		},
	}

	plugin, err := extism.NewPlugin(context.Background(), manifest, extism.PluginConfig{
		EnableWasi: true,
	}, hostFuncs)
	if err != nil {
		return nil, err
	}

	plugin.SetLogger(func(level extism.LogLevel, message string) {
		switch level {
		case extism.LogLevelError:
			logger.Error("Driver log", fmt.Errorf(message), "driver", driverName)
		case extism.LogLevelWarn:
			logger.Warn("Driver log", "driver", driverName, "message", message)
		default:
			logger.Info("Driver log", "driver", driverName, "level", level.String(), "message", message)
		}
	})

	if config != nil {
		plugin.Config = config
	}

	return plugin, nil
}
