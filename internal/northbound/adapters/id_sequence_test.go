package adapters

import (
	"strconv"
	"strings"
	"testing"
)

func TestNextPrefixedID_SequenceAndFormat(t *testing.T) {
	var seq uint64
	id1 := nextPrefixedID("req", &seq)
	id2 := nextPrefixedID("req", &seq)

	parts1 := strings.Split(id1, "_")
	if len(parts1) != 3 {
		t.Fatalf("id1 format invalid: %q", id1)
	}
	if parts1[0] != "req" {
		t.Fatalf("id1 prefix=%q, want=req", parts1[0])
	}
	if _, err := strconv.ParseInt(parts1[1], 10, 64); err != nil {
		t.Fatalf("id1 millis invalid: %q", parts1[1])
	}
	if parts1[2] != "1" {
		t.Fatalf("id1 seq=%q, want=1", parts1[2])
	}

	parts2 := strings.Split(id2, "_")
	if len(parts2) != 3 {
		t.Fatalf("id2 format invalid: %q", id2)
	}
	if parts2[2] != "2" {
		t.Fatalf("id2 seq=%q, want=2", parts2[2])
	}
}
