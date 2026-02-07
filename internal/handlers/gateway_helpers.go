package handlers

import (
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

func (h *Handler) syncGatewayIdentityToNorthboundTypes(productKey, deviceKey string) ([]string, []string, map[string]string) {
	configs, err := database.GetAllNorthboundConfigs()
	if err != nil {
		return nil, nil, map[string]string{"_system": err.Error()}
	}

	updated := make([]string, 0)
	skipped := make([]string, 0)
	failed := make(map[string]string)

	for _, cfg := range configs {
		nbType := strings.ToLower(strings.TrimSpace(cfg.Type))
		if cfg == nil || (nbType != "xunji" && nbType != "pandax" && nbType != "ithings") {
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
	oldPK := strings.TrimSpace(current.ProductKey)
	oldDK := strings.TrimSpace(current.DeviceKey)
	newPK := strings.TrimSpace(productKey)
	newDK := strings.TrimSpace(deviceKey)

	if oldPK == newPK && oldDK == newDK {
		return nil, false, nil
	}

	next := *current
	next.ProductKey = newPK
	next.DeviceKey = newDK
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
