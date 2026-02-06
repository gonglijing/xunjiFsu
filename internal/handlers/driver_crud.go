package handlers

import (
	"net/http"
	"os"

	"github.com/gonglijing/xunjiFsu/internal/database"
	"github.com/gonglijing/xunjiFsu/internal/models"
)

// GetDrivers 从数据库列出驱动信息，附带文件大小（若存在）
func (h *Handler) GetDrivers(w http.ResponseWriter, r *http.Request) {
	drivers, err := database.GetAllDrivers()
	if err != nil {
		WriteServerError(w, err.Error())
		return
	}
	for _, d := range drivers {
		enrichDriverModel(h, d)
	}
	WriteSuccess(w, drivers)
}

// CreateDriver 创建驱动（手动录入元数据 + 已有文件）
func (h *Handler) CreateDriver(w http.ResponseWriter, r *http.Request) {
	var driver models.Driver
	if err := ParseRequest(r, &driver); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	if err := normalizeDriverInput(h, &driver); err != nil {
		WriteBadRequest(w, "driver name is required")
		return
	}

	if _, err := os.Stat(driver.FilePath); err != nil {
		WriteBadRequest(w, "driver wasm file not found")
		return
	}

	loadAndSyncDriverVersion(h, &driver)

	id, err := database.CreateDriver(&driver)
	if err != nil {
		WriteServerError(w, err.Error())
		return
	}

	driver.ID = id

	if driver.Enabled == 1 {
		if err := h.driverManager.LoadDriverFromModel(&driver, 0); err != nil {
			WriteServerError(w, "driver loaded failed: "+err.Error())
			return
		}
	}

	WriteCreated(w, driver)
}

// UpdateDriver 更新驱动
func (h *Handler) UpdateDriver(w http.ResponseWriter, r *http.Request) {
	id, err := ParseID(r)
	if err != nil {
		WriteBadRequest(w, "Invalid ID")
		return
	}

	var driver models.Driver
	if err := ParseRequest(r, &driver); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	driver.ID = id
	if err := normalizeDriverInput(h, &driver); err != nil {
		WriteBadRequest(w, "driver name is required")
		return
	}
	if _, err := os.Stat(driver.FilePath); err != nil {
		WriteBadRequest(w, "driver wasm file not found")
		return
	}
	loadAndSyncDriverVersion(h, &driver)
	if err := database.UpdateDriver(&driver); err != nil {
		WriteServerError(w, err.Error())
		return
	}

	if driver.Enabled == 1 {
		if err := h.driverManager.LoadDriverFromModel(&driver, 0); err != nil {
			WriteServerError(w, "driver reloaded failed: "+err.Error())
			return
		}
	} else {
		_ = h.driverManager.UnloadDriver(driver.ID)
	}

	WriteSuccess(w, driver)
}

// DeleteDriver 删除驱动（按ID，同时删除文件）
func (h *Handler) DeleteDriver(w http.ResponseWriter, r *http.Request) {
	id, err := ParseID(r)
	if err != nil {
		WriteBadRequest(w, "Invalid ID")
		return
	}
	drv, err := database.GetDriverByID(id)
	if err != nil {
		WriteNotFound(w, "Driver not found")
		return
	}
	_ = database.DeleteDriver(id)
	_ = h.driverManager.UnloadDriver(id)

	path := h.driverFilePath(drv.Name, drv.FilePath)
	_ = os.Remove(path)

	w.WriteHeader(http.StatusNoContent)
}
