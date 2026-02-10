package handlers

import (
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/gonglijing/xunjiFsu/internal/northbound/nbtype"
	northboundschema "github.com/gonglijing/xunjiFsu/internal/northbound/schema"
)

// GetNorthboundSchema returns schema metadata for specific northbound type.
func (h *Handler) GetNorthboundSchema(w http.ResponseWriter, r *http.Request) {
	nbType := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("type")))
	if nbType == "" {
		nbType = "pandax"
	}
	nbType = nbtype.Normalize(nbType)

	fields, ok := northboundschema.FieldsByType(nbType)
	if !ok {
		supported := append([]string(nil), northboundschema.SupportedNorthboundSchemaTypes...)
		sort.Strings(supported)
		WriteBadRequest(w, fmt.Sprintf("unsupported northbound type schema: %s (supported: %s)", nbType, strings.Join(supported, ",")))
		return
	}

	response := map[string]interface{}{
		"type":           nbType,
		"schemaVersion":  northboundschema.XunJiSchemaVersion,
		"supportedTypes": northboundschema.SupportedNorthboundSchemaTypes,
		"fields":         fields,
	}

	WriteSuccess(w, response)
}
