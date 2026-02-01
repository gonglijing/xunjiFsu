package handlers

import (
	"net/http"
	"strconv"

	"github.com/gonglijing/xunjiFsu/internal/database"
)

// PaginationParams 分页参数
type PaginationParams struct {
	Page     int // 页码（从1开始）
	PageSize int // 每页数量
	Offset   int // 计算出的偏移量
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
	totalPages := (totalItems + params.PageSize - 1) / params.PageSize

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

// GetPaginatedDevices 获取分页设备列表
func GetPaginatedDevices(w http.ResponseWriter, r *http.Request) {
	params := GetPagination(r, 20)

	devices, err := database.GetAllDevices()
	if err != nil {
		WriteServerError(w, err.Error())
		return
	}

	// 计算分页
	totalItems := len(devices)
	totalPages := (totalItems + params.PageSize - 1) / params.PageSize

	start := params.Offset
	end := start + params.PageSize
	if end > totalItems {
		end = totalItems
	}

	var paginatedItems []interface{}
	if start < totalItems {
		for _, d := range devices[start:end] {
			paginatedItems = append(paginatedItems, d)
		}
	}

	response := PaginatedResponse{
		Items:      paginatedItems,
		Page:       params.Page,
		PageSize:   params.PageSize,
		TotalItems: totalItems,
		TotalPages: totalPages,
		HasNext:    params.Page < totalPages,
		HasPrev:    params.Page > 1,
	}

	WriteSuccess(w, response)
}

// GetPaginatedDataPoints 获取分页历史数据
func GetPaginatedDataPoints(w http.ResponseWriter, r *http.Request) {
	params := GetPagination(r, 100)

	deviceIDStr := r.URL.Query().Get("device_id")
	var deviceID int64
	if deviceIDStr != "" {
		parsed, err := strconv.ParseInt(deviceIDStr, 10, 64)
		if err != nil {
			WriteBadRequest(w, "Invalid device_id")
			return
		}
		deviceID = parsed
	}

	var points []*database.DataPoint
	var err error

	if deviceID > 0 {
		points, err = database.GetDataPointsByDevice(deviceID, params.PageSize)
	} else {
		points, err = database.GetLatestDataPoints(params.PageSize)
	}

	if err != nil {
		WriteServerError(w, err.Error())
		return
	}

	// 简单返回，不计算总数
	WriteSuccess(w, map[string]interface{}{
		"items":    points,
		"page":     params.Page,
		"page_size": params.PageSize,
		"has_next": len(points) == params.PageSize,
	})
}
