package httpapi

import "github.com/gonglijing/xunjiFsu/internal/service"

type ResourceAPI struct {
	service *service.ResourceService
}

func NewResourceAPI(resourceService *service.ResourceService) *ResourceAPI {
	return &ResourceAPI{service: resourceService}
}

var (
	apiErrInvalidID            = APIErrorDef{Code: "E_INVALID_ID", Message: "Invalid ID"}
	apiErrInvalidRequestBody   = APIErrorDef{Code: "E_INVALID_REQUEST_BODY", Message: "Invalid request body"}
	apiErrResourceNotFound     = APIErrorDef{Code: "E_RESOURCE_NOT_FOUND", Message: "resource not found"}
	apiErrListResourcesFailed  = APIErrorDef{Code: "E_LIST_RESOURCES_FAILED", Message: "获取资源列表失败"}
	apiErrCreateResourceFailed = APIErrorDef{Code: "E_CREATE_RESOURCE_FAILED", Message: "创建资源失败"}
	apiErrUpdateResourceFailed = APIErrorDef{Code: "E_UPDATE_RESOURCE_FAILED", Message: "更新资源失败"}
	apiErrDeleteResourceFailed = APIErrorDef{Code: "E_DELETE_RESOURCE_FAILED", Message: "删除资源失败"}
)
