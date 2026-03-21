package httpapi

import (
	"encoding/json"
	"testing"
)

func TestNorthboundEnabledView_JSONShape(t *testing.T) {
	data, err := json.Marshal(northboundEnabledView{Enabled: 1})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if string(data) != `{"enabled":1}` {
		t.Fatalf("json = %s", data)
	}
}

func TestNorthboundSyncView_JSONShape(t *testing.T) {
	data, err := json.Marshal(northboundSyncView{
		ID:      7,
		Name:    "demo",
		Type:    "mqtt",
		Message: "同步设备已触发",
	})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if string(data) != `{"id":7,"name":"demo","type":"mqtt","message":"同步设备已触发"}` {
		t.Fatalf("json = %s", data)
	}
}
