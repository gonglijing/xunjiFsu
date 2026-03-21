package httpapi

import "github.com/gonglijing/xunjiFsu/internal/service"

type GatewayAPI struct {
	configService  *service.GatewayConfigService
	runtimeService *service.GatewayRuntimeService
}

func NewGatewayAPI(configService *service.GatewayConfigService, runtimeService *service.GatewayRuntimeService) *GatewayAPI {
	return &GatewayAPI{
		configService:  configService,
		runtimeService: runtimeService,
	}
}

type runtimeConfigAuditView struct {
	ID               int64                                  `json:"id"`
	OperatorUserID   int64                                  `json:"operator_user_id"`
	OperatorUsername string                                 `json:"operator_username"`
	SourceIP         string                                 `json:"source_ip"`
	CreatedAt        string                                 `json:"created_at"`
	Changes          map[string]service.RuntimeConfigChange `json:"changes,omitempty"`
	ChangesRaw       string                                 `json:"changes_raw,omitempty"`
}

var (
	errGetGatewayConfigFailed        = APIErrorDef{Code: "E_GET_GATEWAY_CONFIG_FAILED", Message: "获取网关配置失败"}
	errUpdateGatewayConfigFailed     = APIErrorDef{Code: "E_UPDATE_GATEWAY_CONFIG_FAILED", Message: "更新网关配置失败"}
	errUpdateRuntimeConfigFailed     = APIErrorDef{Code: "E_UPDATE_RUNTIME_CONFIG_FAILED", Message: "更新运行时参数失败"}
	errListRuntimeConfigAuditsFailed = APIErrorDef{Code: "E_LIST_RUNTIME_CONFIG_AUDITS_FAILED", Message: "查询运行时参数审计日志失败"}
)
