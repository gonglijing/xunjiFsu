package handlers

import (
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gonglijing/xunjiFsu/internal/database"
	driverpkg "github.com/gonglijing/xunjiFsu/internal/driver"
)

// UploadDriverFile 上传驱动文件
func (h *Handler) UploadDriverFile(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		WriteBadRequest(w, "Failed to parse form: "+err.Error())
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		WriteBadRequest(w, "Failed to get file: "+err.Error())
		return
	}
	defer file.Close()

	if !isWasmFileName(header.Filename) {
		WriteBadRequest(w, errOnlyWasmFilesAllowedMessage)
		return
	}

	driversDir := h.resolvedDriversDir()
	destPath, err := saveDriverUploadFile(driversDir, header.Filename, file)
	if err != nil {
		if os.IsPermission(err) {
			writeServerErrorWithLog(w, apiErrSaveDriverFileFailed, err)
			return
		}
		if os.IsNotExist(err) {
			writeServerErrorWithLog(w, apiErrCreateDriversDirFailed, err)
			return
		}
		writeServerErrorWithLog(w, apiErrWriteDriverFileFailed, err)
		return
	}

	_ = database.UpsertDriverFile(strings.TrimSuffix(header.Filename, ".wasm"), destPath)
	driverName := strings.TrimSuffix(header.Filename, ".wasm")
	version := ""
	if wasmData, err := os.ReadFile(destPath); err == nil {
		if v, err := driverpkg.ExtractDriverVersion(wasmData); err == nil && v != "" {
			_ = database.UpdateDriverVersionByName(driverName, v)
			version = v
		}
	}

	WriteSuccess(w, map[string]interface{}{
		"filename": header.Filename,
		"path":     destPath,
		"size":     header.Size,
		"version":  version,
	})
}

// DownloadDriver 下载驱动文件
func (h *Handler) DownloadDriver(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDOrWriteBadRequestDefault(w, r)
	if !ok {
		return
	}
	driver, err := database.GetDriverByID(id)
	if err != nil {
		WriteNotFoundDef(w, apiErrDriverNotFound)
		return
	}
	filePath := h.driverFilePath(driver.Name, driver.FilePath)

	file, err := os.Open(filePath)
	if err != nil {
		WriteNotFound(w, "File not found: "+err.Error())
		return
	}
	defer file.Close()

	fileInfo, _ := file.Stat()

	w.Header().Set("Content-Description", "File Transfer")
	w.Header().Set("Content-Disposition", "attachment; filename="+filepath.Base(filePath))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", strconv.FormatInt(fileInfo.Size(), 10))

	http.ServeFile(w, r, filePath)
}

// ListDriverFiles 列出驱动目录中的文件
func (h *Handler) ListDriverFiles(w http.ResponseWriter, r *http.Request) {
	driversDir := h.resolvedDriversDir()
	files, err := listDriverWasmFiles(driversDir)
	if err != nil {
		if os.IsNotExist(err) {
			WriteSuccess(w, []interface{}{})
			return
		}
		writeServerErrorWithLog(w, apiErrListDriverFilesFailed, err)
		return
	}

	WriteSuccess(w, files)
}
