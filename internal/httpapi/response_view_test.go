package httpapi

import (
	"encoding/json"
	"testing"
)

func TestDeletedCountView_JSONShape(t *testing.T) {
	data, err := json.Marshal(deletedCountView{Deleted: 5})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if string(data) != `{"deleted":5}` {
		t.Fatalf("json = %s", data)
	}
}

func TestOperationStatusView_JSONShape(t *testing.T) {
	data, err := json.Marshal(operationStatusView{Status: "acknowledged"})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if string(data) != `{"status":"acknowledged"}` {
		t.Fatalf("json = %s", data)
	}
}

func TestEnabledStateView_JSONShape(t *testing.T) {
	data, err := json.Marshal(enabledStateView{Enabled: 1})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if string(data) != `{"enabled":1}` {
		t.Fatalf("json = %s", data)
	}
}
