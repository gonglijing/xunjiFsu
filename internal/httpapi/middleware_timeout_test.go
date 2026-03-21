package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestTimeoutMiddleware_Timeout(t *testing.T) {
	mw := TimeoutMiddleware(&TimeoutConfig{ReadTimeout: 20 * time.Millisecond})
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(60 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/slow", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d, want=%d", rr.Code, http.StatusServiceUnavailable)
	}
}

func TestTimeoutMiddleware_NoTimeoutWhenDisabled(t *testing.T) {
	mw := TimeoutMiddleware(&TimeoutConfig{ReadTimeout: 0})
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	}))

	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Fatalf("status=%d, want=%d", rr.Code, http.StatusAccepted)
	}
}
