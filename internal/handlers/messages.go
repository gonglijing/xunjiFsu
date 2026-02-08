package handlers

const (
	errInvalidRequestBodyWithDetailPrefix = "Invalid request body: "
	errInvalidDeviceIDMessage             = "Invalid device_id"

	errDeviceNameRequiredMessage = "device name is required"
	errDriverNameRequiredMessage = "driver name is required"

	errDeviceNotFoundMessage           = "Device not found"
	errDriverNotFoundMessage           = "Driver not found"
	errNorthboundConfigNotFoundMessage = "Northbound config not found"
	errResourceNotFoundMessage         = "resource not found"

	errDriverWasmFileNotFoundMessage      = "driver wasm file not found"
	errOnlyWasmFilesAllowedMessage        = "Only .wasm files are allowed"
	errDriverLookupFailedMessage          = "driver not found"
	errDriverNotLoadedMessage             = "driver is not loaded"
	errDriverConfigSchemaInvalidJSONError = "driver config_schema is invalid JSON"
	errDeviceHasNoDriverMessage           = "Device has no driver"

	errGatewayIdentityRequiredMessage = "请先在网关配置中设置 product_key 和 device_key"
	errHistoryStartAfterEndDetail     = "start time must be before end time"
	errHistoryFilterRequiresDevice    = "device_id is required when using field_name/start/end filters"
)

func newAPIError(code, message string) APIErrorDef {
	return APIErrorDef{Code: code, Message: message}
}

var (
	apiErrInvalidID                     = newAPIError("E_INVALID_ID", "Invalid ID")
	apiErrInvalidRequestBody            = newAPIError("E_INVALID_REQUEST_BODY", "Invalid request body")
	apiErrInvalidDeviceID               = newAPIError("E_INVALID_DEVICE_ID", errInvalidDeviceIDMessage)
	apiErrDeviceNotFound                = newAPIError("E_DEVICE_NOT_FOUND", errDeviceNotFoundMessage)
	apiErrDriverNotFound                = newAPIError("E_DRIVER_NOT_FOUND", errDriverNotFoundMessage)
	apiErrNorthboundConfigNotFound      = newAPIError("E_NORTHBOUND_NOT_FOUND", errNorthboundConfigNotFoundMessage)
	apiErrResourceNotFound              = newAPIError("E_RESOURCE_NOT_FOUND", errResourceNotFoundMessage)
	apiErrGatewayIdentityRequired       = newAPIError("E_GATEWAY_IDENTITY_REQUIRED", errGatewayIdentityRequiredMessage)
	apiErrDriverWasmFileNotFound        = newAPIError("E_DRIVER_WASM_NOT_FOUND", errDriverWasmFileNotFoundMessage)
	apiErrDriverLookupFailed            = newAPIError("E_DRIVER_LOOKUP_FAILED", errDriverLookupFailedMessage)
	apiErrDriverNotLoaded               = newAPIError("E_DRIVER_NOT_LOADED", errDriverNotLoadedMessage)
	apiErrDriverConfigSchemaInvalid     = newAPIError("E_DRIVER_CONFIG_SCHEMA_INVALID", errDriverConfigSchemaInvalidJSONError)
	apiErrDeviceHasNoDriver             = newAPIError("E_DEVICE_HAS_NO_DRIVER", errDeviceHasNoDriverMessage)
	apiErrDeviceNameRequired            = newAPIError("E_DEVICE_NAME_REQUIRED", errDeviceNameRequiredMessage)
	apiErrDriverNameRequired            = newAPIError("E_DRIVER_NAME_REQUIRED", errDriverNameRequiredMessage)
	apiErrNorthboundConfigInvalid       = newAPIError("E_NORTHBOUND_CONFIG_INVALID", "config 参数无效")
	apiErrNorthboundInitializeFailed    = newAPIError("E_NORTHBOUND_INITIALIZE_FAILED", "北向初始化失败")
	apiErrNorthboundReloadFailed        = newAPIError("E_NORTHBOUND_RELOAD_FAILED", "北向重载失败")
	apiErrListNorthboundConfigsFailed   = newAPIError("E_LIST_NORTHBOUND_CONFIGS_FAILED", "获取北向配置失败")
	apiErrCreateNorthboundConfigFailed  = newAPIError("E_CREATE_NORTHBOUND_CONFIG_FAILED", "创建北向配置失败")
	apiErrUpdateNorthboundConfigFailed  = newAPIError("E_UPDATE_NORTHBOUND_CONFIG_FAILED", "更新北向配置失败")
	apiErrDeleteNorthboundConfigFailed  = newAPIError("E_DELETE_NORTHBOUND_CONFIG_FAILED", "删除北向配置失败")
	apiErrToggleNorthboundFailed        = newAPIError("E_TOGGLE_NORTHBOUND_FAILED", "切换北向状态失败")
	apiErrListNorthboundStatusFailed    = newAPIError("E_LIST_NORTHBOUND_STATUS_FAILED", "获取北向运行态失败")
	apiErrListResourcesFailed           = newAPIError("E_LIST_RESOURCES_FAILED", "获取资源列表失败")
	apiErrCreateResourceFailed          = newAPIError("E_CREATE_RESOURCE_FAILED", "创建资源失败")
	apiErrUpdateResourceFailed          = newAPIError("E_UPDATE_RESOURCE_FAILED", "更新资源失败")
	apiErrDeleteResourceFailed          = newAPIError("E_DELETE_RESOURCE_FAILED", "删除资源失败")
	apiErrToggleResourceFailed          = newAPIError("E_TOGGLE_RESOURCE_FAILED", "切换资源状态失败")
	apiErrListUsersFailed               = newAPIError("E_LIST_USERS_FAILED", "获取用户列表失败")
	apiErrCreateUserFailed              = newAPIError("E_CREATE_USER_FAILED", "创建用户失败")
	apiErrUpdateUserFailed              = newAPIError("E_UPDATE_USER_FAILED", "更新用户失败")
	apiErrDeleteUserFailed              = newAPIError("E_DELETE_USER_FAILED", "删除用户失败")
	apiErrGetGatewayConfigFailed        = newAPIError("E_GET_GATEWAY_CONFIG_FAILED", "获取网关配置失败")
	apiErrUpdateGatewayConfigFailed     = newAPIError("E_UPDATE_GATEWAY_CONFIG_FAILED", "更新网关配置失败")
	apiErrUpdateRuntimeConfigFailed     = newAPIError("E_UPDATE_RUNTIME_CONFIG_FAILED", "更新运行时参数失败")
	apiErrListRuntimeConfigAuditsFailed = newAPIError("E_LIST_RUNTIME_CONFIG_AUDITS_FAILED", "查询运行时参数审计日志失败")
	apiErrListStorageConfigsFailed      = newAPIError("E_LIST_STORAGE_CONFIGS_FAILED", "获取存储策略失败")
	apiErrCreateStorageConfigFailed     = newAPIError("E_CREATE_STORAGE_CONFIG_FAILED", "创建存储策略失败")
	apiErrUpdateStorageConfigFailed     = newAPIError("E_UPDATE_STORAGE_CONFIG_FAILED", "更新存储策略失败")
	apiErrDeleteStorageConfigFailed     = newAPIError("E_DELETE_STORAGE_CONFIG_FAILED", "删除存储策略失败")
	apiErrCleanupDataFailed             = newAPIError("E_CLEANUP_DATA_FAILED", "数据清理失败")
	apiErrCleanupByPolicyFailed         = newAPIError("E_CLEANUP_BY_POLICY_FAILED", "按策略清理失败")
	apiErrListDevicesFailed             = newAPIError("E_LIST_DEVICES_FAILED", "获取设备列表失败")
	apiErrCreateDeviceFailed            = newAPIError("E_CREATE_DEVICE_FAILED", "创建设备失败")
	apiErrUpdateDeviceFailed            = newAPIError("E_UPDATE_DEVICE_FAILED", "更新设备失败")
	apiErrDeleteDeviceFailed            = newAPIError("E_DELETE_DEVICE_FAILED", "删除设备失败")
	apiErrToggleDeviceFailed            = newAPIError("E_TOGGLE_DEVICE_FAILED", "切换设备状态失败")
	apiErrListAlarmLogsFailed           = newAPIError("E_LIST_ALARM_LOGS_FAILED", "获取报警日志失败")
	apiErrAcknowledgeAlarmFailed        = newAPIError("E_ACKNOWLEDGE_ALARM_FAILED", "确认报警失败")
	apiErrListDataCacheFailed           = newAPIError("E_LIST_DATA_CACHE_FAILED", "获取数据缓存失败")
	apiErrGetDeviceDataCacheFailed      = newAPIError("E_GET_DEVICE_DATA_CACHE_FAILED", "获取设备缓存失败")
	apiErrQueryHistoryDataFailed        = newAPIError("E_QUERY_HISTORY_DATA_FAILED", "查询历史数据失败")
	apiErrListThresholdsFailed          = newAPIError("E_LIST_THRESHOLDS_FAILED", "获取阈值列表失败")
	apiErrCreateThresholdFailed         = newAPIError("E_CREATE_THRESHOLD_FAILED", "创建阈值失败")
	apiErrUpdateThresholdFailed         = newAPIError("E_UPDATE_THRESHOLD_FAILED", "更新阈值失败")
	apiErrDeleteThresholdFailed         = newAPIError("E_DELETE_THRESHOLD_FAILED", "删除阈值失败")
	apiErrListDriversFailed             = newAPIError("E_LIST_DRIVERS_FAILED", "获取驱动列表失败")
	apiErrCreateDriverFailed            = newAPIError("E_CREATE_DRIVER_FAILED", "创建驱动失败")
	apiErrLoadDriverFailed              = newAPIError("E_LOAD_DRIVER_FAILED", "加载驱动失败")
	apiErrUpdateDriverFailed            = newAPIError("E_UPDATE_DRIVER_FAILED", "更新驱动失败")
	apiErrReloadDriverFailed            = newAPIError("E_RELOAD_DRIVER_FAILED", "重载驱动失败")
	apiErrGetDriverRuntimeFailed        = newAPIError("E_GET_DRIVER_RUNTIME_FAILED", "获取驱动运行态失败")
	apiErrSyncGatewayIdentityFailed     = newAPIError("E_SYNC_GATEWAY_IDENTITY_FAILED", "同步网关身份失败")
	apiErrCreateDriversDirFailed        = newAPIError("E_CREATE_DRIVERS_DIR_FAILED", "创建驱动目录失败")
	apiErrSaveDriverFileFailed          = newAPIError("E_SAVE_DRIVER_FILE_FAILED", "保存驱动文件失败")
	apiErrWriteDriverFileFailed         = newAPIError("E_WRITE_DRIVER_FILE_FAILED", "写入驱动文件失败")
	apiErrListDriverFilesFailed         = newAPIError("E_LIST_DRIVER_FILES_FAILED", "读取驱动目录失败")
	apiErrExecuteDriverFailed           = newAPIError("E_EXECUTE_DRIVER_FAILED", "执行驱动函数失败")
	apiErrExecuteDriverParamInvalid     = newAPIError("E_EXECUTE_DRIVER_PARAM_INVALID", "驱动执行参数无效")
	apiErrStartCollectorFailed          = newAPIError("E_START_COLLECTOR_FAILED", "启动采集器失败")
	apiErrListPaginatedDevicesFailed    = newAPIError("E_LIST_PAGINATED_DEVICES_FAILED", "分页查询设备失败")
	apiErrListPaginatedDataPointsFailed = newAPIError("E_LIST_PAGINATED_DATA_POINTS_FAILED", "分页查询历史数据失败")
	apiErrHistoryDataQueryInvalid       = newAPIError("E_HISTORY_DATA_QUERY_INVALID", "历史数据查询参数无效")
)
