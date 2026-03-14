package adapters

import (
	"encoding/json"
	"sort"
	"strings"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gonglijing/xunjiFsu/internal/models"
)

func (a *IThingsAdapter) handleDownlink(_ mqtt.Client, message mqtt.Message) {
	topicType, productID, deviceName := parseIThingsDownTopic(message.Topic())
	if topicType == "" {
		return
	}

	var req struct {
		Method   string      `json:"method"`
		MsgToken string      `json:"msgToken"`
		ActionID string      `json:"actionID"`
		Params   interface{} `json:"params"`
		Data     interface{} `json:"data"`
	}
	if err := json.Unmarshal(message.Payload(), &req); err != nil {
		return
	}

	requestID := strings.TrimSpace(req.MsgToken)
	if requestID == "" {
		requestID = a.nextID("req")
	}

	commands, method, actionID := buildIThingsCommands(requestID, topicType, req.Method, req.ActionID, req.Params, productID, deviceName)
	if len(commands) == 0 {
		return
	}

	a.commandMu.Lock()
	a.commandQueue = appendCommandQueueWithCap(a.commandQueue, commands, a.commandCap)
	a.commandMu.Unlock()

	a.requestMu.Lock()
	a.requestStates[requestID] = &iThingsRequestState{
		RequestID:  requestID,
		ProductID:  productID,
		DeviceName: deviceName,
		Method:     method,
		TopicType:  topicType,
		ActionID:   actionID,
		Pending:    len(commands),
		Success:    true,
	}
	a.requestMu.Unlock()
}

func buildIThingsCommands(requestID, topicType, method, actionID string, params interface{}, productID, deviceName string) ([]*models.NorthboundCommand, string, string) {
	topicType = strings.ToLower(strings.TrimSpace(topicType))
	method = strings.TrimSpace(method)
	actionID = strings.TrimSpace(actionID)

	out := make([]*models.NorthboundCommand, 0)

	appendPropertyCommands := func(values map[string]interface{}) {
		if len(values) == 0 {
			return
		}
		keys := make([]string, 0, len(values))
		for key := range values {
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
			out = append(out, &models.NorthboundCommand{
				RequestID:  requestID,
				ProductKey: productID,
				DeviceKey:  deviceName,
				FieldName:  key,
				Value:      stringifyAny(values[key]),
				Source:     "ithings.down.property",
			})
		}
	}

	switch topicType {
	case "property":
		if strings.EqualFold(method, "control") {
			if obj, ok := mapFromAny(params); ok {
				appendPropertyCommands(obj)
			}
		}
	case "action":
		if strings.EqualFold(method, "action") && actionID != "" {
			out = append(out, &models.NorthboundCommand{
				RequestID:  requestID,
				ProductKey: productID,
				DeviceKey:  deviceName,
				FieldName:  actionID,
				Value:      stringifyAny(params),
				Source:     "ithings.down.action",
			})
		}
	}

	return out, method, actionID
}

func (a *IThingsAdapter) applyCommandResult(result *models.NorthboundCommandResult) (*iThingsRequestState, bool) {
	a.requestMu.Lock()
	defer a.requestMu.Unlock()

	state, ok := a.requestStates[result.RequestID]
	if !ok {
		fallback := &iThingsRequestState{
			RequestID:  result.RequestID,
			ProductID:  strings.TrimSpace(result.ProductKey),
			DeviceName: strings.TrimSpace(result.DeviceKey),
			TopicType:  "property",
			Method:     "control",
			Pending:    1,
			Success:    result.Success,
			Code:       result.Code,
			Message:    strings.TrimSpace(result.Message),
			FieldName:  strings.TrimSpace(result.FieldName),
			Value:      result.Value,
		}
		return fallback, true
	}

	state.Pending--
	if state.Pending < 0 {
		state.Pending = 0
	}
	if !result.Success {
		state.Success = false
	}
	if result.Code != 0 {
		state.Code = result.Code
	}
	if text := strings.TrimSpace(result.Message); text != "" {
		state.Message = text
	}
	if field := strings.TrimSpace(result.FieldName); field != "" {
		state.FieldName = field
	}
	if strings.TrimSpace(result.Value) != "" {
		state.Value = result.Value
	}
	if strings.TrimSpace(result.ProductKey) != "" {
		state.ProductID = strings.TrimSpace(result.ProductKey)
	}
	if strings.TrimSpace(result.DeviceKey) != "" {
		state.DeviceName = strings.TrimSpace(result.DeviceKey)
	}

	if state.Pending > 0 {
		return nil, false
	}

	delete(a.requestStates, result.RequestID)
	return state, true
}
