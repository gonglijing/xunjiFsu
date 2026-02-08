package handlers

import (
	"encoding/json"
	"mime"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

var stringOnlyFormFields = map[string]struct{}{
	"device_address": {},
	"name":           {},
	"description":    {},
	"serial_port":    {},
	"ip_address":     {},
	"protocol":       {},
	"parity":         {},
	"interface_type": {},
}

var formDefaultValues = map[string]interface{}{
	"parity":         "N",
	"interface_type": "network",
	"protocol":       "tcp",
	"baud_rate":      9600,
	"data_bits":      8,
	"stop_bits":      1,
}

type APIErrorDef struct {
	Code    string
	Message string
}

const (
	defaultBadRequestCode   = "E_BAD_REQUEST"
	defaultUnauthorizedCode = "E_UNAUTHORIZED"
	defaultNotFoundCode     = "E_NOT_FOUND"
	defaultServerErrorCode  = "E_SERVER_ERROR"
)

// APIResponse 统一 API 响应格式
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Code    string      `json:"code,omitempty"`
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

// WriteDeleted 删除成功响应
func WriteDeleted(w http.ResponseWriter) {
	WriteSuccess(w, nil)
}

// WriteError 错误响应
func WriteError(w http.ResponseWriter, status int, message string) {
	WriteErrorCode(w, status, "", message)
}

// WriteErrorCode 带错误码的错误响应
func WriteErrorCode(w http.ResponseWriter, status int, code, message string) {
	WriteJSON(w, status, APIResponse{
		Success: false,
		Error:   message,
		Code:    code,
		Message: message,
	})
}

func WriteErrorDef(w http.ResponseWriter, status int, def APIErrorDef) {
	WriteErrorCode(w, status, def.Code, def.Message)
}

// WriteBadRequest 400 错误
func WriteBadRequest(w http.ResponseWriter, message string) {
	WriteBadRequestCode(w, defaultBadRequestCode, message)
}

func WriteBadRequestDef(w http.ResponseWriter, def APIErrorDef) {
	WriteErrorDef(w, http.StatusBadRequest, def)
}

func WriteBadRequestCode(w http.ResponseWriter, code, message string) {
	WriteErrorCode(w, http.StatusBadRequest, code, message)
}

// WriteUnauthorized 401 错误
func WriteUnauthorized(w http.ResponseWriter, message string) {
	WriteErrorCode(w, http.StatusUnauthorized, defaultUnauthorizedCode, message)
}

// WriteNotFound 404 错误
func WriteNotFound(w http.ResponseWriter, message string) {
	WriteErrorCode(w, http.StatusNotFound, defaultNotFoundCode, message)
}

func WriteNotFoundDef(w http.ResponseWriter, def APIErrorDef) {
	WriteErrorDef(w, http.StatusNotFound, def)
}

// WriteServerError 500 错误
func WriteServerError(w http.ResponseWriter, message string) {
	WriteServerErrorCode(w, defaultServerErrorCode, message)
}

func WriteServerErrorDef(w http.ResponseWriter, def APIErrorDef) {
	WriteErrorDef(w, http.StatusInternalServerError, def)
}

func WriteServerErrorCode(w http.ResponseWriter, code, message string) {
	WriteErrorCode(w, http.StatusInternalServerError, code, message)
}

// ParseRequest 解析请求 JSON 或表单数据
func ParseRequest(r *http.Request, v interface{}) error {
	contentType := r.Header.Get("Content-Type")
	if mediaType, _, err := mime.ParseMediaType(contentType); err == nil && mediaType == "application/json" {
		return json.NewDecoder(r.Body).Decode(v)
	}

	if err := r.ParseForm(); err != nil {
		return err
	}

	formData := make(map[string]interface{})
	for key, values := range r.Form {
		if len(values) > 0 {
			formData[key] = parseFormValue(key, values[0])
		}
	}

	applyFormDefaults(formData)

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

func parseIDOrWriteBadRequest(w http.ResponseWriter, r *http.Request, message string) (int64, bool) {
	id, err := ParseID(r)
	if err != nil {
		WriteBadRequest(w, message)
		return 0, false
	}
	return id, true
}

func parseIDOrWriteBadRequestDefault(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := ParseID(r)
	if err != nil {
		WriteBadRequestDef(w, apiErrInvalidID)
		return 0, false
	}
	return id, true
}

func parseRequestOrWriteBadRequest(w http.ResponseWriter, r *http.Request, payload interface{}, message string) bool {
	if err := ParseRequest(r, payload); err != nil {
		WriteBadRequest(w, message)
		return false
	}
	return true
}

func parseRequestOrWriteBadRequestDefault(w http.ResponseWriter, r *http.Request, payload interface{}) bool {
	if err := ParseRequest(r, payload); err != nil {
		WriteBadRequestDef(w, apiErrInvalidRequestBody)
		return false
	}
	return true
}

func parseFormValue(key, value string) interface{} {
	if isStringOnlyFormField(key) {
		return value
	}

	if intVal, err := strconv.ParseInt(value, 10, 64); err == nil {
		return intVal
	}
	if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
		return floatVal
	}
	if value == "1" || value == "true" {
		return true
	}
	if value == "0" || value == "false" {
		return false
	}

	return value
}

func isStringOnlyFormField(key string) bool {
	_, ok := stringOnlyFormFields[key]
	return ok
}

func applyFormDefaults(formData map[string]interface{}) {
	for key, defaultValue := range formDefaultValues {
		if _, exists := formData[key]; !exists {
			formData[key] = defaultValue
		}
	}
}
