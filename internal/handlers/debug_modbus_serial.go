package handlers

import (
	"database/sql"
	"encoding/binary"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/database"
	"github.com/gonglijing/xunjiFsu/internal/driver"
)

const (
	modbusFuncReadHoldingRegisters = 0x03
	modbusFuncWriteSingleRegister  = 0x06
)

type modbusSerialDebugRequest struct {
	ResourceID    *int64 `json:"resource_id"`
	SerialPort    string `json:"serial_port"`
	BaudRate      int    `json:"baud_rate"`
	DataBits      int    `json:"data_bits"`
	StopBits      int    `json:"stop_bits"`
	Parity        string `json:"parity"`
	TimeoutMs     int    `json:"timeout_ms"`
	RawRequest    string `json:"raw_request"`
	ExpectRespLen int    `json:"expect_response_len"`
	SlaveID       int    `json:"slave_id"`
	FunctionCode  int    `json:"function_code"`
	Address       int    `json:"address"`
	Quantity      int    `json:"quantity"`
	Value         int    `json:"value"`
}

type modbusSerialDebugResponse struct {
	Port          string `json:"port"`
	RequestHex    string `json:"request_hex"`
	ResponseHex   string `json:"response_hex"`
	SlaveID       int    `json:"slave_id"`
	FunctionCode  int    `json:"function_code"`
	Address       *int   `json:"address,omitempty"`
	Quantity      *int   `json:"quantity,omitempty"`
	Value         *int   `json:"value,omitempty"`
	Registers     []int  `json:"registers,omitempty"`
	ExceptionCode *int   `json:"exception_code,omitempty"`
}

// DebugModbusSerial 串口 Modbus 调试
func (h *Handler) DebugModbusSerial(w http.ResponseWriter, r *http.Request) {
	var req modbusSerialDebugRequest
	if !parseRequestOrWriteBadRequestDefault(w, r, &req) {
		return
	}

	if err := validateModbusSerialDebugRequest(&req); err != nil {
		WriteBadRequestCode(w, apiErrDebugModbusParamInvalid.Code, fmt.Sprintf("%s: %v", apiErrDebugModbusParamInvalid.Message, err))
		return
	}

	path, err := h.resolveModbusSerialPath(&req)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			WriteNotFoundDef(w, apiErrResourceNotFound)
			return
		}
		WriteBadRequestCode(w, apiErrDebugModbusParamInvalid.Code, fmt.Sprintf("%s: %v", apiErrDebugModbusParamInvalid.Message, err))
		return
	}

	request, expectLen := buildModbusRTURequest(&req)
	if strings.TrimSpace(req.RawRequest) != "" {
		var buildErr error
		request, expectLen, buildErr = buildRawModbusRTURequest(&req)
		if buildErr != nil {
			WriteBadRequestCode(w, apiErrDebugModbusParamInvalid.Code, fmt.Sprintf("%s: %v", apiErrDebugModbusParamInvalid.Message, buildErr))
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
		WriteErrorCode(w, http.StatusBadGateway, apiErrDebugModbusSerialFailed.Code, fmt.Sprintf("%s: %v", apiErrDebugModbusSerialFailed.Message, err))
		return
	}

	debugResp, err := parseModbusRTUResponse(&req, path, request, response)
	if strings.TrimSpace(req.RawRequest) != "" {
		debugResp, err = parseModbusRTURawResponse(path, request, response)
	}
	if err != nil {
		WriteErrorCode(w, http.StatusBadGateway, apiErrDebugModbusResponseInvalid.Code, fmt.Sprintf("%s: %v", apiErrDebugModbusResponseInvalid.Message, err))
		return
	}

	WriteSuccess(w, debugResp)
}

func validateModbusSerialDebugRequest(req *modbusSerialDebugRequest) error {
	if req == nil {
		return fmt.Errorf("request is nil")
	}

	if strings.TrimSpace(req.SerialPort) == "" && (req.ResourceID == nil || *req.ResourceID <= 0) {
		return fmt.Errorf("serial_port or resource_id is required")
	}

	if req.BaudRate <= 0 {
		req.BaudRate = 9600
	}
	if req.DataBits < 5 || req.DataBits > 8 {
		req.DataBits = 8
	}
	if req.StopBits != 2 {
		req.StopBits = 1
	}
	req.Parity = strings.ToUpper(strings.TrimSpace(req.Parity))
	switch req.Parity {
	case "", "N", "NONE":
		req.Parity = "N"
	case "E", "EVEN":
		req.Parity = "E"
	case "O", "ODD":
		req.Parity = "O"
	default:
		return fmt.Errorf("parity must be one of N/E/O")
	}

	if req.TimeoutMs <= 0 {
		req.TimeoutMs = 800
	}
	if req.TimeoutMs > 60000 {
		return fmt.Errorf("timeout_ms must be <= 60000")
	}

	req.RawRequest = strings.TrimSpace(req.RawRequest)
	if req.RawRequest != "" {
		if req.ExpectRespLen <= 0 {
			req.ExpectRespLen = 256
		}
		if req.ExpectRespLen > 4096 {
			return fmt.Errorf("expect_response_len must be in [1,4096] when raw_request is provided")
		}
		return nil
	}

	if req.FunctionCode == 0 {
		req.FunctionCode = modbusFuncReadHoldingRegisters
	}
	if req.FunctionCode != modbusFuncReadHoldingRegisters && req.FunctionCode != modbusFuncWriteSingleRegister {
		return fmt.Errorf("unsupported function_code %d, only 3(read) and 6(write) are supported", req.FunctionCode)
	}

	if req.SlaveID < 0 || req.SlaveID > 247 {
		return fmt.Errorf("slave_id must be in [0,247]")
	}
	if req.Address < 0 || req.Address > 0xFFFF {
		return fmt.Errorf("address must be in [0,65535]")
	}

	switch req.FunctionCode {
	case modbusFuncReadHoldingRegisters:
		if req.Quantity <= 0 {
			req.Quantity = 1
		}
		if req.Quantity > 125 {
			return fmt.Errorf("quantity must be in [1,125] for function_code=3")
		}
	case modbusFuncWriteSingleRegister:
		if req.Value < 0 || req.Value > 0xFFFF {
			return fmt.Errorf("value must be in [0,65535] for function_code=6")
		}
	}

	return nil
}

func buildRawModbusRTURequest(req *modbusSerialDebugRequest) ([]byte, int, error) {
	if req == nil {
		return nil, 0, fmt.Errorf("request is nil")
	}
	frame, err := parseRawModbusRTUBytes(req.RawRequest)
	if err != nil {
		return nil, 0, err
	}
	expectLen := req.ExpectRespLen
	if expectLen <= 0 {
		expectLen = 256
	}
	return frame, expectLen, nil
}

func parseRawModbusRTUBytes(raw string) ([]byte, error) {
	tokens := strings.FieldsFunc(raw, func(r rune) bool {
		switch r {
		case ' ', '\t', '\n', '\r', ',', ';':
			return true
		default:
			return false
		}
	})
	if len(tokens) == 0 {
		return nil, fmt.Errorf("raw_request is empty")
	}

	type rawPart struct {
		value      byte
		isCRCLow   bool
		isCRCHigh  bool
		isCRCField bool
	}

	parts := make([]rawPart, 0, len(tokens)+2)
	for _, token := range tokens {
		normalized := strings.ToLower(strings.TrimSpace(token))
		normalized = strings.TrimPrefix(normalized, "0x")
		normalized = strings.ReplaceAll(normalized, "_", "")
		normalized = strings.ReplaceAll(normalized, "-", "")

		switch normalized {
		case "crc", "crc16":
			parts = append(parts,
				rawPart{isCRCLow: true, isCRCField: true},
				rawPart{isCRCHigh: true, isCRCField: true},
			)
			continue
		case "crca", "crcl", "crclo", "crclow", "crclsb":
			parts = append(parts, rawPart{isCRCLow: true, isCRCField: true})
			continue
		case "crcb", "crch", "crchi", "crchigh", "crcmsb":
			parts = append(parts, rawPart{isCRCHigh: true, isCRCField: true})
			continue
		}

		parsed, err := strconv.ParseUint(normalized, 16, 8)
		if err != nil {
			parsed, err = strconv.ParseUint(strings.TrimSpace(token), 10, 8)
			if err != nil {
				return nil, fmt.Errorf("invalid raw byte %q", token)
			}
		}
		parts = append(parts, rawPart{value: byte(parsed)})
	}

	firstCRCIndex := -1
	crcFields := 0
	hasLow := false
	hasHigh := false
	for i, part := range parts {
		if part.isCRCField {
			crcFields++
			if firstCRCIndex == -1 {
				firstCRCIndex = i
			}
			hasLow = hasLow || part.isCRCLow
			hasHigh = hasHigh || part.isCRCHigh
		}
	}

	if crcFields > 0 {
		if crcFields != 2 || !hasLow || !hasHigh {
			return nil, fmt.Errorf("raw crc placeholder must contain low and high bytes")
		}
		if firstCRCIndex <= 0 {
			return nil, fmt.Errorf("raw_request missing payload before crc")
		}
		for i := firstCRCIndex; i < len(parts); i++ {
			if !parts[i].isCRCField {
				return nil, fmt.Errorf("crc placeholder must be at frame tail")
			}
		}

		payload := make([]byte, 0, firstCRCIndex)
		for i := 0; i < firstCRCIndex; i++ {
			payload = append(payload, parts[i].value)
		}
		crc := crc16Modbus(payload)
		for i := firstCRCIndex; i < len(parts); i++ {
			if parts[i].isCRCLow {
				parts[i].value = byte(crc & 0xFF)
			}
			if parts[i].isCRCHigh {
				parts[i].value = byte(crc >> 8)
			}
		}
	}

	frame := make([]byte, 0, len(parts))
	for _, part := range parts {
		frame = append(frame, part.value)
	}
	return frame, nil
}

func (h *Handler) resolveModbusSerialPath(req *modbusSerialDebugRequest) (string, error) {
	if req == nil {
		return "", fmt.Errorf("request is nil")
	}
	if path := strings.TrimSpace(req.SerialPort); path != "" {
		return path, nil
	}

	if req.ResourceID == nil || *req.ResourceID <= 0 {
		return "", fmt.Errorf("resource_id is required when serial_port is empty")
	}

	resource, err := database.GetResourceByID(*req.ResourceID)
	if err != nil {
		return "", err
	}
	if resource.Type != "serial" {
		return "", fmt.Errorf("resource %d type is %s, only serial is supported", resource.ID, resource.Type)
	}
	path := strings.TrimSpace(resource.Path)
	if path == "" {
		return "", fmt.Errorf("resource %d path is empty", resource.ID)
	}
	return path, nil
}

func buildModbusRTURequest(req *modbusSerialDebugRequest) ([]byte, int) {
	frame := make([]byte, 6)
	frame[0] = byte(req.SlaveID)
	frame[1] = byte(req.FunctionCode)
	binary.BigEndian.PutUint16(frame[2:4], uint16(req.Address))

	expectLen := 0
	switch req.FunctionCode {
	case modbusFuncReadHoldingRegisters:
		binary.BigEndian.PutUint16(frame[4:6], uint16(req.Quantity))
		expectLen = 5 + req.Quantity*2
	case modbusFuncWriteSingleRegister:
		binary.BigEndian.PutUint16(frame[4:6], uint16(req.Value))
		expectLen = 8
	}

	crc := crc16Modbus(frame)
	frame = append(frame, byte(crc&0xFF), byte(crc>>8))
	return frame, expectLen
}

func parseModbusRTUResponse(req *modbusSerialDebugRequest, port string, request []byte, response []byte) (*modbusSerialDebugResponse, error) {
	if len(response) < 5 {
		return nil, fmt.Errorf("response too short: %d", len(response))
	}
	if response[0] != byte(req.SlaveID) {
		return nil, fmt.Errorf("slave id mismatch: got %d expect %d", response[0], req.SlaveID)
	}

	crcRead := binary.LittleEndian.Uint16(response[len(response)-2:])
	crcWant := crc16Modbus(response[:len(response)-2])
	if crcRead != crcWant {
		return nil, fmt.Errorf("crc mismatch: got 0x%04X expect 0x%04X", crcRead, crcWant)
	}

	resp := &modbusSerialDebugResponse{
		Port:         port,
		RequestHex:   formatHex(request),
		ResponseHex:  formatHex(response),
		SlaveID:      req.SlaveID,
		FunctionCode: req.FunctionCode,
	}

	functionCode := int(response[1])
	if functionCode&0x80 != 0 {
		if len(response) < 5 {
			return nil, fmt.Errorf("exception response too short: %d", len(response))
		}
		exceptionCode := int(response[2])
		resp.ExceptionCode = &exceptionCode
		return resp, nil
	}

	if functionCode != req.FunctionCode {
		return nil, fmt.Errorf("function code mismatch: got %d expect %d", functionCode, req.FunctionCode)
	}

	switch req.FunctionCode {
	case modbusFuncReadHoldingRegisters:
		byteCount := int(response[2])
		expectedLen := 3 + byteCount + 2
		if expectedLen > len(response) {
			return nil, fmt.Errorf("invalid byte count: %d", byteCount)
		}
		if byteCount%2 != 0 {
			return nil, fmt.Errorf("invalid register byte count: %d", byteCount)
		}

		quantity := byteCount / 2
		registers := make([]int, 0, quantity)
		for i := 0; i < byteCount; i += 2 {
			value := int(binary.BigEndian.Uint16(response[3+i : 3+i+2]))
			registers = append(registers, value)
		}
		address := req.Address
		resp.Address = &address
		resp.Quantity = &quantity
		resp.Registers = registers
	case modbusFuncWriteSingleRegister:
		if len(response) < 8 {
			return nil, fmt.Errorf("write response too short: %d", len(response))
		}
		address := int(binary.BigEndian.Uint16(response[2:4]))
		value := int(binary.BigEndian.Uint16(response[4:6]))
		resp.Address = &address
		resp.Value = &value
	}

	return resp, nil
}

func parseModbusRTURawResponse(port string, request []byte, response []byte) (*modbusSerialDebugResponse, error) {
	if len(request) < 2 {
		return nil, fmt.Errorf("raw request too short: %d", len(request))
	}
	if len(response) < 5 {
		return nil, fmt.Errorf("response too short: %d", len(response))
	}

	reqSlaveID := int(request[0])
	reqFuncCode := int(request[1])
	if response[0] != byte(reqSlaveID) {
		return nil, fmt.Errorf("slave id mismatch: got %d expect %d", response[0], reqSlaveID)
	}

	crcRead := binary.LittleEndian.Uint16(response[len(response)-2:])
	crcWant := crc16Modbus(response[:len(response)-2])
	if crcRead != crcWant {
		return nil, fmt.Errorf("crc mismatch: got 0x%04X expect 0x%04X", crcRead, crcWant)
	}

	functionCode := int(response[1])
	resp := &modbusSerialDebugResponse{
		Port:         port,
		RequestHex:   formatHex(request),
		ResponseHex:  formatHex(response),
		SlaveID:      reqSlaveID,
		FunctionCode: functionCode,
	}

	if functionCode&0x80 != 0 {
		exceptionCode := int(response[2])
		resp.ExceptionCode = &exceptionCode
		return resp, nil
	}

	switch functionCode {
	case modbusFuncReadHoldingRegisters:
		byteCount := int(response[2])
		expectedLen := 3 + byteCount + 2
		if expectedLen > len(response) {
			return nil, fmt.Errorf("invalid byte count: %d", byteCount)
		}
		if byteCount%2 != 0 {
			return nil, fmt.Errorf("invalid register byte count: %d", byteCount)
		}

		quantity := byteCount / 2
		registers := make([]int, 0, quantity)
		for i := 0; i < byteCount; i += 2 {
			value := int(binary.BigEndian.Uint16(response[3+i : 3+i+2]))
			registers = append(registers, value)
		}
		if len(request) >= 4 {
			address := int(binary.BigEndian.Uint16(request[2:4]))
			resp.Address = &address
		}
		resp.Quantity = &quantity
		resp.Registers = registers
	case modbusFuncWriteSingleRegister:
		if len(response) < 8 {
			return nil, fmt.Errorf("write response too short: %d", len(response))
		}
		address := int(binary.BigEndian.Uint16(response[2:4]))
		value := int(binary.BigEndian.Uint16(response[4:6]))
		resp.Address = &address
		resp.Value = &value
	default:
		resp.FunctionCode = reqFuncCode
	}

	return resp, nil
}

func crc16Modbus(data []byte) uint16 {
	crc := uint16(0xFFFF)
	for _, v := range data {
		crc ^= uint16(v)
		for i := 0; i < 8; i++ {
			if crc&0x0001 != 0 {
				crc >>= 1
				crc ^= 0xA001
			} else {
				crc >>= 1
			}
		}
	}
	return crc
}

func formatHex(data []byte) string {
	if len(data) == 0 {
		return ""
	}
	return fmt.Sprintf("% X", data)
}
