package handlers

import (
	"net/http"
	"sort"

	collectorpkg "github.com/gonglijing/xunjiFsu/internal/collector"
	"github.com/gonglijing/xunjiFsu/internal/database"
	"github.com/gonglijing/xunjiFsu/internal/models"
)

// GetDeviceRuntimeStatuses 获取所有设备采集运行时状态。
func (h *Handler) GetDeviceRuntimeStatuses(w http.ResponseWriter, r *http.Request) {
	devices, err := database.GetAllDevices()
	if err != nil {
		writeServerErrorWithLog(w, apiErrListDevicesFailed, err)
		return
	}

	runtimeStatusMap := make(map[int64]collectorpkg.DeviceRuntimeStatus)
	if h.collector != nil {
		runtimeStatusMap = h.collector.ListDeviceRuntimeStatus()
	}

	WriteSuccess(w, buildDeviceRuntimeStatusList(devices, runtimeStatusMap))
}

// GetDeviceRuntimeStatus 获取单个设备采集运行时状态。
func (h *Handler) GetDeviceRuntimeStatus(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDOrWriteBadRequestDefault(w, r)
	if !ok {
		return
	}

	if _, err := database.GetDeviceByID(id); err != nil {
		WriteNotFoundDef(w, apiErrDeviceNotFound)
		return
	}

	status := collectorpkg.DeviceRuntimeStatus{
		DeviceID:   id,
		Registered: false,
	}
	if h.collector != nil {
		if runtimeStatus, exists := h.collector.GetDeviceRuntimeStatus(id); exists {
			status = runtimeStatus
		}
	}
	if status.DeviceID == 0 {
		status.DeviceID = id
	}

	WriteSuccess(w, status)
}

func buildDeviceRuntimeStatusList(devices []*models.Device, runtimeStatusMap map[int64]collectorpkg.DeviceRuntimeStatus) []collectorpkg.DeviceRuntimeStatus {
	statuses := make([]collectorpkg.DeviceRuntimeStatus, 0, len(devices))
	for _, device := range devices {
		if device == nil {
			continue
		}

		status, ok := runtimeStatusMap[device.ID]
		if !ok {
			status = collectorpkg.DeviceRuntimeStatus{
				DeviceID:   device.ID,
				Registered: false,
			}
		}
		if status.DeviceID == 0 {
			status.DeviceID = device.ID
		}
		statuses = append(statuses, status)
	}

	sort.Slice(statuses, func(i, j int) bool {
		return statuses[i].DeviceID < statuses[j].DeviceID
	})
	return statuses
}
