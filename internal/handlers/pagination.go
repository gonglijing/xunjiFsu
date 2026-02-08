package handlers

import (
	"net/http"
	"strconv"

	"github.com/gonglijing/xunjiFsu/internal/database"
	"github.com/gonglijing/xunjiFsu/internal/models"
)

// PaginationParams 分页参数
type PaginationParams struct {
	Page     int // 页码（从1开始）
	PageSize int // 每页数量
	Offset   int // 计算出的偏移量
}

type paginationWindow struct {
	start int
	end   int
}

// GetPagination 从请求获取分页参数
func GetPagination(r *http.Request, defaultPageSize int) PaginationParams {
	page := 1
	pageSize := defaultPageSize

	if p := r.URL.Query().Get("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}

	if ps := r.URL.Query().Get("page_size"); ps != "" {
		if parsed, err := strconv.Atoi(ps); err == nil && parsed > 0 {
			pageSize = parsed
			// 限制最大每页数量
			if pageSize > 100 {
				pageSize = 100
			}
		}
	}

	// 计算偏移量
	offset := (page - 1) * pageSize

	return PaginationParams{
		Page:     page,
		PageSize: pageSize,
		Offset:   offset,
	}
}

// PaginatedResponse 分页响应
type PaginatedResponse struct {
	Items      interface{} `json:"items"`
	Page       int         `json:"page"`
	PageSize   int         `json:"page_size"`
	TotalItems int         `json:"total_items"`
	TotalPages int         `json:"total_pages"`
	HasNext    bool        `json:"has_next"`
	HasPrev    bool        `json:"has_prev"`
}

// NewPaginatedResponse 创建分页响应
func NewPaginatedResponse(items interface{}, params PaginationParams, totalItems int) PaginatedResponse {
	totalPages := calculateTotalPages(totalItems, params.PageSize)

	return PaginatedResponse{
		Items:      items,
		Page:       params.Page,
		PageSize:   params.PageSize,
		TotalItems: totalItems,
		TotalPages: totalPages,
		HasNext:    params.Page < totalPages,
		HasPrev:    params.Page > 1,
	}
}

func calculateTotalPages(totalItems, pageSize int) int {
	if totalItems <= 0 || pageSize <= 0 {
		return 0
	}
	return (totalItems + pageSize - 1) / pageSize
}

func buildPaginationWindow(params PaginationParams, totalItems int) paginationWindow {
	start := params.Offset
	if start < 0 {
		start = 0
	}
	if start > totalItems {
		start = totalItems
	}

	end := start + params.PageSize
	if end > totalItems {
		end = totalItems
	}

	return paginationWindow{start: start, end: end}
}

func paginateDevices(devices []*models.Device, params PaginationParams) ([]interface{}, int) {
	totalItems := len(devices)
	window := buildPaginationWindow(params, totalItems)
	if window.start >= window.end {
		return []interface{}{}, totalItems
	}

	items := make([]interface{}, 0, window.end-window.start)
	for _, device := range devices[window.start:window.end] {
		items = append(items, device)
	}

	return items, totalItems
}

func parsePaginatedDeviceID(r *http.Request) (int64, error) {
	deviceID, err := parseOptionalInt64Query(r, "device_id")
	if err != nil || deviceID == nil {
		return 0, err
	}
	return *deviceID, nil
}

func queryPaginatedDataPoints(deviceID int64, pageSize int) ([]*database.DataPoint, error) {
	if deviceID > 0 {
		return database.GetDataPointsByDevice(deviceID, pageSize)
	}
	return database.GetLatestDataPoints(pageSize)
}

func newDataPointsPage(points []*database.DataPoint, params PaginationParams) map[string]interface{} {
	return map[string]interface{}{
		"items":     points,
		"page":      params.Page,
		"page_size": params.PageSize,
		"has_next":  len(points) == params.PageSize,
	}
}

// GetPaginatedDevices 获取分页设备列表
func GetPaginatedDevices(w http.ResponseWriter, r *http.Request) {
	params := GetPagination(r, 20)

	devices, err := database.GetAllDevices()
	if err != nil {
		writeServerErrorWithLog(w, apiErrListPaginatedDevicesFailed, err)
		return
	}

	paginatedItems, totalItems := paginateDevices(devices, params)

	WriteSuccess(w, NewPaginatedResponse(paginatedItems, params, totalItems))
}

// GetPaginatedDataPoints 获取分页历史数据
func GetPaginatedDataPoints(w http.ResponseWriter, r *http.Request) {
	params := GetPagination(r, 100)

	deviceID, err := parsePaginatedDeviceID(r)
	if err != nil {
		WriteBadRequestDef(w, apiErrInvalidDeviceID)
		return
	}

	points, err := queryPaginatedDataPoints(deviceID, params.PageSize)
	if err != nil {
		writeServerErrorWithLog(w, apiErrListPaginatedDataPointsFailed, err)
		return
	}

	WriteSuccess(w, newDataPointsPage(points, params))
}
