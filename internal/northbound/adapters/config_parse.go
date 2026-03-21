package adapters

import (
	"encoding/json"
	"fmt"
	"strings"
)

type adapterRawConfig struct {
	values map[string]any
}

func parseAdapterRawConfig(configStr string) (adapterRawConfig, error) {
	values := make(map[string]any)
	if err := json.Unmarshal([]byte(configStr), &values); err != nil {
		return adapterRawConfig{}, fmt.Errorf("failed to parse config: %w", err)
	}
	return adapterRawConfig{values: values}, nil
}

func (c adapterRawConfig) pickString(keys ...string) string {
	return strings.TrimSpace(pickConfigString(c.values, keys...))
}

func (c adapterRawConfig) pickInt(defaultValue int, keys ...string) int {
	return pickConfigInt(c.values, defaultValue, keys...)
}

func (c adapterRawConfig) pickBool(defaultValue bool, keys ...string) bool {
	return pickConfigBool(c.values, defaultValue, keys...)
}

func (c adapterRawConfig) pickNormalizedServerURL(keys ...string) string {
	return normalizeServerURLWithPort(
		pickConfigString(c.values, keys...),
		c.pickString("protocol"),
		c.pickInt(0, "port"),
	)
}
