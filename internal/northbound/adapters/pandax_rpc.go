package adapters

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gonglijing/xunjiFsu/internal/models"
)

func (a *PandaXAdapter) PullCommands(limit int) ([]*models.NorthboundCommand, error) {
	if limit <= 0 {
		limit = 20
	}

	a.commandMu.Lock()
	defer a.commandMu.Unlock()

	if !a.isInitialized() {
		slog.Info("PandaX pull commands before initialization", "adapter", a.name)
		return nil, fmt.Errorf("adapter not initialized")
	}

	if len(a.commandQueue) == 0 {
		return nil, nil
	}

	if limit > len(a.commandQueue) {
		limit = len(a.commandQueue)
	}

	items := make([]*models.NorthboundCommand, limit)
	copy(items, a.commandQueue[:limit])
	clear(a.commandQueue[:limit])
	a.commandQueue = a.commandQueue[limit:]

	slog.Info("PandaX commands pulled", "adapter", a.name, "count", len(items))
	return items, nil
}

func (a *PandaXAdapter) handleRPCRequest(_ mqtt.Client, message mqtt.Message) {
	slog.Info("PandaX RPC request received", "adapter", a.name, "topic", message.Topic())

	var req struct {
		RequestID string      `json:"requestId"`
		Method    string      `json:"method"`
		Params    any `json:"params"`
	}
	if err := json.Unmarshal(message.Payload(), &req); err != nil {
		slog.Info("PandaX RPC request JSON parse failed", "adapter", a.name, "error", err)
		return
	}

	if strings.TrimSpace(req.RequestID) == "" {
		req.RequestID = parsePandaXRPCRequestID(message.Topic())
	}

	slog.Info("PandaX RPC request parsed", "adapter", a.name, "request_id", req.RequestID, "method", req.Method)

	commands := a.buildRPCCommands(req.RequestID, req.Method, req.Params)
	if len(commands) == 0 {
		slog.Info("PandaX RPC request produced no commands", "adapter", a.name, "request_id", req.RequestID)
		return
	}

	a.commandMu.Lock()
	a.commandQueue = appendCommandQueueWithCap(a.commandQueue, commands, a.commandCap)
	queueLen := len(a.commandQueue)
	a.commandMu.Unlock()

	slog.Info("PandaX RPC commands enqueued", "adapter", a.name, "count", len(commands), "queue_len", queueLen)
}

func (a *PandaXAdapter) buildRPCCommands(requestID, method string, params any) []*models.NorthboundCommand {
	defaultPK, defaultDK := a.defaultIdentity()
	commands := buildPandaXRPCCommands(requestID, method, params, defaultPK, defaultDK)
	if len(commands) == 0 {
		return nil
	}

	for _, cmd := range commands {
		if cmd == nil {
			continue
		}
		cmd.Source = "pandax.rpc.request"
	}

	return commands
}

func buildPandaXRPCCommands(requestID, method string, params any, defaultPK, defaultDK string) []*models.NorthboundCommand {
	out := make([]*models.NorthboundCommand, 0)
	appendProperties := func(pk, dk string, props map[string]any) {
		if len(props) == 0 {
			return
		}
		keys := make([]string, 0, len(props))
		for key := range props {
			trimmed := strings.TrimSpace(key)
			if trimmed == "" {
				continue
			}
			keys = append(keys, trimmed)
		}
		if len(keys) == 0 {
			return
		}
		slices.Sort(keys)

		for _, key := range keys {
			item := &models.NorthboundCommand{
				RequestID:  requestID,
				ProductKey: pk,
				DeviceKey:  dk,
				FieldName:  key,
				Value:      stringifyAny(props[key]),
			}
			if strings.TrimSpace(item.ProductKey) == "" || strings.TrimSpace(item.DeviceKey) == "" {
				continue
			}
			out = append(out, item)
		}
	}

	obj, ok := params.(map[string]any)
	if ok {
		pk := pickFirstNonEmpty(pickConfigString(obj, "productKey", "product_key"), defaultPK)
		dk := pickFirstNonEmpty(pickConfigString(obj, "deviceKey", "device_key"), defaultDK)

		if props, ok := resolveMapValue(obj["properties"]); ok {
			appendProperties(pk, dk, props)
		}

		for _, key := range []string{"sub_device", "subDevice"} {
			sub, ok := resolveMapValue(obj[key])
			if !ok {
				continue
			}
			subPK := pk
			subDK := dk
			if identity, ok := resolveMapValue(sub["identity"]); ok {
				subPK = pickFirstNonEmpty(pickConfigString(identity, "productKey", "product_key"), subPK)
				subDK = pickFirstNonEmpty(pickConfigString(identity, "deviceKey", "device_key"), subDK)
			}
			if props, ok := resolveMapValue(sub["properties"]); ok {
				appendProperties(subPK, subDK, props)
			}
		}

		for _, key := range []string{"sub_devices", "subDevices"} {
			list, ok := obj[key].([]any)
			if !ok || len(list) == 0 {
				continue
			}
			for _, item := range list {
				row, ok := resolveMapValue(item)
				if !ok {
					continue
				}
				subPK := pk
				subDK := dk
				if identity, ok := resolveMapValue(row["identity"]); ok {
					subPK = pickFirstNonEmpty(pickConfigString(identity, "productKey", "product_key"), subPK)
					subDK = pickFirstNonEmpty(pickConfigString(identity, "deviceKey", "device_key"), subDK)
				}
				if props, ok := resolveMapValue(row["properties"]); ok {
					appendProperties(subPK, subDK, props)
				}
			}
		}

		if fieldName := strings.TrimSpace(pickConfigString(obj, "fieldName", "field_name")); fieldName != "" {
			if rawValue, exists := obj["value"]; exists {
				appendProperties(pk, dk, map[string]any{fieldName: rawValue})
			}
		}

		if len(out) == 0 {
			generic := make(map[string]any)
			for key, value := range obj {
				if isPandaXReservedRPCKey(key) {
					continue
				}
				generic[key] = value
			}
			appendProperties(pk, dk, generic)
		}
	}

	if len(out) == 0 && strings.TrimSpace(method) != "" {
		if strings.TrimSpace(defaultPK) != "" && strings.TrimSpace(defaultDK) != "" {
			out = append(out, &models.NorthboundCommand{
				RequestID:  requestID,
				ProductKey: defaultPK,
				DeviceKey:  defaultDK,
				FieldName:  strings.TrimSpace(method),
				Value:      stringifyAny(params),
			})
		}
	}

	return out
}

func isPandaXReservedRPCKey(key string) bool {
	switch strings.TrimSpace(key) {
	case "productKey", "product_key", "deviceKey", "device_key",
		"properties", "sub_device", "subDevice", "sub_devices", "subDevices",
		"fieldName", "field_name", "value":
		return true
	default:
		return false
	}
}

func parsePandaXRPCRequestID(topic string) string {
	trimmed := strings.Trim(strings.TrimSpace(topic), "/")
	const prefix = "v1/devices/me/rpc/request/"
	if !strings.HasPrefix(trimmed, prefix) {
		return ""
	}

	requestID := trimmed[len(prefix):]
	if requestID == "" {
		return ""
	}
	if idx := strings.IndexByte(requestID, '/'); idx >= 0 {
		requestID = requestID[:idx]
	}
	return strings.TrimSpace(requestID)
}
