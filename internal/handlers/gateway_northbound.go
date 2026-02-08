package handlers

import (
	"errors"
	"net/http"
)

// SyncGatewayIdentityToNorthbound 将网关身份同步到 xunji/pandax/ithings 北向配置
func (h *Handler) SyncGatewayIdentityToNorthbound(w http.ResponseWriter, r *http.Request) {
	gw, ok := loadGatewayConfigOrWriteServerError(w)
	if !ok {
		return
	}

	productKey, deviceKey, ok := extractGatewayIdentity(gw)
	if !ok {
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
