package httpapi

import (
	"encoding/json"
	"log"
	"net/http"
)

type APIErrorDef struct {
	Code    string
	Message string
}

type APIResponse struct {
	Success bool   `json:"success"`
	Data    any    `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

func WriteJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func WriteSuccess(w http.ResponseWriter, data any) {
	writeSuccessStatus(w, http.StatusOK, data)
}

func WriteCreated(w http.ResponseWriter, data any) {
	writeSuccessStatus(w, http.StatusCreated, data)
}

func WriteDeleted(w http.ResponseWriter) {
	WriteSuccess(w, nil)
}

func WriteBadRequest(w http.ResponseWriter, message string) {
	WriteErrorCode(w, http.StatusBadRequest, "E_BAD_REQUEST", message)
}

func WriteBadRequestCode(w http.ResponseWriter, code, message string) {
	WriteErrorCode(w, http.StatusBadRequest, code, message)
}

func WriteBadRequestDef(w http.ResponseWriter, def APIErrorDef) {
	WriteErrorCode(w, http.StatusBadRequest, def.Code, def.Message)
}

func WriteNotFoundDef(w http.ResponseWriter, def APIErrorDef) {
	WriteErrorCode(w, http.StatusNotFound, def.Code, def.Message)
}

func WriteNotFound(w http.ResponseWriter, message string) {
	WriteErrorCode(w, http.StatusNotFound, "E_NOT_FOUND", message)
}

func WriteServerErrorDef(w http.ResponseWriter, def APIErrorDef) {
	WriteErrorCode(w, http.StatusInternalServerError, def.Code, def.Message)
}

func WriteErrorCode(w http.ResponseWriter, status int, code, message string) {
	WriteJSON(w, status, APIResponse{
		Success: false,
		Error:   message,
		Code:    code,
		Message: message,
	})
}

func writeSuccessStatus(w http.ResponseWriter, status int, data any) {
	WriteJSON(w, status, APIResponse{
		Success: true,
		Data:    data,
	})
}

func writeServerErrorWithLog(w http.ResponseWriter, def APIErrorDef, err error) {
	if err != nil {
		if def.Code != "" {
			log.Printf("%s: %v", def.Code, err)
		} else {
			log.Printf("%s: %v", def.Message, err)
		}
	}
	WriteServerErrorDef(w, def)
}
