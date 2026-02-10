package nbtype

import "strings"

const (
	TypeMQTT    = "mqtt"
	TypePandaX  = "pandax"
	TypeIThings = "ithings"
	TypeSagoo   = "sagoo"

	LegacyTypeXunJi = "xunji"
)

var supportedTypes = []string{TypeMQTT, TypePandaX, TypeIThings, TypeSagoo}

func Normalize(raw string) string {
	nbType := strings.ToLower(strings.TrimSpace(raw))
	if nbType == LegacyTypeXunJi {
		return TypeSagoo
	}
	return nbType
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
