package handlers

import (
	"fmt"
	"strings"
)

func validateModbusSerialDebugRequest(req *modbusSerialDebugRequest) error {
	if req == nil {
		return fmt.Errorf("request is nil")
	}
	if err := validateModbusSerialTarget(req.SerialPort, req.ResourceID); err != nil {
		return err
	}
	if err := normalizeModbusSerialLineSettings(req); err != nil {
		return err
	}
	if err := normalizeModbusTimeout(&req.TimeoutMs, 800); err != nil {
		return err
	}

	req.RawRequest = strings.TrimSpace(req.RawRequest)
	if isRawModbusRequest(req.RawRequest) {
		return normalizeModbusRawResponseLength(&req.ExpectRespLen)
	}

	return validateModbusStructuredRequest(&req.FunctionCode, req.SlaveID, req.Address, &req.Quantity, &req.Value)
}

func validateModbusTCPDebugRequest(req *modbusTCPDebugRequest) error {
	if req == nil {
		return fmt.Errorf("request is nil")
	}
	if err := validateModbusTCPTarget(req.Endpoint, req.ResourceID); err != nil {
		return err
	}

	req.RawRequest = strings.TrimSpace(req.RawRequest)
	if err := normalizeModbusTimeout(&req.TimeoutMs, 2000); err != nil {
		return err
	}
	if isRawModbusRequest(req.RawRequest) {
		return nil
	}
	if err := validateModbusStructuredRequest(&req.FunctionCode, req.SlaveID, req.Address, &req.Quantity, &req.Value); err != nil {
		return err
	}
	return validateModbusTransactionID(req.TransactionID)
}

func validateModbusSerialTarget(serialPort string, resourceID *int64) error {
	if strings.TrimSpace(serialPort) == "" && (resourceID == nil || *resourceID <= 0) {
		return fmt.Errorf("serial_port or resource_id is required")
	}
	return nil
}

func validateModbusTCPTarget(endpoint string, resourceID *int64) error {
	if strings.TrimSpace(endpoint) == "" && (resourceID == nil || *resourceID <= 0) {
		return fmt.Errorf("endpoint or resource_id is required")
	}
	return nil
}

func normalizeModbusSerialLineSettings(req *modbusSerialDebugRequest) error {
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
	return nil
}

func normalizeModbusRawResponseLength(expectRespLen *int) error {
	if *expectRespLen <= 0 {
		*expectRespLen = 256
	}
	if *expectRespLen > 4096 {
		return fmt.Errorf("expect_response_len must be in [1,4096] when raw_request is provided")
	}
	return nil
}

func validateModbusStructuredRequest(functionCode *int, slaveID, address int, quantity, value *int) error {
	if err := normalizeModbusFunctionCode(functionCode); err != nil {
		return err
	}
	if err := validateModbusAddressing(slaveID, address); err != nil {
		return err
	}
	return normalizeModbusOperation(*functionCode, quantity, value)
}

func validateModbusTransactionID(transactionID int) error {
	if transactionID < 0 || transactionID > 0xFFFF {
		return fmt.Errorf("transaction_id must be in [0,65535]")
	}
	return nil
}
