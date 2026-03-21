package service

import (
	"fmt"
	"reflect"
	"strings"
	"time"
)

func parseOptionalDuration(raw string) (time.Duration, bool, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, false, nil
	}

	value, err := time.ParseDuration(raw)
	if err != nil {
		return 0, false, fmt.Errorf("invalid duration: %s", raw)
	}
	if value <= 0 {
		return 0, false, fmt.Errorf("duration must be > 0: %s", raw)
	}
	return value, true, nil
}

func applyDurationConfigChange(changes map[string]RuntimeConfigChange, key, raw string, target *time.Duration) error {
	parsed, ok, err := parseOptionalDuration(raw)
	if err != nil || !ok {
		return err
	}

	recordRuntimeConfigChange(changes, key, target.String(), parsed.String())
	*target = parsed
	return nil
}

func applyRetryConfigChange(changes map[string]RuntimeConfigChange, key string, value *int, target *int) error {
	if value == nil {
		return nil
	}
	if *value < 0 {
		return fmt.Errorf("%s must be >= 0", key)
	}

	recordRuntimeConfigChange(changes, key, *target, *value)
	*target = *value
	return nil
}

func applyPositiveIntConfigChange(changes map[string]RuntimeConfigChange, key string, value *int, target *int) error {
	if value == nil {
		return nil
	}
	if *value <= 0 {
		return fmt.Errorf("%s must be > 0", key)
	}

	recordRuntimeConfigChange(changes, key, *target, *value)
	*target = *value
	return nil
}

func recordRuntimeConfigChange(changes map[string]RuntimeConfigChange, field string, from, to any) {
	if changes == nil || strings.TrimSpace(field) == "" {
		return
	}
	if reflect.DeepEqual(from, to) {
		return
	}
	changes[field] = RuntimeConfigChange{From: from, To: to}
}
