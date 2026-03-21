package adapters

import "fmt"

func validateConfigQOS(qos int) error {
	if qos < 0 || qos > 2 {
		return fmt.Errorf("qos must be between 0 and 2")
	}
	return nil
}

func applyDefaultPositiveInt(value *int, defaultValue int) {
	if *value <= 0 {
		*value = defaultValue
	}
}

func applyMinimumPositiveInt(value *int, minimum int) {
	if *value > 0 && *value < minimum {
		*value = minimum
	}
}

func applyFallbackPositiveInt(value *int, fallback int) {
	if *value <= 0 {
		*value = fallback
	}
}

func applyFallbackOrDefaultPositiveInt(value *int, fallback, defaultValue int) {
	if *value > 0 {
		return
	}
	if fallback > 0 {
		*value = fallback
		return
	}
	*value = defaultValue
}

func applyDefaultString(value *string, defaultValue string) {
	if *value == "" {
		*value = defaultValue
	}
}
