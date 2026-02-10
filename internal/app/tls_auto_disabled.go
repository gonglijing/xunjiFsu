//go:build !autocert

package app

import (
	"fmt"
	"net/http"

	"github.com/gonglijing/xunjiFsu/internal/config"
)

func listenAndServeWithAutoCert(server *http.Server, cfg *config.Config) error {
	if server == nil || cfg == nil {
		return fmt.Errorf("invalid auto-cert config")
	}
	return fmt.Errorf("tls auto-cert is not enabled in this build; rebuild with -tags autocert")
}
