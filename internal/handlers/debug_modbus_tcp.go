package handlers

import (
	"database/sql"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/database"
	"github.com/gonglijing/xunjiFsu/internal/driver"
)

type modbusTCPDebugRequest struct {
	ResourceID    *int64 `json:"resource_id"`
	Endpoint      string `json:"endpoint"`
	TimeoutMs     int    `json:"timeout_ms"`
	RawRequest    string `json:"raw_request"`
	SlaveID       int    `json:"slave_id"`
	FunctionCode  int    `json:"function_code"`
	Address       int    `json:"address"`
	Quantity      int    `json:"quantity"`
	Value         int    `json:"value"`
	TransactionID int    `json:"transaction_id"`
}

type modbusTCPDebugResponse struct {
	Endpoint      string `json:"endpoint"`
	RequestHex    string `json:"request_hex"`
	ResponseHex   string `json:"response_hex"`
	TransactionID int    `json:"transaction_id"`
	SlaveID       int    `json:"slave_id"`
	FunctionCode  int    `json:"function_code"`
	Address       *int   `json:"address,omitempty"`
	Quantity      *int   `json:"quantity,omitempty"`
	Value         *int   `json:"value,omitempty"`
	Registers     []int  `json:"registers,omitempty"`
	ExceptionCode *int   `json:"exception_code,omitempty"`
}

// DebugModbusTCP Modbus TCP 调试
func (h *Handler) DebugModbusTCP(w http.ResponseWriter, r *http.Request) {
	var req modbusTCPDebugRequest
	if !parseRequestOrWriteBadRequestDefault(w, r, &req) {
		return
	}

	if err := validateModbusTCPDebugRequest(&req); err != nil {
		WriteBadRequestCode(w, apiErrDebugModbusParamInvalid.Code, fmt.Sprintf("%s: %v", apiErrDebugModbusParamInvalid.Message, err))
		return
	}

	endpoint, err := h.resolveModbusTCPEndpoint(&req)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			WriteNotFoundDef(w, apiErrResourceNotFound)
			return
		}
		WriteBadRequestCode(w, apiErrDebugModbusParamInvalid.Code, fmt.Sprintf("%s: %v", apiErrDebugModbusParamInvalid.Message, err))
		return
	}

	request, txID := buildModbusTCPRequest(&req)
	if strings.TrimSpace(req.RawRequest) != "" {
		var buildErr error
		request, txID, buildErr = buildRawModbusTCPRequest(&req)
		if buildErr != nil {
			WriteBadRequestCode(w, apiErrDebugModbusParamInvalid.Code, fmt.Sprintf("%s: %v", apiErrDebugModbusParamInvalid.Message, buildErr))
			return
		}
	}
	response, err := driver.TransceiveTCP(endpoint, driver.TCPConfig{Timeout: time.Duration(req.TimeoutMs) * time.Millisecond}, request)
	if err != nil {
		WriteErrorCode(w, http.StatusBadGateway, apiErrDebugModbusTCPFailed.Code, fmt.Sprintf("%s: %v", apiErrDebugModbusTCPFailed.Message, err))
		return
	}

	debugResp, err := parseModbusTCPResponse(&req, endpoint, request, response, txID)
	if strings.TrimSpace(req.RawRequest) != "" {
		debugResp, err = parseModbusTCPRawResponse(endpoint, request, response, txID)
	}
	if err != nil {
		WriteErrorCode(w, http.StatusBadGateway, apiErrDebugModbusResponseInvalid.Code, fmt.Sprintf("%s: %v", apiErrDebugModbusResponseInvalid.Message, err))
		return
	}

	WriteSuccess(w, debugResp)
}

func validateModbusTCPDebugRequest(req *modbusTCPDebugRequest) error {
	if req == nil {
		return fmt.Errorf("request is nil")
	}
	if strings.TrimSpace(req.Endpoint) == "" && (req.ResourceID == nil || *req.ResourceID <= 0) {
		return fmt.Errorf("endpoint or resource_id is required")
	}
	req.RawRequest = strings.TrimSpace(req.RawRequest)

	if req.TimeoutMs <= 0 {
		req.TimeoutMs = 2000
	}
	if req.TimeoutMs > 60000 {
		return fmt.Errorf("timeout_ms must be <= 60000")
	}

	if req.RawRequest != "" {
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

	if req.TransactionID < 0 || req.TransactionID > 0xFFFF {
		return fmt.Errorf("transaction_id must be in [0,65535]")
	}

	return nil
}

func buildRawModbusTCPRequest(req *modbusTCPDebugRequest) ([]byte, int, error) {
	if req == nil {
		return nil, 0, fmt.Errorf("request is nil")
	}
	frame, err := parseRawModbusRTUBytes(req.RawRequest)
	if err != nil {
		return nil, 0, err
	}
	txID, err := validateRawModbusTCPFrame(frame)
	if err != nil {
		return nil, 0, err
	}
	return frame, txID, nil
}

func validateRawModbusTCPFrame(frame []byte) (int, error) {
	if len(frame) < 8 {
		return 0, fmt.Errorf("raw_request too short: %d", len(frame))
	}

	protocolID := int(binary.BigEndian.Uint16(frame[2:4]))
	if protocolID != 0 {
		return 0, fmt.Errorf("raw_request protocol id must be 0, got %d", protocolID)
	}

	length := int(binary.BigEndian.Uint16(frame[4:6]))
	if length < 2 {
		return 0, fmt.Errorf("raw_request invalid length: %d", length)
	}
	expect := 6 + length
	if expect != len(frame) {
		return 0, fmt.Errorf("raw_request length mismatch: mbap=%d frame=%d", expect, len(frame))
	}

	txID := int(binary.BigEndian.Uint16(frame[0:2]))
	return txID, nil
}

func (h *Handler) resolveModbusTCPEndpoint(req *modbusTCPDebugRequest) (string, error) {
	if req == nil {
		return "", fmt.Errorf("request is nil")
	}
	if endpoint := strings.TrimSpace(req.Endpoint); endpoint != "" {
		if _, err := net.ResolveTCPAddr("tcp", endpoint); err != nil {
			return "", fmt.Errorf("invalid endpoint %s: %w", endpoint, err)
		}
		return endpoint, nil
	}

	if req.ResourceID == nil || *req.ResourceID <= 0 {
		return "", fmt.Errorf("resource_id is required when endpoint is empty")
	}

	resource, err := database.GetResourceByID(*req.ResourceID)
	if err != nil {
		return "", err
	}
	if resource.Type != "net" {
		return "", fmt.Errorf("resource %d type is %s, only net is supported", resource.ID, resource.Type)
	}

	endpoint := strings.TrimSpace(resource.Path)
	if endpoint == "" {
		return "", fmt.Errorf("resource %d path is empty", resource.ID)
	}
	if _, err := net.ResolveTCPAddr("tcp", endpoint); err != nil {
		return "", fmt.Errorf("resource %d path is invalid: %w", resource.ID, err)
	}
	return endpoint, nil
}

func buildModbusTCPRequest(req *modbusTCPDebugRequest) ([]byte, int) {
	pdu := make([]byte, 5)
	pdu[0] = byte(req.FunctionCode)
	binary.BigEndian.PutUint16(pdu[1:3], uint16(req.Address))
	switch req.FunctionCode {
	case modbusFuncReadHoldingRegisters:
		binary.BigEndian.PutUint16(pdu[3:5], uint16(req.Quantity))
	case modbusFuncWriteSingleRegister:
		binary.BigEndian.PutUint16(pdu[3:5], uint16(req.Value))
	}

	txID := req.TransactionID
	if txID == 0 {
		txID = int(time.Now().UnixNano() & 0xFFFF)
		if txID == 0 {
			txID = 1
		}
	}

	mbap := make([]byte, 7)
	binary.BigEndian.PutUint16(mbap[0:2], uint16(txID))
	binary.BigEndian.PutUint16(mbap[2:4], 0)
	binary.BigEndian.PutUint16(mbap[4:6], uint16(1+len(pdu)))
	mbap[6] = byte(req.SlaveID)

	frame := append(mbap, pdu...)
	return frame, txID
}

func parseModbusTCPResponse(req *modbusTCPDebugRequest, endpoint string, request []byte, response []byte, txID int) (*modbusTCPDebugResponse, error) {
	if len(response) < 9 {
		return nil, fmt.Errorf("response too short: %d", len(response))
	}

	respTxID := int(binary.BigEndian.Uint16(response[0:2]))
	if respTxID != txID {
		return nil, fmt.Errorf("transaction id mismatch: got %d expect %d", respTxID, txID)
	}

	protocolID := int(binary.BigEndian.Uint16(response[2:4]))
	if protocolID != 0 {
		return nil, fmt.Errorf("protocol id must be 0, got %d", protocolID)
	}

	length := int(binary.BigEndian.Uint16(response[4:6]))
	if length != len(response)-6 {
		return nil, fmt.Errorf("invalid length field: %d", length)
	}

	slaveID := int(response[6])
	if slaveID != req.SlaveID {
		return nil, fmt.Errorf("slave id mismatch: got %d expect %d", slaveID, req.SlaveID)
	}

	pdu := response[7:]
	if len(pdu) < 2 {
		return nil, fmt.Errorf("pdu too short: %d", len(pdu))
	}

	result := &modbusTCPDebugResponse{
		Endpoint:      endpoint,
		RequestHex:    formatHex(request),
		ResponseHex:   formatHex(response),
		TransactionID: txID,
		SlaveID:       req.SlaveID,
		FunctionCode:  req.FunctionCode,
	}

	functionCode := int(pdu[0])
	if functionCode&0x80 != 0 {
		exceptionCode := int(pdu[1])
		result.ExceptionCode = &exceptionCode
		return result, nil
	}
	if functionCode != req.FunctionCode {
		return nil, fmt.Errorf("function code mismatch: got %d expect %d", functionCode, req.FunctionCode)
	}

	switch req.FunctionCode {
	case modbusFuncReadHoldingRegisters:
		byteCount := int(pdu[1])
		if len(pdu) < 2+byteCount {
			return nil, fmt.Errorf("invalid byte count: %d", byteCount)
		}
		if byteCount%2 != 0 {
			return nil, fmt.Errorf("invalid register byte count: %d", byteCount)
		}
		quantity := byteCount / 2
		registers := make([]int, 0, quantity)
		for i := 0; i < byteCount; i += 2 {
			value := int(binary.BigEndian.Uint16(pdu[2+i : 2+i+2]))
			registers = append(registers, value)
		}
		address := req.Address
		result.Address = &address
		result.Quantity = &quantity
		result.Registers = registers
	case modbusFuncWriteSingleRegister:
		if len(pdu) < 5 {
			return nil, fmt.Errorf("write response too short: %d", len(pdu))
		}
		address := int(binary.BigEndian.Uint16(pdu[1:3]))
		value := int(binary.BigEndian.Uint16(pdu[3:5]))
		result.Address = &address
		result.Value = &value
	}

	return result, nil
}

func parseModbusTCPRawResponse(endpoint string, request []byte, response []byte, txID int) (*modbusTCPDebugResponse, error) {
	if len(request) < 8 {
		return nil, fmt.Errorf("raw request too short: %d", len(request))
	}
	if len(response) < 9 {
		return nil, fmt.Errorf("response too short: %d", len(response))
	}

	respTxID := int(binary.BigEndian.Uint16(response[0:2]))
	if respTxID != txID {
		return nil, fmt.Errorf("transaction id mismatch: got %d expect %d", respTxID, txID)
	}

	protocolID := int(binary.BigEndian.Uint16(response[2:4]))
	if protocolID != 0 {
		return nil, fmt.Errorf("protocol id must be 0, got %d", protocolID)
	}

	length := int(binary.BigEndian.Uint16(response[4:6]))
	if length != len(response)-6 {
		return nil, fmt.Errorf("invalid length field: %d", length)
	}

	slaveID := int(response[6])
	pdu := response[7:]
	if len(pdu) < 2 {
		return nil, fmt.Errorf("pdu too short: %d", len(pdu))
	}

	functionCode := int(pdu[0])
	result := &modbusTCPDebugResponse{
		Endpoint:      endpoint,
		RequestHex:    formatHex(request),
		ResponseHex:   formatHex(response),
		TransactionID: txID,
		SlaveID:       slaveID,
		FunctionCode:  functionCode,
	}

	if functionCode&0x80 != 0 {
		exceptionCode := int(pdu[1])
		result.ExceptionCode = &exceptionCode
		return result, nil
	}

	switch functionCode {
	case modbusFuncReadHoldingRegisters:
		byteCount := int(pdu[1])
		if len(pdu) < 2+byteCount {
			return nil, fmt.Errorf("invalid byte count: %d", byteCount)
		}
		if byteCount%2 != 0 {
			return nil, fmt.Errorf("invalid register byte count: %d", byteCount)
		}

		quantity := byteCount / 2
		registers := make([]int, 0, quantity)
		for i := 0; i < byteCount; i += 2 {
			value := int(binary.BigEndian.Uint16(pdu[2+i : 2+i+2]))
			registers = append(registers, value)
		}
		if len(request) >= 12 {
			address := int(binary.BigEndian.Uint16(request[8:10]))
			result.Address = &address
		}
		result.Quantity = &quantity
		result.Registers = registers
	case modbusFuncWriteSingleRegister:
		if len(pdu) < 5 {
			return nil, fmt.Errorf("write response too short: %d", len(pdu))
		}
		address := int(binary.BigEndian.Uint16(pdu[1:3]))
		value := int(binary.BigEndian.Uint16(pdu[3:5]))
		result.Address = &address
		result.Value = &value
	}

	return result, nil
}
