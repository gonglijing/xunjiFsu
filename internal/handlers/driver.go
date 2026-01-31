package handlers

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gonglijing/xunjiFsu/internal/database"
	"github.com/gonglijing/xunjiFsu/internal/models"
)

// ==================== 驱动管理 ====================

// GetDrivers 获取所有驱动
func (h *Handler) GetDrivers(w http.ResponseWriter, r *http.Request) {
	drivers, err := database.GetAllDrivers()
	if err != nil {
		WriteServerError(w, err.Error())
		return
	}
	WriteSuccess(w, drivers)
}

// CreateDriver 创建驱动
func (h *Handler) CreateDriver(w http.ResponseWriter, r *http.Request) {
	var driver models.Driver
	if err := ParseRequest(r, &driver); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	id, err := database.CreateDriver(&driver)
	if err != nil {
		WriteServerError(w, err.Error())
		return
	}

	driver.ID = id

	// 加载驱动
	if err := h.driverManager.LoadDriver(&driver); err != nil {
		WriteServerError(w, err.Error())
		return
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
	if err := database.UpdateDriver(&driver); err != nil {
		WriteServerError(w, err.Error())
		return
	}

	WriteSuccess(w, driver)
}

// DeleteDriver 删除驱动
func (h *Handler) DeleteDriver(w http.ResponseWriter, r *http.Request) {
	id, err := ParseID(r)
	if err != nil {
		WriteBadRequest(w, "Invalid ID")
		return
	}

	if err := database.DeleteDriver(id); err != nil {
		WriteServerError(w, err.Error())
		return
	}

	WriteSuccess(w, nil)
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
	driversDir := "drivers"
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

	// 返回成功信息
	WriteSuccess(w, map[string]interface{}{
		"filename": header.Filename,
		"path":     destPath,
		"size":     header.Size,
	})
}

// DownloadDriver 下载驱动文件
func (h *Handler) DownloadDriver(w http.ResponseWriter, r *http.Request) {
	id, err := ParseID(r)
	if err != nil {
		WriteBadRequest(w, "Invalid ID")
		return
	}

	// 获取驱动信息
	driver, err := database.GetDriverByID(id)
	if err != nil {
		WriteNotFound(w, "Driver not found")
		return
	}

	// 检查文件路径
	filePath := driver.FilePath
	if filePath == "" {
		filePath = filepath.Join("drivers", driver.Name+".wasm")
	}

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
	driversDir := "drivers"

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
