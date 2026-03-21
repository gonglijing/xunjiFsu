package httpapi

import (
	authpkg "github.com/gonglijing/xunjiFsu/internal/auth"
	"github.com/gonglijing/xunjiFsu/internal/service"
)

type UserAPI struct {
	service     *service.UserService
	authManager *authpkg.JWTManager
}

func NewUserAPI(userService *service.UserService, authManager *authpkg.JWTManager) *UserAPI {
	return &UserAPI{service: userService, authManager: authManager}
}

var (
	errListUsersFailed  = APIErrorDef{Code: "E_LIST_USERS_FAILED", Message: "获取用户列表失败"}
	errCreateUserFailed = APIErrorDef{Code: "E_CREATE_USER_FAILED", Message: "创建用户失败"}
	errUpdateUserFailed = APIErrorDef{Code: "E_UPDATE_USER_FAILED", Message: "更新用户失败"}
	errDeleteUserFailed = APIErrorDef{Code: "E_DELETE_USER_FAILED", Message: "删除用户失败"}
	errChangePassword   = APIErrorDef{Code: "E_CHANGE_PASSWORD_FAILED", Message: "修改密码失败"}
	errUserNotFound     = APIErrorDef{Code: "E_USER_NOT_FOUND", Message: "User not found"}
)
