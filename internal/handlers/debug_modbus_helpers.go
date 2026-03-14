package handlers

import (
	"database/sql"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/gonglijing/xunjiFsu/internal/database"
)

func isRawModbusRequest(raw string) bool {
	return strings.TrimSpace(raw) != ""
}

func writeModbusDebugParamError(w http.ResponseWriter, err error) {
	WriteBadRequestCode(w, apiErrDebugModbusParamInvalid.Code, fmt.Sprintf("%s: %v", apiErrDebugModbusParamInvalid.Message, err))
}

func writeModbusDebugResolveError(w http.ResponseWriter, err error) {
	if errors.Is(err, sql.ErrNoRows) {
		WriteNotFoundDef(w, apiErrResourceNotFound)
		return
	}
	writeModbusDebugParamError(w, err)
}

func writeModbusDebugCommError(w http.ResponseWriter, def APIErrorDef, err error) {
	WriteErrorCode(w, http.StatusBadGateway, def.Code, fmt.Sprintf("%s: %v", def.Message, err))
}

func writeModbusDebugResponseError(w http.ResponseWriter, err error) {
	WriteErrorCode(w, http.StatusBadGateway, apiErrDebugModbusResponseInvalid.Code, fmt.Sprintf("%s: %v", apiErrDebugModbusResponseInvalid.Message, err))
}

func normalizeModbusTimeout(timeoutMs *int, defaultValue int) error {
	if *timeoutMs <= 0 {
		*timeoutMs = defaultValue
	}
	if *timeoutMs > 60000 {
		return fmt.Errorf("timeout_ms must be <= 60000")
	}
	return nil
}

func normalizeModbusFunctionCode(functionCode *int) error {
	if *functionCode == 0 {
		*functionCode = modbusFuncReadHoldingRegisters
	}
	if *functionCode != modbusFuncReadHoldingRegisters && *functionCode != modbusFuncWriteSingleRegister {
		return fmt.Errorf("unsupported function_code %d, only 3(read) and 6(write) are supported", *functionCode)
	}
	return nil
}

func validateModbusAddressing(slaveID, address int) error {
	if slaveID < 0 || slaveID > 247 {
		return fmt.Errorf("slave_id must be in [0,247]")
	}
	if address < 0 || address > 0xFFFF {
		return fmt.Errorf("address must be in [0,65535]")
	}
	return nil
}

func normalizeModbusOperation(functionCode int, quantity, value *int) error {
	switch functionCode {
	case modbusFuncReadHoldingRegisters:
		if *quantity <= 0 {
			*quantity = 1
		}
		if *quantity > 125 {
			return fmt.Errorf("quantity must be in [1,125] for function_code=3")
		}
	case modbusFuncWriteSingleRegister:
		if *value < 0 || *value > 0xFFFF {
			return fmt.Errorf("value must be in [0,65535] for function_code=6")
		}
	}
	return nil
}

func resolveDebugResourcePath(resourceID *int64, expectedType string) (string, error) {
	if resourceID == nil || *resourceID <= 0 {
		return "", fmt.Errorf("resource_id is required")
	}

	resource, err := database.GetResourceByID(*resourceID)
	if err != nil {
		return "", err
	}
	if resource.Type != expectedType {
		return "", fmt.Errorf("resource %d type is %s, only %s is supported", resource.ID, resource.Type, expectedType)
	}

	path := strings.TrimSpace(resource.Path)
	if path == "" {
		return "", fmt.Errorf("resource %d path is empty", resource.ID)
	}
	return path, nil
}

func validateModbusTCPEndpoint(endpoint string) error {
	if _, err := net.ResolveTCPAddr("tcp", endpoint); err != nil {
		return fmt.Errorf("invalid endpoint %s: %w", endpoint, err)
	}
	return nil
}
