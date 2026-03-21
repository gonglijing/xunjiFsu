package app

import (
	"net/http"

	"github.com/gonglijing/xunjiFsu/internal/auth"
)

func registerAPIRoutes(r *http.ServeMux, apiDeps *apiRouteDeps, authManager *auth.JWTManager) {
	api := http.NewServeMux()

	api.HandleFunc("GET /status", apiDeps.status.GetStatus)

	registerCollectorRoutes(api, apiDeps)
	registerDriverRoutes(api, apiDeps)
	registerDeviceRoutes(api, apiDeps)
	registerNorthboundRoutes(api, apiDeps)
	registerThresholdRoutes(api, apiDeps)
	registerAlarmRoutes(api, apiDeps)
	registerDataRoutes(api, apiDeps)
	registerUserRoutes(api, apiDeps)
	registerResourceRoutes(api, apiDeps)
	registerGatewayRoutes(api, apiDeps)
	registerDebugRoutes(api, apiDeps)

	r.Handle("/api/", authManager.RequireAuth(http.StripPrefix("/api", api)))
}
