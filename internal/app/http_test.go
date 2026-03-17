package app

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCorsMiddleware_AllowedOrigin(t *testing.T) {
	mw := corsMiddleware([]string{"http://localhost:8080"})
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("Origin", "http://localhost:8080")
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, want=%d", rr.Code, http.StatusOK)
	}
	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:8080" {
		t.Fatalf("allow-origin=%q, want=%q", got, "http://localhost:8080")
	}
}

func TestCorsMiddleware_RejectUnknownOrigin(t *testing.T) {
	mw := corsMiddleware([]string{"http://localhost:8080"})
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("Origin", "https://evil.example.com")
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("allow-origin=%q, want empty", got)
	}
}

func TestCorsMiddleware_Preflight(t *testing.T) {
	mw := corsMiddleware([]string{"*"})
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodOptions, "/api/test", nil)
	req.Header.Set("Origin", "http://localhost:8080")
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("status=%d, want=%d", rr.Code, http.StatusNoContent)
	}
}

func TestAppendVaryHeader_NoDuplicate(t *testing.T) {
	got := appendVaryHeader("Origin", "origin")
	if got != "Origin" {
		t.Fatalf("vary=%q, want=%q", got, "Origin")
	}
}

func TestBuildAllowedOriginSet(t *testing.T) {
	allowSet, allowAll := buildAllowedOriginSet([]string{" http://a.example ", "", "*", "http://b.example"})

	if !allowAll {
		t.Fatal("allowAll = false, want true")
	}
	if _, ok := allowSet["http://a.example"]; !ok {
		t.Fatalf("missing trimmed origin in allowSet: %#v", allowSet)
	}
	if _, ok := allowSet["http://b.example"]; !ok {
		t.Fatalf("missing second origin in allowSet: %#v", allowSet)
	}
}

func TestShouldServeSPA(t *testing.T) {
	tests := []struct {
		name   string
		method string
		path   string
		want   bool
	}{
		{name: "spa page", method: http.MethodGet, path: "/dashboard", want: true},
		{name: "api path", method: http.MethodGet, path: "/api/devices", want: false},
		{name: "static path", method: http.MethodGet, path: "/static/app.js", want: false},
		{name: "health path", method: http.MethodGet, path: "/health", want: false},
		{name: "post request", method: http.MethodPost, path: "/dashboard", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			if got := shouldServeSPA(req); got != tt.want {
				t.Fatalf("shouldServeSPA(%s %s) = %v, want %v", tt.method, tt.path, got, tt.want)
			}
		})
	}
}
