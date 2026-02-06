package handlers

import (
	"database/sql"
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
	dir := h.driversDir
	if dir == "" {
		dir = "drivers"
	}
	return filepath.Join(dir, driverName+".wasm")
}
