package handlers

import (
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gonglijing/xunjiFsu/internal/database"
	driverpkg "github.com/gonglijing/xunjiFsu/internal/driver"
	"github.com/gonglijing/xunjiFsu/internal/models"
)

// ==================== 驱动管理 ====================

// GetDrivers 从数据库列出驱动信息，附带文件大小（若存在）
func (h *Handler) GetDrivers(w http.ResponseWriter, r *http.Request) {
	drivers, err := database.GetAllDrivers()
	if err != nil {
		WriteServerError(w, err.Error())
		return
	}
	for _, d := range drivers {
		path := h.driverFilePath(d.Name, d.FilePath)
		if info, err := os.Stat(path); err == nil {
			d.Size = info.Size()
			d.Filename = filepath.Base(path)
		}
		if runtime, err := h.driverManager.GetRuntime(d.ID); err == nil && runtime != nil {
			d.Loaded = runtime.Loaded
			d.ResourceID = runtime.ResourceID
			d.LastActive = runtime.LastActive
			d.Exports = runtime.ExportedFunctions
		}
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

	if driver.FilePath == "" {
		driver.FilePath = h.driverFilePath(driver.Name, "")
	}

	if wasmData, err := os.ReadFile(h.driverFilePath(driver.Name, driver.FilePath)); err == nil {
		if version, err := driverpkg.ExtractDriverVersion(wasmData); err == nil && version != "" {
			driver.Version = version
		}
	}

	id, err := database.CreateDriver(&driver)
	if err != nil {
		WriteServerError(w, err.Error())
		return
	}

	driver.ID = id

	// 尝试加载驱动到内存
	if wasmData, err := os.ReadFile(driver.FilePath); err == nil {
		var cfg struct {
			ResourceID int64 `json:"resource_id"`
		}
		resourceID := int64(0)
		if driver.ConfigSchema != "" {
			if err := json.Unmarshal([]byte(driver.ConfigSchema), &cfg); err == nil {
				resourceID = cfg.ResourceID
			}
		}
		_ = h.driverManager.LoadDriver(&driver, wasmData, resourceID)
	}

	WriteCreated(w, driver)
}

// readDriverFile 读取驱动文件
func readDriverFile(filePath, driverName string) ([]byte, error) {
	if filePath == "" {
		filePath = filepath.Join("drivers", driverName+".wasm")
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	return data, nil
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
	if strings.TrimSpace(driver.Name) == "" {
		WriteBadRequest(w, "driver name is required")
		return
	}
	if strings.TrimSpace(driver.FilePath) == "" {
		driver.FilePath = h.driverFilePath(driver.Name, "")
	}
	if _, err := os.Stat(driver.FilePath); err != nil {
		WriteBadRequest(w, "driver wasm file not found")
		return
	}
	if err := database.UpdateDriver(&driver); err != nil {
		WriteServerError(w, err.Error())
		return
	}

	if wasmData, err := os.ReadFile(driver.FilePath); err == nil {
		if err := h.driverManager.ReloadDriver(&driver, wasmData, 0); err != nil {
			WriteServerError(w, "driver reloaded failed: "+err.Error())
			return
		}
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

	// 删除文件（忽略错误）
	path := drv.FilePath
	path = h.driverFilePath(drv.Name, path)
	_ = os.Remove(path)

	w.WriteHeader(http.StatusNoContent)
}

// ReloadDriver 重载驱动
func (h *Handler) ReloadDriver(w http.ResponseWriter, r *http.Request) {
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

	path := h.driverFilePath(drv.Name, drv.FilePath)
	wasmData, err := os.ReadFile(path)
	if err != nil {
		WriteBadRequest(w, "driver wasm file not found")
		return
	}

	if err := h.driverManager.ReloadDriver(drv, wasmData, 0); err != nil {
		WriteServerError(w, "reload driver failed: "+err.Error())
		return
	}

	if version, err := driverpkg.ExtractDriverVersion(wasmData); err == nil && version != "" {
		_ = database.UpdateDriverVersion(drv.ID, version)
		drv.Version = version
	}

	runtime, err := h.driverManager.GetRuntime(drv.ID)
	if err != nil {
		WriteSuccess(w, map[string]interface{}{"id": drv.ID, "loaded": true})
		return
	}
	WriteSuccess(w, runtime)
}

// GetDriverRuntime 获取驱动运行态
func (h *Handler) GetDriverRuntime(w http.ResponseWriter, r *http.Request) {
	id, err := ParseID(r)
	if err != nil {
		WriteBadRequest(w, "Invalid ID")
		return
	}

	_, err = database.GetDriverByID(id)
	if err != nil {
		if err == sql.ErrNoRows {
			WriteNotFound(w, "Driver not found")
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

// UploadDriverFile 上传驱动文件
func (h *Handler) UploadDriverFile(w http.ResponseWriter, r *http.Request) {
	// 解析 multipart 表单
	if err := r.ParseMultipartForm(10 << 20); err != nil { // 10MB max
		WriteBadRequest(w, "Failed to parse form: "+err.Error())
		return
	}

	// 获取文件
	file, header, err := r.FormFile("file")
	if err != nil {
		WriteBadRequest(w, "Failed to get file: "+err.Error())
		return
	}
	defer file.Close()

	// 验证文件扩展名
	if !strings.HasSuffix(strings.ToLower(header.Filename), ".wasm") {
		WriteBadRequest(w, "Only .wasm files are allowed")
		return
	}

	// 创建 drivers 目录
	driversDir := h.driversDir
	if driversDir == "" {
		driversDir = "drivers"
	}
	if err := os.MkdirAll(driversDir, 0755); err != nil {
		WriteServerError(w, "Failed to create drivers directory: "+err.Error())
		return
	}

	// 保存文件
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

	// 写入/忽略驱动元数据
	_ = database.UpsertDriverFile(strings.TrimSuffix(header.Filename, ".wasm"), destPath)
	driverName := strings.TrimSuffix(header.Filename, ".wasm")
	version := ""
	if wasmData, err := os.ReadFile(destPath); err == nil {
		if v, err := driverpkg.ExtractDriverVersion(wasmData); err == nil && v != "" {
			_ = database.UpdateDriverVersionByName(driverName, v)
			version = v
		}
	}

	// 返回成功信息
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
	filePath := driver.FilePath
	filePath = h.driverFilePath(driver.Name, filePath)

	// 打开文件
	file, err := os.Open(filePath)
	if err != nil {
		WriteNotFound(w, "File not found: "+err.Error())
		return
	}
	defer file.Close()

	// 获取文件信息
	fileInfo, _ := file.Stat()

	// 设置响应头
	w.Header().Set("Content-Description", "File Transfer")
	w.Header().Set("Content-Disposition", "attachment; filename="+filepath.Base(filePath))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", strconv.FormatInt(fileInfo.Size(), 10))

	// 发送文件
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
