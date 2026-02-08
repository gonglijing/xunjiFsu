package handlers

import (
	"database/sql"
	"net/http"
	"os"

	"github.com/gonglijing/xunjiFsu/internal/database"
	driverpkg "github.com/gonglijing/xunjiFsu/internal/driver"
)

// ReloadDriver 重载驱动
func (h *Handler) ReloadDriver(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDOrWriteBadRequestDefault(w, r)
	if !ok {
		return
	}

	drv, err := database.GetDriverByID(id)
	if err != nil {
		WriteNotFoundDef(w, apiErrDriverNotFound)
		return
	}

	path := h.driverFilePath(drv.Name, drv.FilePath)
	if _, err := os.Stat(path); err != nil {
		WriteBadRequestDef(w, apiErrDriverWasmFileNotFound)
		return
	}

	if err := h.driverManager.LoadDriverFromModel(drv, 0); err != nil {
		WriteServerError(w, "reload driver failed: "+err.Error())
		return
	}

	loadAndSyncDriverVersion(h, drv)
	WriteSuccess(w, h.runtimeResponse(drv.ID))
}

// GetDriverRuntime 获取驱动运行态
func (h *Handler) GetDriverRuntime(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDOrWriteBadRequestDefault(w, r)
	if !ok {
		return
	}

	_, err := database.GetDriverByID(id)
	if err != nil {
		if err == sql.ErrNoRows {
			WriteNotFoundDef(w, apiErrDriverNotFound)
			return
		}
		WriteServerError(w, err.Error())
		return
	}

	runtime, err := h.driverManager.GetRuntime(id)
	if err != nil {
		if err == driverpkg.ErrDriverNotLoaded {
			WriteSuccess(w, map[string]interface{}{"id": id, "loaded": false})
			return
		}
		WriteServerError(w, err.Error())
		return
	}

	WriteSuccess(w, runtime)
}

// GetDriverRuntimeList 获取所有已加载驱动的运行态
func (h *Handler) GetDriverRuntimeList(w http.ResponseWriter, r *http.Request) {
	runtimes := h.driverManager.ListRuntimes()
	WriteSuccess(w, runtimes)
}
