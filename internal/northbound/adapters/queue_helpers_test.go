package adapters

import (
	"testing"

	"github.com/gonglijing/xunjiFsu/internal/models"
)

func TestAppendCommandQueueWithCap_DropsOldest(t *testing.T) {
	queue := []*models.NorthboundCommand{
		{RequestID: "1"},
		{RequestID: "2"},
	}
	incoming := []*models.NorthboundCommand{{RequestID: "3"}, {RequestID: "4"}}

	out := appendCommandQueueWithCap(queue, incoming, 3)

	if len(out) != 3 {
		t.Fatalf("len(out)=%d, want=3", len(out))
	}
	if out[0].RequestID != "2" || out[1].RequestID != "3" || out[2].RequestID != "4" {
		t.Fatalf("unexpected queue order: %#v", out)
	}
}

func TestAppendCommandQueueWithCap_IgnoreNilIncoming(t *testing.T) {
	queue := []*models.NorthboundCommand{{RequestID: "1"}}
	incoming := []*models.NorthboundCommand{nil, {RequestID: "2"}, nil}

	out := appendCommandQueueWithCap(queue, incoming, 3)

	if len(out) != 2 {
		t.Fatalf("len(out)=%d, want=2", len(out))
	}
	if out[0].RequestID != "1" || out[1].RequestID != "2" {
		t.Fatalf("unexpected queue order: %#v", out)
	}
}
