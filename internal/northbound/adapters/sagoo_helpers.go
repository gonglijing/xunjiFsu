package adapters

import "strings"

func sagooSysTopic(productKey, deviceKey, suffix string) string {
	if suffix == "" {
		return "/sys/" + productKey + "/" + deviceKey
	}
	return "/sys/" + productKey + "/" + deviceKey + "/" + suffix
}

func splitTopic(topic string) []string {
	trimmed := strings.TrimSpace(topic)
	if trimmed == "" {
		return make([]string, 0)
	}

	parts := make([]string, 0, 1+strings.Count(trimmed, "/"))
	segmentStart := -1
	for i := 0; i < len(trimmed); i++ {
		if trimmed[i] == '/' {
			if segmentStart >= 0 {
				parts = append(parts, trimmed[segmentStart:i])
				segmentStart = -1
			}
			continue
		}
		if segmentStart < 0 {
			segmentStart = i
		}
	}
	if segmentStart >= 0 {
		parts = append(parts, trimmed[segmentStart:])
	}

	return parts
}

func extractIdentity(topic string) (string, string, bool) {
	parts := splitTopic(topic)
	if len(parts) < 4 || parts[0] != "sys" {
		return "", "", false
	}
	return parts[1], parts[2], true
}

func extractCommandProperties(params map[string]interface{}) (map[string]interface{}, string, string) {
	identityPK, identityDK := "", ""
	if identity, ok := mapFromAny(params["identity"]); ok {
		identityPK, identityDK = parseIdentityMap(identity)
	}

	if props, ok := mapFromAny(params["properties"]); ok {
		return props, identityPK, identityDK
	}

	if sub, ok := mapFromAnyByKey2(params, "sub_device", "subDevice"); ok {
		if identity, ok := mapFromAny(sub["identity"]); ok {
			identityPK, identityDK = parseIdentityMap(identity)
		}
		if props, ok := mapFromAny(sub["properties"]); ok {
			return props, identityPK, identityDK
		}
	}

	if list, ok := interfaceSliceByKey2(params, "sub_devices", "subDevices"); ok && len(list) > 0 {
		if item, ok := mapFromAny(list[0]); ok {
			if identity, ok := mapFromAny(item["identity"]); ok {
				identityPK, identityDK = parseIdentityMap(identity)
			}
			if props, ok := mapFromAny(item["properties"]); ok {
				return props, identityPK, identityDK
			}
		}
	}

	directProperties := make(map[string]interface{}, len(params))
	for key, raw := range params {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" || isReservedCommandKeyNormalized(strings.ToLower(trimmedKey)) {
			continue
		}
		switch raw.(type) {
		case map[string]interface{}, []interface{}:
			continue
		}
		directProperties[trimmedKey] = raw
	}
	if len(directProperties) > 0 {
		return directProperties, identityPK, identityDK
	}

	return nil, identityPK, identityDK
}

func parseIdentityMap(identity map[string]interface{}) (string, string) {
	if identity == nil {
		return "", ""
	}
	productKey, deviceKey := "", ""
	if value, ok := identity["productKey"].(string); ok {
		productKey = strings.TrimSpace(value)
	}
	if value, ok := identity["deviceKey"].(string); ok {
		deviceKey = strings.TrimSpace(value)
	}
	return productKey, deviceKey
}

func isReservedCommandKey(key string) bool {
	normalized := strings.ToLower(strings.TrimSpace(key))
	return isReservedCommandKeyNormalized(normalized)
}

func isReservedCommandKeyNormalized(normalized string) bool {
	switch normalized {
	case "id", "method", "version", "params",
		"identity", "properties", "events",
		"sub_device", "subdevice", "sub_devices", "subdevices":
		return true
	default:
		return false
	}
}
