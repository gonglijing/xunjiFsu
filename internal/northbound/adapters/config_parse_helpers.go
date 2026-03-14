package adapters

import (
	"encoding/json"
	"fmt"
	"strings"
)

type adapterRawConfig struct {
	values map[string]interface{}
}

func parseAdapterRawConfig(configStr string) (adapterRawConfig, error) {
	values := make(map[string]interface{})
	if err := json.Unmarshal([]byte(configStr), &values); err != nil {
		return adapterRawConfig{}, fmt.Errorf("failed to parse config: %w", err)
	}
	return adapterRawConfig{values: values}, nil
}

func (c adapterRawConfig) string(keys ...string) string {
	return strings.TrimSpace(pickConfigString(c.values, keys...))
}

func (c adapterRawConfig) int(defaultValue int, keys ...string) int {
	return pickConfigInt(c.values, defaultValue, keys...)
}

func (c adapterRawConfig) bool(defaultValue bool, keys ...string) bool {
	return pickConfigBool(c.values, defaultValue, keys...)
}

func (c adapterRawConfig) normalizedServerURL(keys ...string) string {
	return normalizeServerURLWithPort(
		pickConfigString(c.values, keys...),
		c.string("protocol"),
		c.int(0, "port"),
	)
}
