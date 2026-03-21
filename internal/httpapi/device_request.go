package httpapi

import (
	"database/sql"
	"net/http"
	"strings"

	"github.com/gonglijing/xunjiFsu/internal/database"
	"github.com/gonglijing/xunjiFsu/internal/models"
)

func (api *DeviceAPI) loadDeviceByRequest(w http.ResponseWriter, r *http.Request) (*models.Device, bool) {
	id, ok := parseIDOrWriteBadRequest(w, r, errInvalidID)
	if !ok {
		return nil, false
	}

	device, err := api.service.LoadDevice(id)
	if err != nil {
		WriteNotFoundDef(w, errDeviceNotFound)
		return nil, false
	}
	return device, true
}

func parseAndNormalizeDevice(w http.ResponseWriter, r *http.Request) (*models.Device, bool) {
	var device models.Device
	if err := ParseRequest(r, &device); err != nil {
		WriteBadRequest(w, errInvalidRequestBodyWithDetailPrefix+err.Error())
		return nil, false
	}
	if err := normalizeDeviceInput(&device); err != nil {
		WriteBadRequestDef(w, errDeviceNameRequired)
		return nil, false
	}
	return &device, true
}

func normalizeDeviceInput(device *models.Device) error {
	if device == nil {
		return sql.ErrNoRows
	}
	device.Name = strings.TrimSpace(device.Name)
	if device.Name == "" {
		return sql.ErrNoRows
	}
	if device.StorageInterval <= 0 {
		device.StorageInterval = database.DefaultStorageIntervalSeconds
	}
	if device.Enabled != 1 {
		device.Enabled = 0
	}
	return nil
}
