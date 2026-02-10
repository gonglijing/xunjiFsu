package handlers

import (
	"github.com/gonglijing/xunjiFsu/internal/database"
	"github.com/gonglijing/xunjiFsu/internal/models"
)

func filterSystemDataCache(cache []*models.DataCache) []*models.DataCache {
	if len(cache) == 0 {
		return cache
	}

	filtered := make([]*models.DataCache, 0, len(cache))
	for _, item := range cache {
		if item == nil || item.DeviceID == models.SystemStatsDeviceID {
			continue
		}
		filtered = append(filtered, item)
	}

	return filtered
}

func filterSystemDataPoints(points []*database.DataPoint) []*database.DataPoint {
	if len(points) == 0 {
		return points
	}

	filtered := make([]*database.DataPoint, 0, len(points))
	for _, point := range points {
		if point == nil || point.DeviceID == models.SystemStatsDeviceID {
			continue
		}
		filtered = append(filtered, point)
	}

	return filtered
}
