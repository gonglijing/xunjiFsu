package httpapi

import (
	"net/http"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/driver"
)

func (api *DebugModbusAPI) DebugModbusSerial(w http.ResponseWriter, r *http.Request) {
	req, ok := parseModbusSerialPayload(w, r)
	if !ok {
		return
	}
	if err := validateModbusSerialDebugRequest(req); err != nil {
		writeModbusDebugParamError(w, err)
		return
	}
	path, err := resolveModbusSerialPath(req)
	if err != nil {
		writeModbusDebugResolveError(w, err)
		return
	}
	request, expectLen, err := buildModbusSerialDebugRequest(req)
	if err != nil {
		writeModbusDebugParamError(w, err)
		return
	}
	response, err := driver.TransceiveSerial(path, buildModbusSerialConfig(req), request, expectLen)
	if err != nil {
		writeModbusDebugCommError(w, errDebugModbusSerialFailed, err)
		return
	}
	debugResp, err := parseModbusSerialDebugResponse(req, path, request, response)
	if err != nil {
		writeModbusDebugResponseError(w, err)
		return
	}
	WriteSuccess(w, debugResp)
}

func parseModbusSerialPayload(w http.ResponseWriter, r *http.Request) (*modbusSerialDebugRequest, bool) {
	var req modbusSerialDebugRequest
	if err := ParseRequest(r, &req); err != nil {
		WriteBadRequestDef(w, apiErrInvalidRequestBody)
		return nil, false
	}
	return &req, true
}

func buildModbusSerialDebugRequest(req *modbusSerialDebugRequest) ([]byte, int, error) {
	if isRawModbusRequest(req.RawRequest) {
		return buildRawModbusRTURequest(req)
	}
	request, expectLen := buildModbusRTURequest(req)
	return request, expectLen, nil
}

func buildModbusSerialConfig(req *modbusSerialDebugRequest) driver.SerialConfig {
	return driver.SerialConfig{
		BaudRate:    req.BaudRate,
		DataBits:    req.DataBits,
		Parity:      req.Parity,
		StopBits:    req.StopBits,
		ReadTimeout: time.Duration(req.TimeoutMs) * time.Millisecond,
	}
}

func parseModbusSerialDebugResponse(req *modbusSerialDebugRequest, path string, request []byte, response []byte) (*modbusSerialDebugResponse, error) {
	if isRawModbusRequest(req.RawRequest) {
		return parseModbusRTURawResponse(path, request, response)
	}
	return parseModbusRTUResponse(req, path, request, response)
}
