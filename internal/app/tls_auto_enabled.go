//go:build autocert

package app

import (
	"crypto/tls"
	"net/http"

	"github.com/gonglijing/xunjiFsu/internal/config"
	"github.com/gonglijing/xunjiFsu/internal/logger"
	"golang.org/x/crypto/acme/autocert"
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

	logger.Info("Starting HTTPS (auto-cert)", "addr", cfg.ListenAddr, "domain", cfg.TLSDomain)
	return server.ListenAndServeTLS("", "")
}
