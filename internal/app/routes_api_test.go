package app

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gonglijing/xunjiFsu/internal/auth"
)

func TestRegisterAPIRoutes_Patterns(t *testing.T) {
	router := http.NewServeMux()

	registerAPIRoutes(
		router,
		&apiRouteDeps{},
		auth.NewJWTManager([]byte("0123456789abcdef0123456789abcdef")),
	)

	testCases := []struct {
		method      string
		path        string
		wantPattern string
	}{
		{method: http.MethodGet, path: "/api/devices", wantPattern: "/api/"},
		{method: http.MethodGet, path: "/api/devices/runtime", wantPattern: "/api/"},
		{method: http.MethodPost, path: "/api/devices/1/execute", wantPattern: "/api/"},
		{method: http.MethodGet, path: "/api/devices/1/writables", wantPattern: "/api/"},
		{method: http.MethodGet, path: "/api/drivers", wantPattern: "/api/"},
		{method: http.MethodGet, path: "/api/drivers/runtime", wantPattern: "/api/"},
		{method: http.MethodPost, path: "/api/debug/modbus/serial", wantPattern: "/api/"},
		{method: http.MethodPost, path: "/api/debug/modbus/tcp", wantPattern: "/api/"},
		{method: http.MethodGet, path: "/api/gateway/config", wantPattern: "/api/"},
		{method: http.MethodPut, path: "/api/gateway/runtime", wantPattern: "/api/"},
		{method: http.MethodGet, path: "/api/resources", wantPattern: "/api/"},
		{method: http.MethodGet, path: "/api/users", wantPattern: "/api/"},
		{method: http.MethodGet, path: "/api/thresholds", wantPattern: "/api/"},
		{method: http.MethodGet, path: "/api/alarms", wantPattern: "/api/"},
	}

	for _, tc := range testCases {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			_, pattern := router.Handler(req)
			if pattern != tc.wantPattern {
				t.Fatalf("pattern=%q, want=%q", pattern, tc.wantPattern)
			}
		})
	}
}
