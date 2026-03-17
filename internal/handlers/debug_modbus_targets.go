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
	endpoint, fromResource, err := resolveModbusTCPTarget(req.Endpoint, req.ResourceID)
	if err != nil {
		return "", err
	}
	if err := validateResolvedModbusTCPEndpoint(endpoint, fromResource); err != nil {
		return "", err
	}
	return endpoint, nil
}

func resolveModbusDirectOrResourceTarget(direct string, resourceID *int64, resourceType string) (string, error) {
	if target := strings.TrimSpace(direct); target != "" {
		return target, nil
	}
	return resolveDebugResourcePath(resourceID, resourceType)
}

func resolveModbusTCPTarget(endpoint string, resourceID *int64) (string, bool, error) {
	if target := strings.TrimSpace(endpoint); target != "" {
		return target, false, nil
	}

	target, err := resolveDebugResourcePath(resourceID, "net")
	if err != nil {
		return "", true, err
	}
	return target, true, nil
}

func validateResolvedModbusTCPEndpoint(endpoint string, fromResource bool) error {
	if err := validateModbusTCPEndpoint(endpoint); err != nil {
		if fromResource {
			return fmt.Errorf("resource path is invalid: %w", err)
		}
		return err
	}
	return nil
}
