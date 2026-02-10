package handlers

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGzipMiddleware_CompressesWhenAccepted(t *testing.T) {
	h := GzipMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("hello world"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if got := rr.Header().Get("Content-Encoding"); got != "gzip" {
		t.Fatalf("Content-Encoding=%q, want gzip", got)
	}

	gr, err := gzip.NewReader(strings.NewReader(rr.Body.String()))
	if err != nil {
		t.Fatalf("gzip.NewReader error: %v", err)
	}
	defer gr.Close()

	body, err := io.ReadAll(gr)
	if err != nil {
		t.Fatalf("read gzip body error: %v", err)
	}
	if string(body) != "hello world" {
		t.Fatalf("body=%q, want=%q", string(body), "hello world")
	}
}

func TestGzipMiddleware_PassThroughWhenNotAccepted(t *testing.T) {
	h := GzipMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("plain text"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if got := rr.Header().Get("Content-Encoding"); got != "" {
		t.Fatalf("Content-Encoding=%q, want empty", got)
	}
	if rr.Body.String() != "plain text" {
		t.Fatalf("body=%q, want=%q", rr.Body.String(), "plain text")
	}
}

func TestHeaderContainsToken(t *testing.T) {
	if !headerContainsToken("br, gzip", "gzip") {
		t.Fatal("expected gzip token found")
	}
	if !headerContainsToken("br, gzip;q=1.0", "gzip") {
		t.Fatal("expected gzip token with q parameter found")
	}
	if headerContainsToken("br, deflate", "gzip") {
		t.Fatal("unexpected gzip token found")
	}
}

func TestAppendVaryToken_NoDuplicate(t *testing.T) {
	got := appendVaryToken("Accept-Encoding", "accept-encoding")
	if got != "Accept-Encoding" {
		t.Fatalf("vary=%q, want=%q", got, "Accept-Encoding")
	}
}
