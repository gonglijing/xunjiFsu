//go:build !no_paho_mqtt

package adapters

import (
	"strings"

	"github.com/gonglijing/xunjiFsu/internal/database"
	"github.com/gonglijing/xunjiFsu/internal/models"
)

func resolveXunjiGatewayName(configGatewayName, fallback string) string {
	if name := sanitizeXunjiTopicSegment(configGatewayName); name != "" {
		return name
	}
	if database.ParamDB != nil {
		if name := sanitizeXunjiTopicSegment(database.GetGatewayName()); name != "" {
			return name
		}
	}
	if name := sanitizeXunjiTopicSegment(fallback); name != "" {
		return name
	}
	return "gateway"
}

func renderXunjiTopic(topicTemplate, gatewayName string) string {
	topic := strings.TrimSpace(topicTemplate)
	if topic == "" {
		topic = defaultXunjiTopicTemplate
	}
	gw := sanitizeXunjiTopicSegment(gatewayName)
	if gw == "" {
		gw = "gateway"
	}

	topic = strings.ReplaceAll(topic, "{gatewayname}", gw)
	topic = strings.ReplaceAll(topic, "{gatewayName}", gw)
	topic = strings.ReplaceAll(topic, "${gatewayname}", gw)
	topic = strings.ReplaceAll(topic, "${gatewayName}", gw)
	topic = strings.TrimSpace(topic)

	if strings.TrimRight(topic, "/") == "v1/gateway" {
		return "v1/gateway/" + gw
	}
	return topic
}

func sanitizeXunjiTopicSegment(raw string) string {
	name := strings.TrimSpace(raw)
	if name == "" {
		return ""
	}
	replacer := strings.NewReplacer("/", "_", "+", "_", "#", "_", " ", "_")
	name = strings.TrimSpace(replacer.Replace(name))
	return name
}

func resolveXunjiSubToken(data *models.CollectData, mode string) string {
	if data == nil {
		return defaultDeviceToken(0)
	}

	pk := strings.TrimSpace(data.ProductKey)
	name := pickFirstNonEmpty2(data.DeviceName, data.DeviceKey)
	dk := pickFirstNonEmpty2(data.DeviceKey, data.DeviceName)

	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "devicekey", "device_key":
		if dk != "" {
			return dk
		}
	case "product_devicekey", "product_device_key":
		if pk != "" && dk != "" {
			return pk + "_" + dk
		}
	case "product_devicename", "product_device_name":
		if pk != "" && name != "" {
			return pk + "_" + name
		}
	default:
		if name != "" {
			return name
		}
	}

	if dk != "" {
		return dk
	}
	if name != "" {
		return name
	}
	return defaultDeviceToken(data.DeviceID)
}
