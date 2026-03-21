package httpapi

import (
	"net/http"
	"os"

	"github.com/gonglijing/xunjiFsu/internal/service"
)

func (api *DriverAPI) GetDrivers(w http.ResponseWriter, r *http.Request) {
	drivers, err := api.service.ListDrivers()
	if err != nil {
		writeServerErrorWithLog(w, errListDriversFailed, err)
		return
	}
	WriteSuccess(w, drivers)
}

func (api *DriverAPI) GetDriverRuntimeList(w http.ResponseWriter, r *http.Request) {
	WriteSuccess(w, api.service.ListDriverRuntimes())
}

func (api *DriverAPI) GetDriverRuntime(w http.ResponseWriter, r *http.Request) {
	driverModel, ok := api.loadDriverByRequest(w, r)
	if !ok {
		return
	}

	runtime, err := api.service.LoadDriverRuntime(driverModel.ID)
	if err != nil {
		WriteNotFoundDef(w, errDriverNotFound)
		return
	}
	WriteSuccess(w, runtime)
}

func (api *DriverAPI) ListDriverFiles(w http.ResponseWriter, r *http.Request) {
	files, err := api.service.ListDriverFiles()
	if err != nil {
		if os.IsNotExist(err) {
			WriteSuccess(w, []service.DriverFileItem{})
			return
		}
		writeServerErrorWithLog(w, errListDriverFilesFailed, err)
		return
	}
	WriteSuccess(w, files)
}
