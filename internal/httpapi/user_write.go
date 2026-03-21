package httpapi

import (
	"net/http"

	"github.com/gonglijing/xunjiFsu/internal/platform/auth"
	"github.com/gonglijing/xunjiFsu/internal/service"
)

func (api *UserAPI) CreateUser(w http.ResponseWriter, r *http.Request) {
	user, ok := parseUserRequest(w, r)
	if !ok {
		return
	}

	user, err := api.service.CreateUser(user)
	if err != nil {
		writeServerErrorWithLog(w, errCreateUserFailed, err)
		return
	}
	WriteCreated(w, service.SanitizeUser(user))
}

func (api *UserAPI) UpdateUser(w http.ResponseWriter, r *http.Request) {
	userModel, ok := api.loadUserByRequest(w, r)
	if !ok {
		return
	}

	user, ok := parseUserRequest(w, r)
	if !ok {
		return
	}
	user.ID = userModel.ID

	user, err := api.service.UpdateUser(user)
	if err != nil {
		writeServerErrorWithLog(w, errUpdateUserFailed, err)
		return
	}

	WriteSuccess(w, service.SanitizeUser(user))
}

func (api *UserAPI) DeleteUser(w http.ResponseWriter, r *http.Request) {
	user, ok := api.loadUserByRequest(w, r)
	if !ok {
		return
	}

	if err := api.service.DeleteUser(user.ID); err != nil {
		writeServerErrorWithLog(w, errDeleteUserFailed, err)
		return
	}

	WriteDeleted(w)
}

func (api *UserAPI) ChangePassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}

	if err := ParseRequest(r, &req); err != nil {
		WriteBadRequestDef(w, apiErrInvalidRequestBody)
		return
	}
	if api.authManager == nil {
		WriteErrorCode(w, http.StatusUnauthorized, "E_UNAUTHORIZED", "not authenticated")
		return
	}

	session, _ := api.authManager.GetSession(r)
	if session == nil {
		WriteErrorCode(w, http.StatusUnauthorized, "E_UNAUTHORIZED", "not authenticated")
		return
	}

	if err := auth.ChangePassword(session.UserID, req.OldPassword, req.NewPassword); err != nil {
		WriteBadRequestCode(w, errChangePassword.Code, err.Error())
		return
	}

	WriteDeleted(w)
}
