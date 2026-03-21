//go:build autocert

package app

import (
	"crypto/tls"
	"log/slog"
	"net/http"

	"golang.org/x/crypto/acme/autocert"

	"github.com/gonglijing/xunjiFsu/internal/platform/config"
)

func listenAndServeWithAutoCert(server *http.Server, cfg *config.Config) error {
	if server == nil || cfg == nil {
		return http.ErrServerClosed
	}

	manager := &autocert.Manager{
		Cache:      autocert.DirCache(cfg.TLSCacheDir),
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(cfg.TLSDomain),
	}
	server.TLSConfig = &tls.Config{
		GetCertificate: manager.GetCertificate,
		MinVersion:     tls.VersionTLS12,
	}

	go func() {
		_ = http.ListenAndServe(":80", manager.HTTPHandler(nil))
	}()

	slog.Info("Starting HTTPS (auto-cert)", "addr", cfg.ListenAddr, "domain", cfg.TLSDomain)
	return server.ListenAndServeTLS("", "")
}
