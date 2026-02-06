package handlers

import (
	"net/http"
	"strings"

	northboundschema "github.com/gonglijing/xunjiFsu/internal/northbound/schema"
)

// GetNorthboundSchema returns schema metadata for specific northbound type.
func (h *Handler) GetNorthboundSchema(w http.ResponseWriter, r *http.Request) {
	nbType := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("type")))
	if nbType == "" {
		nbType = "xunji"
	}

	switch nbType {
	case "xunji":
		WriteSuccess(w, map[string]interface{}{
			"type":   "xunji",
			"fields": northboundschema.XunJiConfigSchema,
		})
	default:
		WriteBadRequest(w, "unsupported northbound type schema")
	}
}
