package collector

import (
	"fmt"
	"log"
	"strings"

	"github.com/gonglijing/xunjiFsu/internal/database"
	"github.com/gonglijing/xunjiFsu/internal/models"
)

const commandDriverFunction = "handle"

func (c *Collector) processNorthboundCommands() {
	if c.northboundMgr == nil {
		return
	}

	commands, err := c.northboundMgr.PullCommands(20)
	if err != nil {
		log.Printf("pull northbound commands failed: %v", err)
		return
	}
	if len(commands) == 0 {
		return
	}

	for _, command := range commands {
		if command == nil {
			continue
		}
		err := c.executeNorthboundCommand(command)
		if err != nil {
			log.Printf("execute northbound command failed: source=%s request_id=%s product_key=%s device_key=%s field=%s error=%v",
				command.Source, command.RequestID, command.ProductKey, command.DeviceKey, command.FieldName, err)
		}
		c.reportCommandResult(command, err)
	}
}

func (c *Collector) reportCommandResult(command *models.NorthboundCommand, execErr error) {
	if c.northboundMgr == nil || command == nil {
		return
	}

	result := buildNorthboundCommandResult(command, execErr)
	c.northboundMgr.ReportCommandResult(result)
}

func buildNorthboundCommandResult(command *models.NorthboundCommand, execErr error) *models.NorthboundCommandResult {
	if command == nil {
		return nil
	}

	result := &models.NorthboundCommandResult{
		RequestID:  command.RequestID,
		ProductKey: command.ProductKey,
		DeviceKey:  command.DeviceKey,
		FieldName:  command.FieldName,
		Value:      command.Value,
		Source:     command.Source,
		Success:    execErr == nil,
		Code:       200,
	}
	if execErr != nil {
		result.Code = 500
		result.Message = execErr.Error()
	}

	return result
}

func (c *Collector) executeNorthboundCommand(command *models.NorthboundCommand) error {
	if c.driverExecutor == nil {
		return fmt.Errorf("driver executor is nil")
	}

	normalizedCommand, err := normalizeNorthboundCommand(command)
	if err != nil {
		return err
	}

	device, err := database.GetDeviceByIdentity(normalizedCommand.ProductKey, normalizedCommand.DeviceKey)
	if err != nil || device == nil {
		return fmt.Errorf("device not found by identity")
	}
	if device.DriverID == nil {
		return fmt.Errorf("device has no driver")
	}

	config := buildNorthboundCommandConfig(normalizedCommand, device)

	result, err := c.driverExecutor.ExecuteCommand(device, commandDriverFunction, config)
	if err != nil {
		return err
	}
	if result != nil && !result.Success {
		if strings.TrimSpace(result.Error) != "" {
			return fmt.Errorf("%s", result.Error)
		}
		return fmt.Errorf("driver write returned success=false")
	}

	log.Printf("northbound command executed: source=%s request_id=%s device_id=%d field=%s value=%s",
		normalizedCommand.Source, normalizedCommand.RequestID, device.ID, normalizedCommand.FieldName, normalizedCommand.Value)
	return nil
}

func normalizeNorthboundCommand(command *models.NorthboundCommand) (*models.NorthboundCommand, error) {
	if command == nil {
		return nil, fmt.Errorf("northbound command is nil")
	}

	normalizedCommand := &models.NorthboundCommand{
		RequestID:  strings.TrimSpace(command.RequestID),
		ProductKey: strings.TrimSpace(command.ProductKey),
		DeviceKey:  strings.TrimSpace(command.DeviceKey),
		FieldName:  strings.TrimSpace(command.FieldName),
		Value:      strings.TrimSpace(command.Value),
		Source:     strings.TrimSpace(command.Source),
	}

	if normalizedCommand.ProductKey == "" || normalizedCommand.DeviceKey == "" {
		return nil, fmt.Errorf("missing product_key/device_key")
	}
	if normalizedCommand.FieldName == "" {
		return nil, fmt.Errorf("missing field_name")
	}

	return normalizedCommand, nil
}

func buildNorthboundCommandConfig(command *models.NorthboundCommand, device *models.Device) map[string]string {
	if command == nil || device == nil {
		return nil
	}

	return map[string]string{
		"func_name":      "write",
		"field_name":     command.FieldName,
		"value":          command.Value,
		"product_key":    command.ProductKey,
		"productKey":     command.ProductKey,
		"device_key":     command.DeviceKey,
		"deviceKey":      command.DeviceKey,
		"device_address": device.DeviceAddress,
	}
}
