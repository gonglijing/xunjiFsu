package httpapi

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gonglijing/xunjiFsu/internal/platform/auth"
	"github.com/gonglijing/xunjiFsu/internal/service"
)

func parseGatewayRuntimeConfigRequest(w http.ResponseWriter, r *http.Request) (*service.GatewayRuntimeConfig, bool) {
	var payload service.GatewayRuntimeConfig
	if err := ParseRequest(r, &payload); err != nil {
		WriteBadRequestDef(w, apiErrInvalidRequestBody)
		return nil, false
	}
	return &payload, true
}

func parseRuntimeAuditLimit(r *http.Request) (int, error) {
	limit := 20
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			return 0, err
		}
		limit = parsed
	}
	return limit, nil
}

func runtimeConfigAuditActor(r *http.Request) service.RuntimeConfigActor {
	session := auth.SessionFromContext(r.Context())
	actor := service.RuntimeConfigActor{
		UserID:   0,
		Username: "unknown",
		SourceIP: strings.TrimSpace(r.RemoteAddr),
	}
	if session != nil {
		actor.UserID = session.UserID
		if strings.TrimSpace(session.Username) != "" {
			actor.Username = strings.TrimSpace(session.Username)
		}
	}
	if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); forwarded != "" {
		actor.SourceIP = forwarded
	}
	return actor
}
