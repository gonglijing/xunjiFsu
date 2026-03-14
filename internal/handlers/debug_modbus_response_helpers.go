package handlers

import (
	"encoding/binary"
	"fmt"
)

func parseModbusException(functionCode int, payload []byte) (*int, bool) {
	if functionCode&0x80 == 0 || len(payload) < 2 {
		return nil, false
	}
	exceptionCode := int(payload[1])
	return &exceptionCode, true
}

func validateModbusFunctionCode(functionCode, expected int) error {
	if functionCode != expected {
		return fmt.Errorf("function code mismatch: got %d expect %d", functionCode, expected)
	}
	return nil
}

func parseModbusRegisterPayload(payload []byte, byteCountOffset, valuesOffset int) (*int, []int, error) {
	if len(payload) <= byteCountOffset {
		return nil, nil, fmt.Errorf("invalid byte count: 0")
	}

	byteCount := int(payload[byteCountOffset])
	expectedLen := valuesOffset + byteCount
	if expectedLen > len(payload) {
		return nil, nil, fmt.Errorf("invalid byte count: %d", byteCount)
	}
	if byteCount%2 != 0 {
		return nil, nil, fmt.Errorf("invalid register byte count: %d", byteCount)
	}

	quantity := byteCount / 2
	registers := make([]int, 0, quantity)
	for i := valuesOffset; i < expectedLen; i += 2 {
		value := int(binary.BigEndian.Uint16(payload[i : i+2]))
		registers = append(registers, value)
	}

	return &quantity, registers, nil
}

func parseModbusWritePayload(payload []byte, offset, reportedLen int) (*int, *int, error) {
	if len(payload) < offset+4 {
		return nil, nil, fmt.Errorf("write response too short: %d", reportedLen)
	}

	address := int(binary.BigEndian.Uint16(payload[offset : offset+2]))
	value := int(binary.BigEndian.Uint16(payload[offset+2 : offset+4]))
	return &address, &value, nil
}
