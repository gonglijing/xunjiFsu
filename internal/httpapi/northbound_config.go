package httpapi

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gonglijing/xunjiFsu/internal/models"
	"github.com/gonglijing/xunjiFsu/internal/northbound/nbtype"
	northboundschema "github.com/gonglijing/xunjiFsu/internal/northbound/schema"
)

const (
	defaultNorthboundUploadInterval = 5000
	defaultNorthboundKeepAlive      = 60
	defaultNorthboundTimeout        = 30
	defaultMQTTPort                 = 1883
)

type requiredFieldRule struct {
	fieldName string
	present   func(*models.NorthboundConfig) bool
}

var northboundRequiredFieldRules = map[string][]requiredFieldRule{
	nbtype.TypeMQTT: {
		{fieldName: "server_url", present: func(cfg *models.NorthboundConfig) bool { return strings.TrimSpace(cfg.ServerURL) != "" }},
		{fieldName: "topic", present: func(cfg *models.NorthboundConfig) bool { return strings.TrimSpace(cfg.Topic) != "" }},
	},
	nbtype.TypeXunji: {
		{fieldName: "server_url", present: func(cfg *models.NorthboundConfig) bool { return strings.TrimSpace(cfg.ServerURL) != "" }},
	},
	nbtype.TypeSagoo: {
		{fieldName: "server_url", present: func(cfg *models.NorthboundConfig) bool { return strings.TrimSpace(cfg.ServerURL) != "" }},
		{fieldName: "product_key", present: func(cfg *models.NorthboundConfig) bool { return strings.TrimSpace(cfg.ProductKey) != "" }},
		{fieldName: "device_key", present: func(cfg *models.NorthboundConfig) bool { return strings.TrimSpace(cfg.DeviceKey) != "" }},
	},
	nbtype.TypePandaX: {
		{fieldName: "server_url", present: func(cfg *models.NorthboundConfig) bool { return strings.TrimSpace(cfg.ServerURL) != "" }},
		{fieldName: "username", present: func(cfg *models.NorthboundConfig) bool { return strings.TrimSpace(cfg.Username) != "" }},
	},
	nbtype.TypeIThings: {
		{fieldName: "server_url", present: func(cfg *models.NorthboundConfig) bool { return strings.TrimSpace(cfg.ServerURL) != "" }},
		{fieldName: "username", present: func(cfg *models.NorthboundConfig) bool { return strings.TrimSpace(cfg.Username) != "" }},
	},
}

func normalizeNorthboundConfig(config *models.NorthboundConfig) {
	if config == nil {
		return
	}
	config.Name = strings.TrimSpace(config.Name)
	config.Type = normalizeNorthboundType(config.Type)
	config.ServerURL = strings.TrimSpace(config.ServerURL)
	config.Path = strings.TrimSpace(config.Path)
	config.Username = strings.TrimSpace(config.Username)
	config.Password = strings.TrimSpace(config.Password)
	config.ClientID = strings.TrimSpace(config.ClientID)
	config.Topic = strings.TrimSpace(config.Topic)
	config.AlarmTopic = strings.TrimSpace(config.AlarmTopic)
	config.ProductKey = strings.TrimSpace(config.ProductKey)
	config.DeviceKey = strings.TrimSpace(config.DeviceKey)
	if config.Enabled != 1 {
		config.Enabled = 0
	}
	if config.UploadInterval <= 0 {
		config.UploadInterval = defaultNorthboundUploadInterval
	}
	if config.Port <= 0 {
		if defaultPort := defaultNorthboundPort(config.Type); defaultPort > 0 {
			config.Port = defaultPort
		}
	}
	if config.QOS < 0 || config.QOS > 2 {
		config.QOS = 0
	}
	if config.KeepAlive <= 0 {
		config.KeepAlive = defaultNorthboundKeepAlive
	}
	if config.Timeout <= 0 {
		config.Timeout = defaultNorthboundTimeout
	}
}

func defaultNorthboundPort(nbType string) int {
	switch nbType {
	case "http":
		return 80
	case nbtype.TypeMQTT, nbtype.TypeXunji, nbtype.TypeSagoo, nbtype.TypePandaX, nbtype.TypeIThings:
		return defaultMQTTPort
	default:
		return 0
	}
}

func validateNorthboundConfig(config *models.NorthboundConfig) error {
	if config == nil {
		return fmt.Errorf("northbound config is nil")
	}
	if config.Name == "" {
		return fmt.Errorf("name is required")
	}
	if config.Type == "" {
		return fmt.Errorf("type is required")
	}

	config.Type = normalizeNorthboundType(config.Type)
	if !nbtype.IsSupported(config.Type) {
		return fmt.Errorf("invalid type: %s, must be one of: %s", config.Type, strings.Join(nbtype.SupportedTypes(), ", "))
	}
	if hasSchemaConfig(config) {
		if err := validateConfigBySchema(config.Type, config.Config); err != nil {
			return err
		}
	}
	return validateRequiredFields(config)
}

func normalizeNorthboundType(raw string) string {
	return nbtype.Normalize(raw)
}

func hasSchemaConfig(config *models.NorthboundConfig) bool {
	if config == nil {
		return false
	}
	trimmed := strings.TrimSpace(config.Config)
	return trimmed != "" && trimmed != "{}"
}

func validateRequiredFields(config *models.NorthboundConfig) error {
	if config == nil || hasSchemaConfig(config) {
		return nil
	}
	rules, ok := northboundRequiredFieldRules[config.Type]
	if !ok {
		return nil
	}
	typeName := nbtype.DisplayName(config.Type)
	for _, rule := range rules {
		if !rule.present(config) {
			return fmt.Errorf("%s or config is required for %s type", rule.fieldName, typeName)
		}
	}
	return nil
}

func validateConfigBySchema(nbType string, configJSON string) error {
	fields, ok := northboundschema.FieldsByType(nbType)
	if !ok {
		return nil
	}
	var config map[string]any
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return fmt.Errorf("config is invalid JSON: %w", err)
	}
	for _, field := range fields {
		if !field.Required {
			continue
		}
		value, exists := config[field.Key]
		if !exists || value == nil {
			return fmt.Errorf("%s is required", field.Label)
		}
		if text, ok := value.(string); ok && strings.TrimSpace(text) == "" {
			return fmt.Errorf("%s is required", field.Label)
		}
	}
	return nil
}

func NormalizeNorthboundConfigCompat(config *models.NorthboundConfig) {
	normalizeNorthboundConfig(config)
}

func ValidateNorthboundConfigCompat(config *models.NorthboundConfig) error {
	return validateNorthboundConfig(config)
}

func HasSchemaConfigCompat(config *models.NorthboundConfig) bool {
	return hasSchemaConfig(config)
}

func ValidateConfigBySchemaCompat(nbType string, configJSON string) error {
	return validateConfigBySchema(nbType, configJSON)
}
