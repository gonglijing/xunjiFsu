package adapters

import (
	"encoding/json"
	"sort"
	"strings"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gonglijing/xunjiFsu/internal/models"
)

func (a *SagooAdapter) handlePropertySet(_ mqtt.Client, message mqtt.Message) {
	pk, dk, ok := extractIdentity(message.Topic())
	if !ok {
		return
	}

	var req struct {
		Id       string                 `json:"id"`
		Params   map[string]interface{} `json:"params"`
		Identity map[string]interface{} `json:"identity"`
	}
	if err := json.Unmarshal(message.Payload(), &req); err != nil {
		return
	}

	identityPK, identityDK := parseIdentityMap(req.Identity)
	a.enqueueCommandFromPropertySet(pk, dk, req.Id, req.Params, identityPK, identityDK)

	resp := map[string]interface{}{
		"code":    200,
		"data":    req.Params,
		"id":      req.Id,
		"message": "success",
		"version": "1.0.0",
	}
	respBody, _ := json.Marshal(resp)
	_ = a.publish(sagooSysTopic(pk, dk, "thing/service/property/set_reply"), respBody)
}

func (a *SagooAdapter) handleServiceCall(_ mqtt.Client, message mqtt.Message) {
	parts := splitTopic(message.Topic())
	if len(parts) != 7 {
		return
	}

	pk, dk, svc := parts[1], parts[2], parts[6]
	if strings.HasSuffix(svc, "reply") || svc == "property" {
		return
	}

	var req struct {
		Id       string                 `json:"id"`
		Params   map[string]interface{} `json:"params"`
		Identity map[string]interface{} `json:"identity"`
	}
	if err := json.Unmarshal(message.Payload(), &req); err != nil {
		return
	}

	identityPK, identityDK := parseIdentityMap(req.Identity)
	if len(req.Params) > 0 {
		a.enqueueCommandFromPropertySet(pk, dk, req.Id, req.Params, identityPK, identityDK)
	}

	resp := map[string]interface{}{
		"code":    200,
		"data":    req.Params,
		"id":      req.Id,
		"message": "success",
		"version": "1.0.0",
	}
	respBody, _ := json.Marshal(resp)
	_ = a.publish(sagooSysTopic(pk, dk, "thing/service/"+svc+"_reply"), respBody)
}

func (a *SagooAdapter) handleConfigPush(_ mqtt.Client, message mqtt.Message) {
	pk, dk, ok := extractIdentity(message.Topic())
	if !ok {
		return
	}

	var req struct {
		Id string `json:"id"`
	}
	if err := json.Unmarshal(message.Payload(), &req); err != nil {
		return
	}

	resp := map[string]interface{}{
		"code": 200,
		"data": map[string]interface{}{},
		"id":   req.Id,
	}
	respBody, _ := json.Marshal(resp)
	_ = a.publish(sagooSysTopic(pk, dk, "thing/config/push/reply"), respBody)
}

func (a *SagooAdapter) enqueueCommandFromPropertySet(defaultPK, defaultDK, requestID string, params map[string]interface{}, rootIdentityPK, rootIdentityDK string) {
	properties, identityPK, identityDK := extractCommandProperties(params)
	if len(properties) == 0 {
		return
	}

	pk := pickFirstNonEmpty3(rootIdentityPK, identityPK, defaultPK)
	dk := pickFirstNonEmpty3(rootIdentityDK, identityDK, defaultDK)
	if pk == "" || dk == "" {
		return
	}

	keys := make([]string, 0, len(properties))
	for key := range properties {
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

	a.commandMu.Lock()
	defer a.commandMu.Unlock()

	commands := make([]*models.NorthboundCommand, 0, len(keys))
	for _, key := range keys {
		raw := properties[key]
		commands = append(commands, &models.NorthboundCommand{
			RequestID:  requestID,
			ProductKey: pk,
			DeviceKey:  dk,
			FieldName:  key,
			Value:      stringifyAny(raw),
			Source:     "sagoo.property.set",
		})
	}
	a.commandQueue = appendCommandQueueWithCap(a.commandQueue, commands, a.commandCap)
}
