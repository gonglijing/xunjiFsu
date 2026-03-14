package adapters

import (
	"strings"

	"github.com/gonglijing/xunjiFsu/internal/models"
)

func (a *IThingsAdapter) isInitialized() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.initialized
}

func (a *IThingsAdapter) resolveCollectDeviceName(data *models.CollectData, mode string) string {
	if data == nil {
		return ""
	}
	return resolveDeviceNameByMode(data.DeviceName, data.DeviceKey, mode)
}

func (a *IThingsAdapter) resolveAlarmDeviceName(alarm *models.AlarmPayload, mode string) string {
	if alarm == nil {
		return ""
	}
	return resolveDeviceNameByMode(alarm.DeviceName, alarm.DeviceKey, mode)
}

func resolveDeviceNameByMode(deviceName, deviceKey, mode string) string {
	name := strings.TrimSpace(deviceName)
	key := strings.TrimSpace(deviceKey)
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "devicename", "device_name":
		return pickFirstNonEmpty2(name, key)
	case "devicekey", "device_key", "":
		return pickFirstNonEmpty2(key, name)
	default:
		return pickFirstNonEmpty2(key, name)
	}
}

func parseIThingsDownTopic(topic string) (topicType, productID, deviceName string) {
	parts := splitTopic(topic)
	if len(parts) < 5 {
		return "", "", ""
	}
	if parts[0] != "$thing" || parts[1] != "down" {
		return "", "", ""
	}

	topicType = strings.TrimSpace(parts[2])
	if topicType == "" {
		return "", "", ""
	}

	if len(parts) >= 6 && parts[3] == "custom" {
		productID = strings.TrimSpace(parts[4])
		deviceName = strings.TrimSpace(parts[5])
	} else {
		productID = strings.TrimSpace(parts[3])
		deviceName = strings.TrimSpace(parts[4])
	}

	if productID == "" || deviceName == "" {
		return "", "", ""
	}

	return topicType, productID, deviceName
}

func renderIThingsTopic(template, productID, deviceName string) string {
	topic := strings.TrimSpace(template)
	if topic == "" {
		return ""
	}
	topic = strings.ReplaceAll(topic, "{productID}", strings.TrimSpace(productID))
	topic = strings.ReplaceAll(topic, "{deviceName}", strings.TrimSpace(deviceName))
	return topic
}

func (a *IThingsAdapter) nextID(prefix string) string {
	return nextPrefixedID(prefix, &a.seq)
}
