package handlers

import (
	"fmt"
	"strings"
)

func (h *Handler) resolveModbusSerialPath(req *modbusSerialDebugRequest) (string, error) {
	if req == nil {
		return "", fmt.Errorf("request is nil")
	}
	return resolveModbusDirectOrResourceTarget(req.SerialPort, req.ResourceID, "serial")
}

func (h *Handler) resolveModbusTCPEndpoint(req *modbusTCPDebugRequest) (string, error) {
	if req == nil {
		return "", fmt.Errorf("request is nil")
	}
	endpoint, err := resolveModbusDirectOrResourceTarget(req.Endpoint, req.ResourceID, "net")
	if err != nil {
		return "", err
	}
	if err := validateModbusTCPEndpoint(endpoint); err != nil {
		if strings.TrimSpace(req.Endpoint) != "" {
			return "", err
		}
		return "", fmt.Errorf("resource path is invalid: %w", err)
	}
	return endpoint, nil
}

func resolveModbusDirectOrResourceTarget(direct string, resourceID *int64, resourceType string) (string, error) {
	if target := strings.TrimSpace(direct); target != "" {
		return target, nil
	}
	return resolveDebugResourcePath(resourceID, resourceType)
}
