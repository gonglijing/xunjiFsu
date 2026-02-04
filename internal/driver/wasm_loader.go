package driver

import (
	"fmt"
	"os"
	"path/filepath"
)

// readWasmFile 尝试读取 wasm 文件，支持相对路径的多种基准
func readWasmFile(path string) ([]byte, error) {
	if path == "" {
		return nil, fmt.Errorf("empty wasm path")
	}
	// 1) 绝对路径或直接相对当前工作目录
	if data, err := os.ReadFile(path); err == nil {
		return data, nil
	}
	// 2) 以当前工作目录拼接
	if cwd, err := os.Getwd(); err == nil {
		if data, err := os.ReadFile(filepath.Join(cwd, path)); err == nil {
			return data, nil
		}
	}
	// 3) 以可执行文件目录为基准
	if exePath, err := os.Executable(); err == nil {
		base := filepath.Dir(exePath)
		if data, err := os.ReadFile(filepath.Join(base, path)); err == nil {
			return data, nil
		}
		// 4) 上一级（常见于 go run 临时目录）
		if data, err := os.ReadFile(filepath.Join(base, "..", path)); err == nil {
			return data, nil
		}
	}
	return nil, fmt.Errorf("cannot read wasm file: %s", path)
}
