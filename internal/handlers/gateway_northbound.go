package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gonglijing/xunjiFsu/internal/database"
)

// SyncGatewayIdentityToNorthbound 将网关身份同步到 xunji/pandax/ithings 北向配置
func (h *Handler) SyncGatewayIdentityToNorthbound(w http.ResponseWriter, r *http.Request) {
	gw, err := database.GetGatewayConfig()
	if err != nil {
		writeServerErrorWithLog(w, apiErrGetGatewayConfigFailed, err)
		return
	}

	productKey := strings.TrimSpace(gw.ProductKey)
	deviceKey := strings.TrimSpace(gw.DeviceKey)
	if productKey == "" || deviceKey == "" {
		WriteBadRequestDef(w, apiErrGatewayIdentityRequired)
		return
	}

	updated, skipped, failed := h.syncGatewayIdentityToNorthboundTypes(productKey, deviceKey)
	if systemErr, ok := failed["_system"]; ok {
		writeServerErrorWithLog(w, apiErrSyncGatewayIdentityFailed, errors.New(systemErr))
		return
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
