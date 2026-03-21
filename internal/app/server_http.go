package app

import (
	"fmt"
	"net/http"

	"github.com/gonglijing/xunjiFsu/internal/config"
	"github.com/gonglijing/xunjiFsu/internal/logger"
)

type httpServerMode string

const (
	httpServerModePlain     httpServerMode = "plain_http"
	httpServerModeManualTLS httpServerMode = "manual_tls"
	httpServerModeAutoTLS   httpServerMode = "auto_tls"
)

func buildHTTPServer(cfg *config.Config, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:         cfg.ListenAddr,
		Handler:      handler,
		ReadTimeout:  cfg.HTTPReadTimeout,
		WriteTimeout: cfg.HTTPWriteTimeout,
		IdleTimeout:  cfg.HTTPIdleTimeout,
	}
}

func serveHTTPServer(server *http.Server, cfg *config.Config) error {
	switch resolveHTTPServerMode(cfg) {
	case httpServerModeAutoTLS:
		logger.Info("Starting HTTPS with automatic certificate management", "addr", cfg.ListenAddr, "domain", cfg.TLSDomain)
		if err := listenAndServeWithAutoCert(server, cfg); err != nil {
			return wrapHTTPServerError(err)
		}
	case httpServerModeManualTLS:
		logger.Info("Starting HTTPS", "addr", cfg.ListenAddr, "cert", cfg.TLSCertFile)
		if err := server.ListenAndServeTLS(cfg.TLSCertFile, cfg.TLSKeyFile); err != nil {
			return wrapHTTPServerError(err)
		}
	default:
		logger.Info("Starting HTTP", "addr", cfg.ListenAddr)
		if err := server.ListenAndServe(); err != nil {
			return wrapHTTPServerError(err)
		}
	}
	return nil
}

func resolveHTTPServerMode(cfg *config.Config) httpServerMode {
	if cfg != nil && cfg.TLSAuto && cfg.TLSDomain != "" {
		return httpServerModeAutoTLS
	}
	if cfg != nil && cfg.TLSCertFile != "" && cfg.TLSKeyFile != "" {
		return httpServerModeManualTLS
	}
	return httpServerModePlain
}

func wrapHTTPServerError(err error) error {
	if err == nil || err == http.ErrServerClosed {
		return nil
	}
	return fmt.Errorf("server error: %w", err)
}
