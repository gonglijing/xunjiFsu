package adapters

import (
	"encoding/json"
	"fmt"
	"log"
	"sort"
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
		log.Printf("[PandaX-%s] PullCommands: 适配器未初始化", a.name)
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

	log.Printf("[PandaX-%s] PullCommands: 取出 %d 条命令", a.name, len(items))
	return items, nil
}

func (a *PandaXAdapter) handleRPCRequest(_ mqtt.Client, message mqtt.Message) {
	log.Printf("[PandaX-%s] handleRPCRequest: 收到 RPC topic=%s", a.name, message.Topic())

	var req struct {
		RequestID string      `json:"requestId"`
		Method    string      `json:"method"`
		Params    interface{} `json:"params"`
	}
	if err := json.Unmarshal(message.Payload(), &req); err != nil {
		log.Printf("[PandaX-%s] handleRPCRequest: JSON 解析失败: %v", a.name, err)
		return
	}

	if strings.TrimSpace(req.RequestID) == "" {
		req.RequestID = requestIDFromPandaXRPCTopic(message.Topic())
	}

	log.Printf("[PandaX-%s] handleRPCRequest: requestId=%s, method=%s", a.name, req.RequestID, req.Method)

	commands := a.buildCommandsFromRPC(req.RequestID, req.Method, req.Params)
	if len(commands) == 0 {
		log.Printf("[PandaX-%s] handleRPCRequest: 无有效命令", a.name)
		return
	}

	a.commandMu.Lock()
	a.commandQueue = appendCommandQueueWithCap(a.commandQueue, commands, a.commandCap)
	queueLen := len(a.commandQueue)
	a.commandMu.Unlock()

	log.Printf("[PandaX-%s] handleRPCRequest: 入队 %d 条命令, queueLen=%d", a.name, len(commands), queueLen)
}

func (a *PandaXAdapter) buildCommandsFromRPC(requestID, method string, params interface{}) []*models.NorthboundCommand {
	defaultPK, defaultDK := a.defaultIdentity()
	commands := buildPandaXCommands(requestID, method, params, defaultPK, defaultDK)
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

func buildPandaXCommands(requestID, method string, params interface{}, defaultPK, defaultDK string) []*models.NorthboundCommand {
	out := make([]*models.NorthboundCommand, 0)
	appendProperties := func(pk, dk string, props map[string]interface{}) {
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
		sort.Strings(keys)

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

	obj, ok := params.(map[string]interface{})
	if ok {
		pk := pickFirstNonEmpty(pickConfigString(obj, "productKey", "product_key"), defaultPK)
		dk := pickFirstNonEmpty(pickConfigString(obj, "deviceKey", "device_key"), defaultDK)

		if props, ok := mapFromAny(obj["properties"]); ok {
			appendProperties(pk, dk, props)
		}

		for _, key := range []string{"sub_device", "subDevice"} {
			sub, ok := mapFromAny(obj[key])
			if !ok {
				continue
			}
			subPK := pk
			subDK := dk
			if identity, ok := mapFromAny(sub["identity"]); ok {
				subPK = pickFirstNonEmpty(pickConfigString(identity, "productKey", "product_key"), subPK)
				subDK = pickFirstNonEmpty(pickConfigString(identity, "deviceKey", "device_key"), subDK)
			}
			if props, ok := mapFromAny(sub["properties"]); ok {
				appendProperties(subPK, subDK, props)
			}
		}

		for _, key := range []string{"sub_devices", "subDevices"} {
			list, ok := obj[key].([]interface{})
			if !ok || len(list) == 0 {
				continue
			}
			for _, item := range list {
				row, ok := mapFromAny(item)
				if !ok {
					continue
				}
				subPK := pk
				subDK := dk
				if identity, ok := mapFromAny(row["identity"]); ok {
					subPK = pickFirstNonEmpty(pickConfigString(identity, "productKey", "product_key"), subPK)
					subDK = pickFirstNonEmpty(pickConfigString(identity, "deviceKey", "device_key"), subDK)
				}
				if props, ok := mapFromAny(row["properties"]); ok {
					appendProperties(subPK, subDK, props)
				}
			}
		}

		if fieldName := strings.TrimSpace(pickConfigString(obj, "fieldName", "field_name")); fieldName != "" {
			if rawValue, exists := obj["value"]; exists {
				appendProperties(pk, dk, map[string]interface{}{fieldName: rawValue})
			}
		}

		if len(out) == 0 {
			generic := make(map[string]interface{})
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

func requestIDFromPandaXRPCTopic(topic string) string {
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
