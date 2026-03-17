package handlers

import (
	"net/http"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/driver"
)

func parseModbusTCPPayload(w http.ResponseWriter, r *http.Request) (*modbusTCPDebugRequest, bool) {
	var req modbusTCPDebugRequest
	if !parseRequestOrWriteBadRequestDefault(w, r, &req) {
		return nil, false
	}
	return &req, true
}

// DebugModbusTCP Modbus TCP 调试
func (h *Handler) DebugModbusTCP(w http.ResponseWriter, r *http.Request) {
	req, ok := parseModbusTCPPayload(w, r)
	if !ok {
		return
	}

	if err := validateModbusTCPDebugRequest(req); err != nil {
		writeModbusDebugParamError(w, err)
		return
	}

	endpoint, err := h.resolveModbusTCPEndpoint(req)
	if err != nil {
		writeModbusDebugResolveError(w, err)
		return
	}

	request, txID, err := buildModbusTCPDebugRequest(req)
	if err != nil {
		writeModbusDebugParamError(w, err)
		return
	}

	response, err := driver.TransceiveTCP(endpoint, buildModbusTCPConfig(req), request)
	if err != nil {
		writeModbusDebugCommError(w, apiErrDebugModbusTCPFailed, err)
		return
	}

	debugResp, err := parseModbusTCPDebugResponse(req, endpoint, request, response, txID)
	if err != nil {
		writeModbusDebugResponseError(w, err)
		return
	}

	WriteSuccess(w, debugResp)
}

func buildModbusTCPDebugRequest(req *modbusTCPDebugRequest) ([]byte, int, error) {
	if isRawModbusRequest(req.RawRequest) {
		return buildRawModbusTCPRequest(req)
	}

	request, txID := buildModbusTCPRequest(req)
	return request, txID, nil
}

func buildModbusTCPConfig(req *modbusTCPDebugRequest) driver.TCPConfig {
	return driver.TCPConfig{Timeout: time.Duration(req.TimeoutMs) * time.Millisecond}
}

func parseModbusTCPDebugResponse(req *modbusTCPDebugRequest, endpoint string, request []byte, response []byte, txID int) (*modbusTCPDebugResponse, error) {
	if isRawModbusRequest(req.RawRequest) {
		return parseModbusTCPRawResponse(endpoint, request, response, txID)
	}
	return parseModbusTCPResponse(req, endpoint, request, response, txID)
}
