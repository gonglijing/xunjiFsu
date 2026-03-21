package httpapi

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

var formDefaultValues = map[string]any{
	"parity":         "N",
	"interface_type": "network",
	"protocol":       "tcp",
	"baud_rate":      9600,
	"data_bits":      8,
	"stop_bits":      1,
}

func parseIDOrWriteBadRequestDefault(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := ParseID(r)
	if err != nil {
		WriteBadRequestDef(w, apiErrInvalidID)
		return 0, false
	}
	return id, true
}

func parseRequestOrWriteBadRequestDefault(w http.ResponseWriter, r *http.Request, dst any) bool {
	if err := ParseRequest(r, dst); err != nil {
		WriteBadRequestDef(w, apiErrInvalidRequestBody)
		return false
	}
	return true
}

func ParseRequest(r *http.Request, dst any) error {
	if isJSONRequest(r.Header.Get("Content-Type")) {
		return json.NewDecoder(r.Body).Decode(dst)
	}
	return decodeFormRequest(r, dst)
}

func ParseID(r *http.Request) (int64, error) {
	idText := r.PathValue("id")
	if idText == "" {
		return 0, errors.New("missing id path param")
	}
	return strconv.ParseInt(idText, 10, 64)
}

func isJSONRequest(contentType string) bool {
	mediaType, _, err := mime.ParseMediaType(contentType)
	return err == nil && mediaType == jsonMediaType
}

func decodeFormRequest(r *http.Request, dst any) error {
	formData, err := parseFormRequestData(r)
	if err != nil {
		return err
	}

	jsonData, err := json.Marshal(formData)
	if err != nil {
		return err
	}
	return json.Unmarshal(jsonData, dst)
}

func parseFormRequestData(r *http.Request) (map[string]any, error) {
	if err := r.ParseForm(); err != nil {
		return nil, err
	}

	formData := make(map[string]any, len(r.Form))
	for key, values := range r.Form {
		if len(values) == 0 {
			continue
		}
		formData[key] = parseFormFieldValue(key, values[0])
	}

	applyFormRequestDefaults(formData)
	return formData, nil
}

func parseFormFieldValue(key, value string) any {
	if _, ok := stringOnlyFormFields[key]; ok {
		return value
	}
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
	default:
		return value
	}
}

func applyFormRequestDefaults(formData map[string]any) {
	for key, defaultValue := range formDefaultValues {
		if _, exists := formData[key]; !exists {
			formData[key] = defaultValue
		}
	}
}
