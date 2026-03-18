package handlers

import (
	"net/http/httptest"
	"testing"

	"github.com/gonglijing/xunjiFsu/internal/database"
	"github.com/gonglijing/xunjiFsu/internal/models"
)

func TestCalculateTotalPages(t *testing.T) {
	tests := []struct {
		name       string
		totalItems int
		pageSize   int
		want       int
	}{
		{name: "normal", totalItems: 21, pageSize: 20, want: 2},
		{name: "exact", totalItems: 40, pageSize: 20, want: 2},
		{name: "zero items", totalItems: 0, pageSize: 20, want: 0},
		{name: "zero page size", totalItems: 10, pageSize: 0, want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := calculateTotalPages(tt.totalItems, tt.pageSize); got != tt.want {
				t.Fatalf("calculateTotalPages(%d, %d) = %d, want %d", tt.totalItems, tt.pageSize, got, tt.want)
			}
		})
	}
}

func TestBuildPaginationWindow(t *testing.T) {
	tests := []struct {
		name   string
		params PaginationParams
		total  int
		start  int
		end    int
	}{
		{name: "first page", params: PaginationParams{Page: 1, PageSize: 10, Offset: 0}, total: 25, start: 0, end: 10},
		{name: "last partial", params: PaginationParams{Page: 3, PageSize: 10, Offset: 20}, total: 25, start: 20, end: 25},
		{name: "overflow", params: PaginationParams{Page: 5, PageSize: 10, Offset: 40}, total: 25, start: 25, end: 25},
		{name: "negative offset", params: PaginationParams{Page: 1, PageSize: 10, Offset: -2}, total: 25, start: 0, end: 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			window := buildPaginationWindow(tt.params, tt.total)
			if window.start != tt.start || window.end != tt.end {
				t.Fatalf("buildPaginationWindow() = {%d,%d}, want {%d,%d}", window.start, window.end, tt.start, tt.end)
			}
		})
	}
}

func TestPaginateDevices(t *testing.T) {
	devices := []*models.Device{{ID: 1}, {ID: 2}, {ID: 3}}
	params := PaginationParams{Page: 2, PageSize: 2, Offset: 2}

	items, total := paginateDevices(devices, params)
	if total != 3 {
		t.Fatalf("total = %d, want %d", total, 3)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want %d", len(items), 1)
	}

	device, ok := items[0].(*models.Device)
	if !ok {
		t.Fatalf("item type = %T, want *models.Device", items[0])
	}
	if device.ID != 3 {
		t.Fatalf("device id = %d, want %d", device.ID, 3)
	}
}

func TestParsePaginatedDeviceID(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/points?device_id=12", nil)
		deviceID, err := parsePaginatedDeviceID(req)
		if err != nil {
			t.Fatalf("parsePaginatedDeviceID returned error: %v", err)
		}
		if deviceID != 12 {
			t.Fatalf("deviceID = %d, want %d", deviceID, 12)
		}
	})

	t.Run("empty", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/points", nil)
		deviceID, err := parsePaginatedDeviceID(req)
		if err != nil {
			t.Fatalf("parsePaginatedDeviceID returned error: %v", err)
		}
		if deviceID != 0 {
			t.Fatalf("deviceID = %d, want %d", deviceID, 0)
		}
	})

	t.Run("invalid", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/points?device_id=abc", nil)
		if _, err := parsePaginatedDeviceID(req); err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestGetPagination(t *testing.T) {
	req := httptest.NewRequest("GET", "/devices?page=3&page_size=25", nil)

	params := GetPagination(req, 20)
	if params.Page != 3 {
		t.Fatalf("params.Page = %d, want %d", params.Page, 3)
	}
	if params.PageSize != 25 {
		t.Fatalf("params.PageSize = %d, want %d", params.PageSize, 25)
	}
	if params.Offset != 50 {
		t.Fatalf("params.Offset = %d, want %d", params.Offset, 50)
	}
}

func TestGetPagination_CapsPageSize(t *testing.T) {
	req := httptest.NewRequest("GET", "/devices?page_size=999", nil)

	params := GetPagination(req, 20)
	if params.PageSize != maxPaginationPageSize {
		t.Fatalf("params.PageSize = %d, want %d", params.PageSize, maxPaginationPageSize)
	}
}

func TestParsePositivePageQuery_InvalidValue(t *testing.T) {
	req := httptest.NewRequest("GET", "/devices?page=bad", nil)

	value := parsePositivePageQuery(req, "page", 7)
	if value != 7 {
		t.Fatalf("value = %d, want %d", value, 7)
	}
}

func TestNormalizePaginationPageSize(t *testing.T) {
	if got := normalizePaginationPageSize(0); got != 1 {
		t.Fatalf("normalizePaginationPageSize(0) = %d, want 1", got)
	}
	if got := normalizePaginationPageSize(25); got != 25 {
		t.Fatalf("normalizePaginationPageSize(25) = %d, want 25", got)
	}
	if got := normalizePaginationPageSize(999); got != maxPaginationPageSize {
		t.Fatalf("normalizePaginationPageSize(999) = %d, want %d", got, maxPaginationPageSize)
	}
}

func TestBuildDataPointsPage(t *testing.T) {
	points := []*database.DataPoint{{DeviceID: 1}, {DeviceID: 2}}
	params := PaginationParams{Page: 1, PageSize: 2}

	page := buildDataPointsPage(points, params)
	if page["page"].(int) != 1 {
		t.Fatalf("page = %v, want %d", page["page"], 1)
	}
	if page["page_size"].(int) != 2 {
		t.Fatalf("page_size = %v, want %d", page["page_size"], 2)
	}
	if page["has_next"].(bool) != true {
		t.Fatalf("has_next = %v, want true", page["has_next"])
	}
}
