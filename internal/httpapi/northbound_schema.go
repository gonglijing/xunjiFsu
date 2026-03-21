package httpapi

import (
	"fmt"
	"slices"
	"strings"

	northboundschema "github.com/gonglijing/xunjiFsu/internal/northbound/schema"
)

type northboundSchemaView struct {
	Type           string                   `json:"type"`
	SchemaVersion  string                   `json:"schemaVersion"`
	SupportedTypes []string                 `json:"supportedTypes"`
	Fields         []northboundschema.Field `json:"fields"`
}

func buildSchemaFieldViews(nbType string) []SchemaFieldView {
	fields, ok := northboundschema.FieldsByType(nbType)
	if !ok {
		return nil
	}
	schemaFields := make([]SchemaFieldView, 0, len(fields))
	for _, field := range fields {
		schemaFields = append(schemaFields, SchemaFieldView{
			Key:         field.Key,
			Label:       field.Label,
			Type:        string(field.Type),
			Required:    field.Required,
			Optional:    field.Optional,
			Default:     field.Default,
			Description: field.Description,
		})
	}
	return schemaFields
}

func loadNorthboundSchema(typeName string) (*northboundSchemaView, error) {
	nbType := normalizeNorthboundType(typeName)
	fields, ok := northboundschema.FieldsByType(nbType)
	if !ok {
		supported := append([]string(nil), northboundschema.SupportedNorthboundSchemaTypes...)
		slices.Sort(supported)
		return nil, fmt.Errorf("unsupported northbound type schema: %s (supported: %s)", nbType, strings.Join(supported, ","))
	}
	return &northboundSchemaView{
		Type:           nbType,
		SchemaVersion:  northboundschema.SagooSchemaVersion,
		SupportedTypes: northboundschema.SupportedNorthboundSchemaTypes,
		Fields:         fields,
	}, nil
}
