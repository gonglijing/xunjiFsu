package handlers

import (
	"encoding/json"
	"errors"
	"mime"
	"net/http"
	"strconv"
)

const jsonMediaType = "application/json"

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
	writeSuccessStatus(w, http.StatusOK, data)
}

// WriteCreated 创建成功响应
func WriteCreated(w http.ResponseWriter, data interface{}) {
	writeSuccessStatus(w, http.StatusCreated, data)
}

func writeSuccessStatus(w http.ResponseWriter, status int, data interface{}) {
	WriteJSON(w, status, APIResponse{
		Success: true,
		Data:    data,
	})
}

// WriteDeleted 删除成功响应
func WriteDeleted(w http.ResponseWriter) {
	WriteSuccess(w, nil)
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
func WriteServerErrorDef(w http.ResponseWriter, def APIErrorDef) {
	WriteErrorDef(w, http.StatusInternalServerError, def)
}

// ParseRequest 解析请求 JSON 或表单数据
func ParseRequest(r *http.Request, v interface{}) error {
	if isJSONRequest(r.Header.Get("Content-Type")) {
		return decodeJSONRequest(r, v)
	}

	return decodeFormRequest(r, v)
}

func isJSONRequest(contentType string) bool {
	mediaType, _, err := mime.ParseMediaType(contentType)
	return err == nil && mediaType == jsonMediaType
}

func decodeJSONRequest(r *http.Request, v interface{}) error {
	return json.NewDecoder(r.Body).Decode(v)
}

func decodeFormRequest(r *http.Request, v interface{}) error {
	formData, err := parseFormRequestData(r)
	if err != nil {
		return err
	}

	jsonData, err := json.Marshal(formData)
	if err != nil {
		return err
	}
	return json.Unmarshal(jsonData, v)
}

func parseFormRequestData(r *http.Request) (map[string]interface{}, error) {
	if err := r.ParseForm(); err != nil {
		return nil, err
	}

	formData := make(map[string]interface{}, len(r.Form))
	for key, values := range r.Form {
		value, ok := resolveFirstFormValue(values)
		if !ok {
			continue
		}
		formData[key] = parseFormFieldValue(key, value)
	}

	applyFormRequestDefaults(formData)
	return formData, nil
}

func resolveFirstFormValue(values []string) (string, bool) {
	if len(values) == 0 {
		return "", false
	}
	return values[0], true
}

// ParseID 从 URL 参数解析 ID
func ParseID(r *http.Request) (int64, error) {
	idText := r.PathValue("id")
	if idText == "" {
		return 0, errors.New("missing id path param")
	}
	id, err := strconv.ParseInt(idText, 10, 64)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func parseIDOrWriteBadRequestDefault(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := ParseID(r)
	if err != nil {
		WriteBadRequestDef(w, apiErrInvalidID)
		return 0, false
	}
	return id, true
}

func parseRequestOrWriteBadRequestDefault(w http.ResponseWriter, r *http.Request, payload interface{}) bool {
	if err := ParseRequest(r, payload); err != nil {
		WriteBadRequestDef(w, apiErrInvalidRequestBody)
		return false
	}
	return true
}

func parseFormFieldValue(key, value string) interface{} {
	if isStringOnlyFormField(key) {
		return value
	}

	return parseTypedFormValue(value)
}

func parseTypedFormValue(value string) interface{} {
	if intVal, err := strconv.ParseInt(value, 10, 64); err == nil {
		return intVal
	}
	if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
		return floatVal
	}
	switch value {
	case "true":
		return true
	case "false":
		return false
	}

	return value
}

func isStringOnlyFormField(key string) bool {
	_, ok := stringOnlyFormFields[key]
	return ok
}

func applyFormRequestDefaults(formData map[string]interface{}) {
	for key, defaultValue := range formDefaultValues {
		if _, exists := formData[key]; !exists {
			formData[key] = defaultValue
		}
	}
}
