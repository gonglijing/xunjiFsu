package handlers

import (
	"io"
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

	if !strings.HasSuffix(strings.ToLower(header.Filename), ".wasm") {
		WriteBadRequest(w, "Only .wasm files are allowed")
		return
	}

	driversDir := h.driversDir
	if driversDir == "" {
		driversDir = "drivers"
	}
	if err := os.MkdirAll(driversDir, 0755); err != nil {
		WriteServerError(w, "Failed to create drivers directory: "+err.Error())
		return
	}

	destPath := filepath.Join(driversDir, header.Filename)
	destFile, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		WriteServerError(w, "Failed to save file: "+err.Error())
		return
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, file); err != nil {
		WriteServerError(w, "Failed to write file: "+err.Error())
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
	id, err := ParseID(r)
	if err != nil {
		WriteBadRequest(w, "Invalid ID")
		return
	}
	driver, err := database.GetDriverByID(id)
	if err != nil {
		WriteNotFound(w, "Driver not found")
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
	driversDir := h.driversDir
	if driversDir == "" {
		driversDir = "drivers"
	}

	entries, err := os.ReadDir(driversDir)
	if err != nil {
		if os.IsNotExist(err) {
			WriteSuccess(w, []interface{}{})
			return
		}
		WriteServerError(w, "Failed to read drivers directory: "+err.Error())
		return
	}

	var files []map[string]interface{}
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(strings.ToLower(entry.Name()), ".wasm") {
			info, _ := entry.Info()
			files = append(files, map[string]interface{}{
				"name":     entry.Name(),
				"size":     info.Size(),
				"modified": info.ModTime().Format("2006-01-02 15:04:05"),
			})
		}
	}

	WriteSuccess(w, files)
}
