package handlers

import "testing"

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
