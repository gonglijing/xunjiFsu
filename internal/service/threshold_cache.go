package service

import (
	"github.com/gonglijing/xunjiFsu/internal/collector"
	"github.com/gonglijing/xunjiFsu/internal/models"
)

func InvalidateThresholdDeviceCaches(current *models.Threshold, previous *models.Threshold) {
	for _, deviceID := range BuildThresholdCacheDeviceIDs(current, previous) {
		collector.InvalidateDeviceCache(deviceID)
	}
}

func BuildThresholdCacheDeviceIDs(current *models.Threshold, previous *models.Threshold) []int64 {
	if current == nil && previous == nil {
		return nil
	}

	deviceIDs := make([]int64, 0, 2)
	appendDeviceID := func(deviceID int64) {
		if deviceID <= 0 {
			return
		}
		for _, existing := range deviceIDs {
			if existing == deviceID {
				return
			}
		}
		deviceIDs = append(deviceIDs, deviceID)
	}

	if current != nil {
		appendDeviceID(current.DeviceID)
	}
	if previous != nil {
		appendDeviceID(previous.DeviceID)
	}

	return deviceIDs
}
