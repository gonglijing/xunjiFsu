package adapters

import (
	"errors"
	"fmt"
	"strings"

	"github.com/gonglijing/xunjiFsu/internal/northbound/nbtype"
)

type configRequirement func(config map[string]interface{}) error

var configValidators = map[string][]configRequirement{
	nbtype.TypeMQTT: {
		requireStringValue("broker", "broker is required for MQTT adapter"),
		requireStringValue("topic", "topic is required for MQTT adapter"),
	},
	nbtype.TypeXunji: {
		requireServerURL("Xunji"),
	},
	nbtype.TypeSagoo: {
		requireStringValue("serverUrl", "serverUrl is required for Sagoo adapter"),
		requireStringValue("productKey", "productKey is required for Sagoo adapter"),
		requireStringValue("deviceKey", "deviceKey is required for Sagoo adapter"),
	},
	nbtype.TypePandaX: {
		requireServerURL("PandaX"),
		requireStringValue("username", "username is required for PandaX adapter"),
		requireGatewayModeTrue("PandaX"),
	},
	nbtype.TypeIThings: {
		requireServerURL("iThings"),
		requireStringValue("username", "username is required for iThings adapter"),
		requireStringValue("productKey", "productKey is required for iThings adapter"),
		requireStringValue("deviceKey", "deviceKey is required for iThings adapter"),
	},
}

// ValidateConfig 验证配置是否有效
func ValidateConfig(northboundType string, config map[string]interface{}) error {
	normalizedType := nbtype.Normalize(northboundType)
	validators, ok := configValidators[normalizedType]
	if !ok {
		return fmt.Errorf("unknown northbound type: %s", northboundType)
	}

	for _, validator := range validators {
		if err := validator(config); err != nil {
			return err
		}
	}

	return nil
}

func requireStringValue(key, errMessage string) configRequirement {
	return func(config map[string]interface{}) error {
		value, ok := config[key].(string)
		if !ok || strings.TrimSpace(value) == "" {
			return errors.New(errMessage)
		}
		return nil
	}
}

func requireServerURL(adapterName string) configRequirement {
	return func(config map[string]interface{}) error {
		if strings.TrimSpace(configServerURL(config)) == "" {
			return fmt.Errorf("serverUrl is required for %s adapter", adapterName)
		}
		return nil
	}
}

func requireGatewayModeTrue(adapterName string) configRequirement {
	return func(config map[string]interface{}) error {
		if _, ok := config["gatewayMode"]; ok && !pickConfigBool(config, true, "gatewayMode") {
			return fmt.Errorf("gatewayMode must be true for %s adapter", adapterName)
		}
		return nil
	}
}
