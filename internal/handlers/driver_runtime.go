package handlers

import (
	"database/sql"
	"net/http"
	"os"

	"github.com/gonglijing/xunjiFsu/internal/database"
	driverpkg "github.com/gonglijing/xunjiFsu/internal/driver"
	"github.com/gonglijing/xunjiFsu/internal/models"
)

// ReloadDriver 重载驱动
func (h *Handler) ReloadDriver(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDOrWriteBadRequestDefault(w, r)
	if !ok {
		return
	}

	driverModel, ok := loadDriverByIDOrWriteNotFound(w, id)
	if !ok {
		return
	}
	if !h.ensureDriverFileExists(driverModel) {
		WriteBadRequestDef(w, apiErrDriverWasmFileNotFound)
		return
	}

	if err := h.driverManager.LoadDriverFromModel(driverModel, 0); err != nil {
		writeServerErrorWithLog(w, apiErrReloadDriverFailed, err)
		return
	}

	loadAndSyncDriverVersion(h, driverModel)
	WriteSuccess(w, h.runtimeResponse(driverModel.ID))
}

// GetDriverRuntime 获取驱动运行态
func (h *Handler) GetDriverRuntime(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDOrWriteBadRequestDefault(w, r)
	if !ok {
		return
	}

	if _, ok := loadDriverByIDOrWriteRuntimeError(w, id); !ok {
		return
	}

	runtime, err := h.driverManager.GetRuntime(id)
	if err != nil {
		if err == driverpkg.ErrDriverNotLoaded {
			WriteSuccess(w, buildUnloadedDriverRuntime(id))
			return
		}
		writeServerErrorWithLog(w, apiErrGetDriverRuntimeFailed, err)
		return
	}

	WriteSuccess(w, runtime)
}

// GetDriverRuntimeList 获取所有已加载驱动的运行态
func (h *Handler) GetDriverRuntimeList(w http.ResponseWriter, r *http.Request) {
	runtimes := h.driverManager.ListRuntimes()
	WriteSuccess(w, runtimes)
}

func loadDriverByIDOrWriteNotFound(w http.ResponseWriter, id int64) (*models.Driver, bool) {
	driverModel, err := database.GetDriverByID(id)
	if err != nil {
		WriteNotFoundDef(w, apiErrDriverNotFound)
		return nil, false
	}
	return driverModel, true
}

func loadDriverByIDOrWriteRuntimeError(w http.ResponseWriter, id int64) (*models.Driver, bool) {
	driverModel, err := database.GetDriverByID(id)
	if err != nil {
		if err == sql.ErrNoRows {
			WriteNotFoundDef(w, apiErrDriverNotFound)
			return nil, false
		}
		writeServerErrorWithLog(w, apiErrGetDriverRuntimeFailed, err)
		return nil, false
	}
	return driverModel, true
}

func (h *Handler) ensureDriverFileExists(driverModel *models.Driver) bool {
	path := h.driverFilePath(driverModel.Name, driverModel.FilePath)
	if _, err := os.Stat(path); err != nil {
		return false
	}
	return true
}

func buildUnloadedDriverRuntime(id int64) map[string]interface{} {
	return map[string]interface{}{
		"id":     id,
		"loaded": false,
	}
}
