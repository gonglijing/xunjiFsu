//go:build !autocert

package app

import (
	"net/http"
	"strings"
	"testing"

	"github.com/gonglijing/xunjiFsu/internal/config"
)

func TestListenAndServeWithAutoCert_DisabledBuild(t *testing.T) {
	err := listenAndServeWithAutoCert(&http.Server{}, &config.Config{TLSDomain: "example.com", TLSAuto: true})
	if err == nil {
		t.Fatal("expected error for disabled autocert build")
	}
	if !strings.Contains(err.Error(), "-tags autocert") {
		t.Fatalf("unexpected error: %v", err)
	}
}
