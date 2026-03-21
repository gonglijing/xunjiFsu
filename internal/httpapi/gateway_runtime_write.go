package httpapi

import (
	"net/http"
)

func (api *GatewayAPI) UpdateGatewayRuntimeConfig(w http.ResponseWriter, r *http.Request) {
	if api.runtimeService == nil {
		writeServerErrorWithLog(w, errUpdateRuntimeConfigFailed, nil)
		return
	}

	payload, ok := parseGatewayRuntimeConfigRequest(w, r)
	if !ok {
		return
	}

	changes, err := api.runtimeService.ApplyRuntimeConfig(payload)
	if err != nil {
		WriteBadRequestCode(w, errUpdateRuntimeConfigFailed.Code, errUpdateRuntimeConfigFailed.Message+": "+err.Error())
		return
	}

	if len(changes) > 0 {
		_ = api.runtimeService.RecordRuntimeConfigAudit(runtimeConfigAuditActor(r), changes)
	}

	api.GetGatewayRuntimeConfig(w, r)
}
