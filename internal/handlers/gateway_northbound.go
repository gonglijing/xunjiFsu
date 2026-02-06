package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gonglijing/xunjiFsu/internal/database"
)

// SyncGatewayIdentityToNorthbound 将网关身份同步到 xunji 北向配置
func (h *Handler) SyncGatewayIdentityToNorthbound(w http.ResponseWriter, r *http.Request) {
	gw, err := database.GetGatewayConfig()
	if err != nil {
		WriteServerError(w, err.Error())
		return
	}

	productKey := strings.TrimSpace(gw.ProductKey)
	deviceKey := strings.TrimSpace(gw.DeviceKey)
	if productKey == "" || deviceKey == "" {
		WriteBadRequest(w, "请先在网关配置中设置 product_key 和 device_key")
		return
	}

	configs, err := database.GetAllNorthboundConfigs()
	if err != nil {
		WriteServerError(w, err.Error())
		return
	}

	updated := make([]string, 0)
	skipped := make([]string, 0)
	failed := make(map[string]string)

	for _, cfg := range configs {
		if cfg == nil || strings.ToLower(strings.TrimSpace(cfg.Type)) != "xunji" {
			continue
		}

		raw := make(map[string]interface{})
		if err := json.Unmarshal([]byte(cfg.Config), &raw); err != nil {
			failed[cfg.Name] = "配置 JSON 无效"
			continue
		}

		oldPK, _ := raw["productKey"].(string)
		oldDK, _ := raw["deviceKey"].(string)
		if strings.TrimSpace(oldPK) == productKey && strings.TrimSpace(oldDK) == deviceKey {
			skipped = append(skipped, cfg.Name)
			continue
		}

		raw["productKey"] = productKey
		raw["deviceKey"] = deviceKey
		b, err := json.Marshal(raw)
		if err != nil {
			failed[cfg.Name] = "配置序列化失败"
			continue
		}

		next := *cfg
		next.Config = string(b)
		if err := database.UpdateNorthboundConfig(&next); err != nil {
			failed[cfg.Name] = err.Error()
			continue
		}

		if err := h.rebuildNorthboundRuntime(&next); err != nil {
			failed[cfg.Name] = "运行时重载失败: " + err.Error()
			continue
		}

		updated = append(updated, cfg.Name)
	}

	WriteSuccess(w, map[string]interface{}{
		"gateway": map[string]string{
			"product_key": productKey,
			"device_key":  deviceKey,
		},
		"updated": updated,
		"skipped": skipped,
		"failed":  failed,
	})
}
