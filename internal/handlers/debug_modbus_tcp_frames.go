package handlers

import (
	"encoding/binary"
	"fmt"
	"time"
)

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
	if exceptionCode, ok := parseModbusException(functionCode, pdu); ok {
		result.ExceptionCode = exceptionCode
		return result, nil
	}
	if err := validateModbusFunctionCode(functionCode, req.FunctionCode); err != nil {
		return nil, err
	}

	switch req.FunctionCode {
	case modbusFuncReadHoldingRegisters:
		quantity, registers, err := parseModbusRegisterPayload(pdu, 1, 2)
		if err != nil {
			return nil, err
		}
		address := req.Address
		result.Address = &address
		result.Quantity = quantity
		result.Registers = registers
	case modbusFuncWriteSingleRegister:
		address, value, err := parseModbusWritePayload(pdu, 1, len(pdu))
		if err != nil {
			return nil, err
		}
		result.Address = address
		result.Value = value
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

	if exceptionCode, ok := parseModbusException(functionCode, pdu); ok {
		result.ExceptionCode = exceptionCode
		return result, nil
	}

	switch functionCode {
	case modbusFuncReadHoldingRegisters:
		quantity, registers, err := parseModbusRegisterPayload(pdu, 1, 2)
		if err != nil {
			return nil, err
		}
		if len(request) >= 12 {
			address := int(binary.BigEndian.Uint16(request[8:10]))
			result.Address = &address
		}
		result.Quantity = quantity
		result.Registers = registers
	case modbusFuncWriteSingleRegister:
		address, value, err := parseModbusWritePayload(pdu, 1, len(pdu))
		if err != nil {
			return nil, err
		}
		result.Address = address
		result.Value = value
	}

	return result, nil
}
