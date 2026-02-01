package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

// APIResponse 统一 API 响应格式
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Message string      `json:"message,omitempty"`
}

// WriteJSON 统一 JSON 响应
func WriteJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// WriteSuccess 成功响应
func WriteSuccess(w http.ResponseWriter, data interface{}) {
	WriteJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    data,
	})
}

// WriteCreated 创建成功响应
func WriteCreated(w http.ResponseWriter, data interface{}) {
	WriteJSON(w, http.StatusCreated, APIResponse{
		Success: true,
		Data:    data,
	})
}

// WriteError 错误响应
func WriteError(w http.ResponseWriter, status int, message string) {
	WriteJSON(w, status, APIResponse{
		Success: false,
		Error:   message,
	})
}

// WriteBadRequest 400 错误
func WriteBadRequest(w http.ResponseWriter, message string) {
	WriteError(w, http.StatusBadRequest, message)
}

// WriteUnauthorized 401 错误
func WriteUnauthorized(w http.ResponseWriter, message string) {
	WriteError(w, http.StatusUnauthorized, message)
}

// WriteNotFound 404 错误
func WriteNotFound(w http.ResponseWriter, message string) {
	WriteError(w, http.StatusNotFound, message)
}

// WriteServerError 500 错误
func WriteServerError(w http.ResponseWriter, message string) {
	WriteError(w, http.StatusInternalServerError, message)
}

// ParseRequest 解析请求 JSON 或表单数据
func ParseRequest(r *http.Request, v interface{}) error {
	// 首先尝试解析 JSON
	contentType := r.Header.Get("Content-Type")
	if contentType == "application/json" {
		return json.NewDecoder(r.Body).Decode(v)
	}

	// 解析表单数据
	if err := r.ParseForm(); err != nil {
		return err
	}

	// 将表单数据转换为 JSON（尝试转换数值类型，但保留可能为纯数字的字符串字段）
	formData := make(map[string]interface{})
	for key, values := range r.Form {
		if len(values) > 0 {
			value := values[0]
			// 对于设备地址等应该保持为字符串的字段，直接使用字符串
			if key == "device_address" || key == "name" || key == "description" || 
			   key == "serial_port" || key == "ip_address" || key == "protocol" || 
			   key == "parity" || key == "interface_type" {
				formData[key] = value
				continue
			}
			// 尝试转换为整数
			if intVal, err := strconv.ParseInt(value, 10, 64); err == nil {
				formData[key] = intVal
				continue
			}
			// 尝试转换为浮点数
			if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
				formData[key] = floatVal
				continue
			}
			// 检查是否是布尔值
			if value == "1" || value == "true" {
				formData[key] = true
				continue
			}
			if value == "0" || value == "false" {
				formData[key] = false
				continue
			}
			formData[key] = value
		}
	}

	// 为空字段设置默认值
	if _, exists := formData["parity"]; !exists {
		formData["parity"] = "N"
	}
	if _, exists := formData["interface_type"]; !exists {
		formData["interface_type"] = "network"
	}
	if _, exists := formData["protocol"]; !exists {
		formData["protocol"] = "tcp"
	}
	if _, exists := formData["baud_rate"]; !exists {
		formData["baud_rate"] = 9600
	}
	if _, exists := formData["data_bits"]; !exists {
		formData["data_bits"] = 8
	}
	if _, exists := formData["stop_bits"]; !exists {
		formData["stop_bits"] = 1
	}

	jsonData, err := json.Marshal(formData)
	if err != nil {
		return err
	}
	return json.Unmarshal(jsonData, v)
}

// ParseID 从 URL 参数解析 ID
func ParseID(r *http.Request) (int64, error) {
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		return 0, err
	}
	return id, nil
}
