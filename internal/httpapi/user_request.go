package httpapi

import (
	"net/http"

	"github.com/gonglijing/xunjiFsu/internal/models"
)

func (api *UserAPI) loadUserByRequest(w http.ResponseWriter, r *http.Request) (*models.User, bool) {
	id, ok := parseIDOrWriteBadRequest(w, r, apiErrInvalidID)
	if !ok {
		return nil, false
	}

	user, err := api.service.LoadUser(id)
	if err != nil {
		WriteNotFoundDef(w, errUserNotFound)
		return nil, false
	}
	return user, true
}

func parseUserRequest(w http.ResponseWriter, r *http.Request) (*models.User, bool) {
	var user models.User
	if err := ParseRequest(r, &user); err != nil {
		WriteBadRequestDef(w, apiErrInvalidRequestBody)
		return nil, false
	}
	return &user, true
}
