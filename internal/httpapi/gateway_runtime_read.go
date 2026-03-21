package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gonglijing/xunjiFsu/internal/database"
	"github.com/gonglijing/xunjiFsu/internal/service"
)

func (api *GatewayAPI) GetGatewayRuntimeConfig(w http.ResponseWriter, r *http.Request) {
	WriteSuccess(w, api.runtimeService.RuntimeConfigView())
}

func (api *GatewayAPI) GetGatewayRuntimeAudits(w http.ResponseWriter, r *http.Request) {
	limit, err := parseRuntimeAuditLimit(r)
	if err != nil {
		WriteBadRequestCode(w, errUpdateRuntimeConfigFailed.Code, errUpdateRuntimeConfigFailed.Message+": invalid limit")
		return
	}

	items, err := api.runtimeService.ListRuntimeConfigAudits(limit)
	if err != nil {
		writeServerErrorWithLog(w, errListRuntimeConfigAuditsFailed, err)
		return
	}

	WriteSuccess(w, buildRuntimeAuditViews(items))
}

func buildRuntimeAuditViews(items []*database.RuntimeConfigAudit) []*runtimeConfigAuditView {
	if len(items) == 0 {
		return []*runtimeConfigAuditView{}
	}

	views := make([]*runtimeConfigAuditView, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}

		view := &runtimeConfigAuditView{
			ID:               item.ID,
			OperatorUserID:   item.OperatorUserID,
			OperatorUsername: item.OperatorUsername,
			SourceIP:         item.SourceIP,
			CreatedAt:        item.CreatedAt,
		}

		raw := strings.TrimSpace(item.Changes)
		if raw != "" {
			parsed := make(map[string]service.RuntimeConfigChange)
			if err := json.Unmarshal([]byte(raw), &parsed); err == nil && len(parsed) > 0 {
				view.Changes = parsed
			} else {
				view.ChangesRaw = item.Changes
			}
		}

		views = append(views, view)
	}

	return views
}
