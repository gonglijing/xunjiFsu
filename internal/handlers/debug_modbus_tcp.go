package handlers

import (
	"net/http"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/driver"
)

// DebugModbusTCP Modbus TCP 调试
func (h *Handler) DebugModbusTCP(w http.ResponseWriter, r *http.Request) {
	var req modbusTCPDebugRequest
	if !parseRequestOrWriteBadRequestDefault(w, r, &req) {
		return
	}

	if err := validateModbusTCPDebugRequest(&req); err != nil {
		writeModbusDebugParamError(w, err)
		return
	}

	endpoint, err := h.resolveModbusTCPEndpoint(&req)
	if err != nil {
		writeModbusDebugResolveError(w, err)
		return
	}

	request, txID := buildModbusTCPRequest(&req)
	if isRawModbusRequest(req.RawRequest) {
		var buildErr error
		request, txID, buildErr = buildRawModbusTCPRequest(&req)
		if buildErr != nil {
			writeModbusDebugParamError(w, buildErr)
			return
		}
	}
	response, err := driver.TransceiveTCP(endpoint, driver.TCPConfig{Timeout: time.Duration(req.TimeoutMs) * time.Millisecond}, request)
	if err != nil {
		writeModbusDebugCommError(w, apiErrDebugModbusTCPFailed, err)
		return
	}

	debugResp, err := parseModbusTCPResponse(&req, endpoint, request, response, txID)
	if isRawModbusRequest(req.RawRequest) {
		debugResp, err = parseModbusTCPRawResponse(endpoint, request, response, txID)
	}
	if err != nil {
		writeModbusDebugResponseError(w, err)
		return
	}

	WriteSuccess(w, debugResp)
}
