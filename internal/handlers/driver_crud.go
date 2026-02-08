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
		writeServerErrorWithLog(w, apiErrListDriversFailed, err)
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
	if !parseRequestOrWriteBadRequestDefault(w, r, &driver) {
		return
	}

	if err := normalizeDriverInput(h, &driver); err != nil {
		WriteBadRequestDef(w, apiErrDriverNameRequired)
		return
	}

	if _, err := os.Stat(driver.FilePath); err != nil {
		WriteBadRequestDef(w, apiErrDriverWasmFileNotFound)
		return
	}

	loadAndSyncDriverVersion(h, &driver)

	id, err := database.CreateDriver(&driver)
	if err != nil {
		writeServerErrorWithLog(w, apiErrCreateDriverFailed, err)
		return
	}

	driver.ID = id

	if driver.Enabled == 1 {
		if err := h.driverManager.LoadDriverFromModel(&driver, 0); err != nil {
			writeServerErrorWithLog(w, apiErrLoadDriverFailed, err)
			return
		}
	}

	WriteCreated(w, driver)
}

// UpdateDriver 更新驱动
func (h *Handler) UpdateDriver(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDOrWriteBadRequestDefault(w, r)
	if !ok {
		return
	}

	var driver models.Driver
	if !parseRequestOrWriteBadRequestDefault(w, r, &driver) {
		return
	}

	driver.ID = id
	if err := normalizeDriverInput(h, &driver); err != nil {
		WriteBadRequestDef(w, apiErrDriverNameRequired)
		return
	}
	if _, err := os.Stat(driver.FilePath); err != nil {
		WriteBadRequestDef(w, apiErrDriverWasmFileNotFound)
		return
	}
	loadAndSyncDriverVersion(h, &driver)
	if err := database.UpdateDriver(&driver); err != nil {
		writeServerErrorWithLog(w, apiErrUpdateDriverFailed, err)
		return
	}

	if driver.Enabled == 1 {
		if err := h.driverManager.LoadDriverFromModel(&driver, 0); err != nil {
			writeServerErrorWithLog(w, apiErrReloadDriverFailed, err)
			return
		}
	} else {
		_ = h.driverManager.UnloadDriver(driver.ID)
	}

	WriteSuccess(w, driver)
}

// DeleteDriver 删除驱动（按ID，同时删除文件）
func (h *Handler) DeleteDriver(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDOrWriteBadRequestDefault(w, r)
	if !ok {
		return
	}
	drv, err := database.GetDriverByID(id)
	if err != nil {
		WriteNotFoundDef(w, apiErrDriverNotFound)
		return
	}
	_ = database.DeleteDriver(id)
	_ = h.driverManager.UnloadDriver(id)

	path := h.driverFilePath(drv.Name, drv.FilePath)
	_ = os.Remove(path)

	WriteDeleted(w)
}
