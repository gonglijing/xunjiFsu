package httpapi

import (
	"net/http"

	"github.com/gonglijing/xunjiFsu/internal/models"
	"github.com/gonglijing/xunjiFsu/internal/service"
)

func (api *DriverAPI) loadDriverByRequest(w http.ResponseWriter, r *http.Request) (*models.Driver, bool) {
	id, ok := parseIDOrWriteBadRequest(w, r, apiErrInvalidID)
	if !ok {
		return nil, false
	}

	driver, err := api.service.LoadDriver(id)
	if err != nil {
		WriteNotFoundDef(w, errDriverNotFound)
		return nil, false
	}
	return driver, true
}

func parseDriverRequest(w http.ResponseWriter, r *http.Request) (*models.Driver, bool) {
	var driver models.Driver
	if err := ParseRequest(r, &driver); err != nil {
		WriteBadRequestDef(w, apiErrInvalidRequestBody)
		return nil, false
	}
	if err := service.NormalizeDriverInput("", &driver); err != nil {
		WriteBadRequestDef(w, errDriverNameRequired)
		return nil, false
	}
	return &driver, true
}
