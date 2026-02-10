package nbtype

import "strings"

const (
	TypeMQTT    = "mqtt"
	TypePandaX  = "pandax"
	TypeIThings = "ithings"
	TypeSagoo   = "sagoo"
)

var supportedTypes = []string{TypeMQTT, TypePandaX, TypeIThings, TypeSagoo}

func Normalize(raw string) string {
	return strings.ToLower(strings.TrimSpace(raw))
}

func SupportedTypes() []string {
	return append([]string(nil), supportedTypes...)
}

func IsSupported(raw string) bool {
	switch Normalize(raw) {
	case TypeMQTT, TypePandaX, TypeIThings, TypeSagoo:
		return true
	default:
		return false
	}
}

func DisplayName(raw string) string {
	switch Normalize(raw) {
	case TypeMQTT:
		return "MQTT"
	case TypePandaX:
		return "PandaX"
	case TypeIThings:
		return "iThings"
	case TypeSagoo:
		return "Sagoo"
	default:
		return strings.TrimSpace(raw)
	}
}
