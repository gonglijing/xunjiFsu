package handlers

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/gonglijing/xunjiFsu/internal/database"
	"github.com/gonglijing/xunjiFsu/internal/models"
)

const defaultGatewayName = "HuShu智能网关"

func normalizeGatewayConfigInput(cfg *models.GatewayConfig) {
	if cfg == nil {
		return
	}
	cfg.ProductKey = strings.TrimSpace(cfg.ProductKey)
	cfg.DeviceKey = strings.TrimSpace(cfg.DeviceKey)
	cfg.GatewayName = strings.TrimSpace(cfg.GatewayName)
	if cfg.GatewayName == "" {
		cfg.GatewayName = defaultGatewayName
	}
}

func toDatabaseGatewayConfig(cfg *models.GatewayConfig) *database.GatewayConfig {
	if cfg == nil {
		return nil
	}
	return &database.GatewayConfig{
		ID:          cfg.ID,
		ProductKey:  cfg.ProductKey,
		DeviceKey:   cfg.DeviceKey,
		GatewayName: cfg.GatewayName,
	}
}

func (h *Handler) syncGatewayIdentityToXunjiNorthbound(productKey, deviceKey string) ([]string, []string, map[string]string) {
	configs, err := database.GetAllNorthboundConfigs()
	if err != nil {
		return nil, nil, map[string]string{"_system": err.Error()}
	}

	updated := make([]string, 0)
	skipped := make([]string, 0)
	failed := make(map[string]string)

	for _, cfg := range configs {
		if cfg == nil || strings.ToLower(strings.TrimSpace(cfg.Type)) != "xunji" {
			continue
		}

		nextCfg, changed, err := buildNorthboundIdentityPatch(cfg, productKey, deviceKey)
		if err != nil {
			failed[cfg.Name] = "配置 JSON 无效"
			continue
		}
		if !changed {
			skipped = append(skipped, cfg.Name)
			continue
		}

		if err := h.applyNorthboundIdentityPatch(cfg, nextCfg); err != nil {
			failed[cfg.Name] = err.Error()
			continue
		}
		updated = append(updated, cfg.Name)
	}

	sort.Strings(updated)
	sort.Strings(skipped)
	return updated, skipped, failed
}

func buildNorthboundIdentityPatch(current *models.NorthboundConfig, productKey, deviceKey string) (*models.NorthboundConfig, bool, error) {
	raw := make(map[string]interface{})
	if err := json.Unmarshal([]byte(current.Config), &raw); err != nil {
		return nil, false, err
	}

	oldPK, _ := raw["productKey"].(string)
	oldDK, _ := raw["deviceKey"].(string)
	if strings.TrimSpace(oldPK) == productKey && strings.TrimSpace(oldDK) == deviceKey {
		return nil, false, nil
	}

	raw["productKey"] = productKey
	raw["deviceKey"] = deviceKey

	b, err := json.Marshal(raw)
	if err != nil {
		return nil, false, err
	}

	next := *current
	next.Config = string(b)
	return &next, true, nil
}

func (h *Handler) applyNorthboundIdentityPatch(prevCfg, nextCfg *models.NorthboundConfig) error {
	if prevCfg == nil || nextCfg == nil {
		return fmt.Errorf("invalid northbound config")
	}

	if err := database.UpdateNorthboundConfig(nextCfg); err != nil {
		return err
	}

	if err := h.rebuildNorthboundRuntime(nextCfg); err != nil {
		_ = database.UpdateNorthboundConfig(prevCfg)
		_ = h.rebuildNorthboundRuntime(prevCfg)
		return fmt.Errorf("运行时重载失败: %w", err)
	}

	return nil
}
