package httpapi

import (
	"net/http"
	"os"
)

func (api *DriverAPI) CreateDriver(w http.ResponseWriter, r *http.Request) {
	driver, ok := parseDriverRequest(w, r)
	if !ok {
		return
	}

	driverModel, err := api.service.CreateDriver(driver)
	if err != nil {
		WriteBadRequestDef(w, errDriverWasmNotFound)
		return
	}
	WriteCreated(w, driverModel)
}

func (api *DriverAPI) UpdateDriver(w http.ResponseWriter, r *http.Request) {
	driverModel, ok := api.loadDriverByRequest(w, r)
	if !ok {
		return
	}

	driver, ok := parseDriverRequest(w, r)
	if !ok {
		return
	}
	driver.ID = driverModel.ID

	driverModel, err := api.service.UpdateDriver(driver)
	if err != nil {
		WriteBadRequestDef(w, errDriverWasmNotFound)
		return
	}
	WriteSuccess(w, driverModel)
}

func (api *DriverAPI) DeleteDriver(w http.ResponseWriter, r *http.Request) {
	driverModel, ok := api.loadDriverByRequest(w, r)
	if !ok {
		return
	}
	if err := api.service.DeleteDriver(driverModel.ID); err != nil {
		WriteNotFoundDef(w, errDriverNotFound)
		return
	}
	WriteDeleted(w)
}

func (api *DriverAPI) ReloadDriver(w http.ResponseWriter, r *http.Request) {
	driverModel, ok := api.loadDriverByRequest(w, r)
	if !ok {
		return
	}

	runtime, err := api.service.ReloadDriver(driverModel.ID)
	if err != nil {
		if err == os.ErrNotExist {
			WriteBadRequestDef(w, errDriverWasmNotFound)
			return
		}
		WriteNotFoundDef(w, errDriverNotFound)
		return
	}
	WriteSuccess(w, runtime)
}
