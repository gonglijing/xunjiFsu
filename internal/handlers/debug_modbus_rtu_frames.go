package handlers

import (
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"
)

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
	if exceptionCode, ok := parseModbusException(functionCode, response[1:]); ok {
		resp.ExceptionCode = exceptionCode
		return resp, nil
	}
	if functionCode&0x80 != 0 {
		return nil, fmt.Errorf("exception response too short: %d", len(response))
	}
	if err := validateModbusFunctionCode(functionCode, req.FunctionCode); err != nil {
		return nil, err
	}

	switch req.FunctionCode {
	case modbusFuncReadHoldingRegisters:
		quantity, registers, err := parseModbusRegisterPayload(response[:len(response)-2], 2, 3)
		if err != nil {
			return nil, err
		}
		address := req.Address
		resp.Address = &address
		resp.Quantity = quantity
		resp.Registers = registers
	case modbusFuncWriteSingleRegister:
		address, value, err := parseModbusWritePayload(response, 2, len(response))
		if err != nil {
			return nil, err
		}
		resp.Address = address
		resp.Value = value
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

	if exceptionCode, ok := parseModbusException(functionCode, response[1:]); ok {
		resp.ExceptionCode = exceptionCode
		return resp, nil
	}

	switch functionCode {
	case modbusFuncReadHoldingRegisters:
		quantity, registers, err := parseModbusRegisterPayload(response[:len(response)-2], 2, 3)
		if err != nil {
			return nil, err
		}
		if len(request) >= 4 {
			address := int(binary.BigEndian.Uint16(request[2:4]))
			resp.Address = &address
		}
		resp.Quantity = quantity
		resp.Registers = registers
	case modbusFuncWriteSingleRegister:
		address, value, err := parseModbusWritePayload(response, 2, len(response))
		if err != nil {
			return nil, err
		}
		resp.Address = address
		resp.Value = value
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
