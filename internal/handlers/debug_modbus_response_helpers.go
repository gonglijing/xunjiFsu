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
	byteCount, valuesEnd, err := resolveModbusRegisterValueRange(payload, byteCountOffset, valuesOffset)
	if err != nil {
		return nil, nil, err
	}

	quantity := byteCount / 2
	registers := make([]int, 0, quantity)
	for i := valuesOffset; i < valuesEnd; i += 2 {
		value := int(binary.BigEndian.Uint16(payload[i : i+2]))
		registers = append(registers, value)
	}

	return &quantity, registers, nil
}

func parseModbusWritePayload(payload []byte, offset, reportedLen int) (*int, *int, error) {
	if !hasModbusWritePayload(payload, offset) {
		return nil, nil, fmt.Errorf("write response too short: %d", reportedLen)
	}

	address := int(binary.BigEndian.Uint16(payload[offset : offset+2]))
	value := int(binary.BigEndian.Uint16(payload[offset+2 : offset+4]))
	return &address, &value, nil
}

func resolveModbusRegisterValueRange(payload []byte, byteCountOffset, valuesOffset int) (int, int, error) {
	if len(payload) <= byteCountOffset {
		return 0, 0, fmt.Errorf("invalid byte count: 0")
	}

	byteCount := int(payload[byteCountOffset])
	valuesEnd := valuesOffset + byteCount
	if valuesEnd > len(payload) {
		return 0, 0, fmt.Errorf("invalid byte count: %d", byteCount)
	}
	if byteCount%2 != 0 {
		return 0, 0, fmt.Errorf("invalid register byte count: %d", byteCount)
	}

	return byteCount, valuesEnd, nil
}

func hasModbusWritePayload(payload []byte, offset int) bool {
	return len(payload) >= offset+4
}
