package handlers

import (
	"net/http"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/driver"
)

// DebugModbusSerial 串口 Modbus 调试
func (h *Handler) DebugModbusSerial(w http.ResponseWriter, r *http.Request) {
	var req modbusSerialDebugRequest
	if !parseRequestOrWriteBadRequestDefault(w, r, &req) {
		return
	}

	if err := validateModbusSerialDebugRequest(&req); err != nil {
		writeModbusDebugParamError(w, err)
		return
	}

	path, err := h.resolveModbusSerialPath(&req)
	if err != nil {
		writeModbusDebugResolveError(w, err)
		return
	}

	request, expectLen := buildModbusRTURequest(&req)
	if isRawModbusRequest(req.RawRequest) {
		var buildErr error
		request, expectLen, buildErr = buildRawModbusRTURequest(&req)
		if buildErr != nil {
			writeModbusDebugParamError(w, buildErr)
			return
		}
	}
	serialCfg := driver.SerialConfig{
		BaudRate:    req.BaudRate,
		DataBits:    req.DataBits,
		Parity:      req.Parity,
		StopBits:    req.StopBits,
		ReadTimeout: time.Duration(req.TimeoutMs) * time.Millisecond,
	}

	response, err := driver.TransceiveSerial(path, serialCfg, request, expectLen)
	if err != nil {
		writeModbusDebugCommError(w, apiErrDebugModbusSerialFailed, err)
		return
	}

	debugResp, err := parseModbusRTUResponse(&req, path, request, response)
	if isRawModbusRequest(req.RawRequest) {
		debugResp, err = parseModbusRTURawResponse(path, request, response)
	}
	if err != nil {
		writeModbusDebugResponseError(w, err)
		return
	}

	WriteSuccess(w, debugResp)
}
