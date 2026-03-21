package httpapi

import (
	"net/http"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/service"
)

func (api *StatusAPI) GetStatus(w http.ResponseWriter, r *http.Request) {
	status, err := api.service.LoadStatus(time.Now())
	if err != nil {
		writeServerErrorWithLog(w, errGetStatusFailed, err)
		return
	}
	WriteSuccess(w, status)
}

func (api *StatusAPI) StartCollector(w http.ResponseWriter, r *http.Request) {
	if err := api.service.StartCollector(); err != nil {
		writeServerErrorWithLog(w, errStartCollectorFailed, err)
		return
	}
	WriteSuccess(w, service.BuildCollectorStatusResponse("started"))
}

func (api *StatusAPI) StopCollector(w http.ResponseWriter, r *http.Request) {
	api.service.StopCollector()
	WriteSuccess(w, service.BuildCollectorStatusResponse("stopped"))
}
