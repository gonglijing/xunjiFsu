package handlers

import (
	"database/sql"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/gonglijing/xunjiFsu/internal/database"
	driverpkg "github.com/gonglijing/xunjiFsu/internal/driver"
	"github.com/gonglijing/xunjiFsu/internal/models"
)

func normalizeDriverInput(h *Handler, driver *models.Driver) error {
	if driver == nil {
		return sql.ErrNoRows
	}
	driver.Name = strings.TrimSpace(driver.Name)
	if driver.Name == "" {
		return sql.ErrNoRows
	}
	driver.FilePath = strings.TrimSpace(driver.FilePath)
	if driver.FilePath == "" {
		driver.FilePath = h.driverFilePath(driver.Name, "")
	}
	if driver.Enabled != 1 {
		driver.Enabled = 0
	}
	return nil
}

func enrichDriverModel(h *Handler, driver *models.Driver) {
	if driver == nil {
		return
	}
	path := h.driverFilePath(driver.Name, driver.FilePath)
	if info, err := os.Stat(path); err == nil {
		driver.Size = info.Size()
		driver.Filename = filepath.Base(path)
	}
	if runtime, err := h.driverManager.GetRuntime(driver.ID); err == nil && runtime != nil {
		driver.Loaded = runtime.Loaded
		driver.ResourceID = runtime.ResourceID
		driver.LastActive = runtime.LastActive
		driver.Exports = runtime.ExportedFunctions
	}
}

func loadAndSyncDriverVersion(h *Handler, driver *models.Driver) {
	if driver == nil {
		return
	}
	wasmPath := h.driverFilePath(driver.Name, driver.FilePath)
	wasmData, err := os.ReadFile(wasmPath)
	if err != nil {
		return
	}
	if version, err := driverpkg.ExtractDriverVersion(wasmData); err == nil && version != "" {
		driver.Version = version
		if driver.ID > 0 {
			_ = database.UpdateDriverVersion(driver.ID, version)
		}
	}
}

func (h *Handler) runtimeResponse(id int64) interface{} {
	runtime, err := h.driverManager.GetRuntime(id)
	if err != nil {
		return map[string]interface{}{"id": id, "loaded": false}
	}
	return runtime
}

func (h *Handler) driverFilePath(driverName, filePath string) string {
	if filePath != "" {
		return filePath
	}
	dir := h.resolvedDriversDir()
	return filepath.Join(dir, driverName+".wasm")
}

func (h *Handler) resolvedDriversDir() string {
	if strings.TrimSpace(h.driversDir) == "" {
		return "drivers"
	}
	return h.driversDir
}

func isWasmFileName(filename string) bool {
	return strings.HasSuffix(strings.ToLower(strings.TrimSpace(filename)), ".wasm")
}

func saveDriverUploadFile(driversDir, filename string, source io.Reader) (string, error) {
	if err := os.MkdirAll(driversDir, 0755); err != nil {
		return "", err
	}

	destPath := filepath.Join(driversDir, filename)
	destFile, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return "", err
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, source); err != nil {
		return "", err
	}

	return destPath, nil
}

func listDriverWasmFiles(driversDir string) ([]map[string]interface{}, error) {
	entries, err := os.ReadDir(driversDir)
	if err != nil {
		return nil, err
	}

	files := make([]map[string]interface{}, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !isWasmFileName(entry.Name()) {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		files = append(files, map[string]interface{}{
			"name":     entry.Name(),
			"size":     info.Size(),
			"modified": info.ModTime().Format("2006-01-02 15:04:05"),
		})
	}

	return files, nil
}
