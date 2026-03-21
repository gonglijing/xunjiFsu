package httpapi

import (
	"net/http"

	"github.com/gonglijing/xunjiFsu/internal/service"
)

func (api *UserAPI) GetUsers(w http.ResponseWriter, r *http.Request) {
	users, err := api.service.ListUsers()
	if err != nil {
		writeServerErrorWithLog(w, errListUsersFailed, err)
		return
	}

	WriteSuccess(w, service.SanitizeUsers(users))
}
