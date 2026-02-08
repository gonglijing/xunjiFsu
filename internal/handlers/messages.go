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
)

var (
	apiErrInvalidID = APIErrorDef{
		Code:    "E_INVALID_ID",
		Message: "Invalid ID",
	}
	apiErrInvalidRequestBody = APIErrorDef{
		Code:    "E_INVALID_REQUEST_BODY",
		Message: "Invalid request body",
	}
	apiErrInvalidDeviceID = APIErrorDef{
		Code:    "E_INVALID_DEVICE_ID",
		Message: errInvalidDeviceIDMessage,
	}
	apiErrDeviceNotFound = APIErrorDef{
		Code:    "E_DEVICE_NOT_FOUND",
		Message: errDeviceNotFoundMessage,
	}
	apiErrDriverNotFound = APIErrorDef{
		Code:    "E_DRIVER_NOT_FOUND",
		Message: errDriverNotFoundMessage,
	}
	apiErrNorthboundConfigNotFound = APIErrorDef{
		Code:    "E_NORTHBOUND_NOT_FOUND",
		Message: errNorthboundConfigNotFoundMessage,
	}
	apiErrResourceNotFound = APIErrorDef{
		Code:    "E_RESOURCE_NOT_FOUND",
		Message: errResourceNotFoundMessage,
	}
	apiErrGatewayIdentityRequired = APIErrorDef{
		Code:    "E_GATEWAY_IDENTITY_REQUIRED",
		Message: errGatewayIdentityRequiredMessage,
	}
	apiErrDriverWasmFileNotFound = APIErrorDef{
		Code:    "E_DRIVER_WASM_NOT_FOUND",
		Message: errDriverWasmFileNotFoundMessage,
	}
	apiErrDriverLookupFailed = APIErrorDef{
		Code:    "E_DRIVER_LOOKUP_FAILED",
		Message: errDriverLookupFailedMessage,
	}
	apiErrDriverNotLoaded = APIErrorDef{
		Code:    "E_DRIVER_NOT_LOADED",
		Message: errDriverNotLoadedMessage,
	}
	apiErrDriverConfigSchemaInvalid = APIErrorDef{
		Code:    "E_DRIVER_CONFIG_SCHEMA_INVALID",
		Message: errDriverConfigSchemaInvalidJSONError,
	}
	apiErrDeviceHasNoDriver = APIErrorDef{
		Code:    "E_DEVICE_HAS_NO_DRIVER",
		Message: errDeviceHasNoDriverMessage,
	}
	apiErrDeviceNameRequired = APIErrorDef{
		Code:    "E_DEVICE_NAME_REQUIRED",
		Message: errDeviceNameRequiredMessage,
	}
	apiErrDriverNameRequired = APIErrorDef{
		Code:    "E_DRIVER_NAME_REQUIRED",
		Message: errDriverNameRequiredMessage,
	}
	apiErrNorthboundConfigInvalid = APIErrorDef{
		Code:    "E_NORTHBOUND_CONFIG_INVALID",
		Message: "config 参数无效",
	}
	apiErrNorthboundInitializeFailed = APIErrorDef{
		Code:    "E_NORTHBOUND_INITIALIZE_FAILED",
		Message: "北向初始化失败",
	}
	apiErrNorthboundReloadFailed = APIErrorDef{
		Code:    "E_NORTHBOUND_RELOAD_FAILED",
		Message: "北向重载失败",
	}
	apiErrListNorthboundConfigsFailed = APIErrorDef{
		Code:    "E_LIST_NORTHBOUND_CONFIGS_FAILED",
		Message: "获取北向配置失败",
	}
	apiErrCreateNorthboundConfigFailed = APIErrorDef{
		Code:    "E_CREATE_NORTHBOUND_CONFIG_FAILED",
		Message: "创建北向配置失败",
	}
	apiErrUpdateNorthboundConfigFailed = APIErrorDef{
		Code:    "E_UPDATE_NORTHBOUND_CONFIG_FAILED",
		Message: "更新北向配置失败",
	}
	apiErrDeleteNorthboundConfigFailed = APIErrorDef{
		Code:    "E_DELETE_NORTHBOUND_CONFIG_FAILED",
		Message: "删除北向配置失败",
	}
	apiErrToggleNorthboundFailed = APIErrorDef{
		Code:    "E_TOGGLE_NORTHBOUND_FAILED",
		Message: "切换北向状态失败",
	}
	apiErrListNorthboundStatusFailed = APIErrorDef{
		Code:    "E_LIST_NORTHBOUND_STATUS_FAILED",
		Message: "获取北向运行态失败",
	}
	apiErrListResourcesFailed = APIErrorDef{
		Code:    "E_LIST_RESOURCES_FAILED",
		Message: "获取资源列表失败",
	}
	apiErrCreateResourceFailed = APIErrorDef{
		Code:    "E_CREATE_RESOURCE_FAILED",
		Message: "创建资源失败",
	}
	apiErrUpdateResourceFailed = APIErrorDef{
		Code:    "E_UPDATE_RESOURCE_FAILED",
		Message: "更新资源失败",
	}
	apiErrDeleteResourceFailed = APIErrorDef{
		Code:    "E_DELETE_RESOURCE_FAILED",
		Message: "删除资源失败",
	}
	apiErrToggleResourceFailed = APIErrorDef{
		Code:    "E_TOGGLE_RESOURCE_FAILED",
		Message: "切换资源状态失败",
	}
	apiErrListUsersFailed = APIErrorDef{
		Code:    "E_LIST_USERS_FAILED",
		Message: "获取用户列表失败",
	}
	apiErrCreateUserFailed = APIErrorDef{
		Code:    "E_CREATE_USER_FAILED",
		Message: "创建用户失败",
	}
	apiErrUpdateUserFailed = APIErrorDef{
		Code:    "E_UPDATE_USER_FAILED",
		Message: "更新用户失败",
	}
	apiErrDeleteUserFailed = APIErrorDef{
		Code:    "E_DELETE_USER_FAILED",
		Message: "删除用户失败",
	}
	apiErrGetGatewayConfigFailed = APIErrorDef{
		Code:    "E_GET_GATEWAY_CONFIG_FAILED",
		Message: "获取网关配置失败",
	}
	apiErrUpdateGatewayConfigFailed = APIErrorDef{
		Code:    "E_UPDATE_GATEWAY_CONFIG_FAILED",
		Message: "更新网关配置失败",
	}
	apiErrListStorageConfigsFailed = APIErrorDef{
		Code:    "E_LIST_STORAGE_CONFIGS_FAILED",
		Message: "获取存储策略失败",
	}
	apiErrCreateStorageConfigFailed = APIErrorDef{
		Code:    "E_CREATE_STORAGE_CONFIG_FAILED",
		Message: "创建存储策略失败",
	}
	apiErrUpdateStorageConfigFailed = APIErrorDef{
		Code:    "E_UPDATE_STORAGE_CONFIG_FAILED",
		Message: "更新存储策略失败",
	}
	apiErrDeleteStorageConfigFailed = APIErrorDef{
		Code:    "E_DELETE_STORAGE_CONFIG_FAILED",
		Message: "删除存储策略失败",
	}
	apiErrCleanupDataFailed = APIErrorDef{
		Code:    "E_CLEANUP_DATA_FAILED",
		Message: "数据清理失败",
	}
	apiErrCleanupByPolicyFailed = APIErrorDef{
		Code:    "E_CLEANUP_BY_POLICY_FAILED",
		Message: "按策略清理失败",
	}
	apiErrListDevicesFailed = APIErrorDef{
		Code:    "E_LIST_DEVICES_FAILED",
		Message: "获取设备列表失败",
	}
	apiErrCreateDeviceFailed = APIErrorDef{
		Code:    "E_CREATE_DEVICE_FAILED",
		Message: "创建设备失败",
	}
	apiErrUpdateDeviceFailed = APIErrorDef{
		Code:    "E_UPDATE_DEVICE_FAILED",
		Message: "更新设备失败",
	}
	apiErrDeleteDeviceFailed = APIErrorDef{
		Code:    "E_DELETE_DEVICE_FAILED",
		Message: "删除设备失败",
	}
	apiErrToggleDeviceFailed = APIErrorDef{
		Code:    "E_TOGGLE_DEVICE_FAILED",
		Message: "切换设备状态失败",
	}
	apiErrListAlarmLogsFailed = APIErrorDef{
		Code:    "E_LIST_ALARM_LOGS_FAILED",
		Message: "获取报警日志失败",
	}
	apiErrAcknowledgeAlarmFailed = APIErrorDef{
		Code:    "E_ACKNOWLEDGE_ALARM_FAILED",
		Message: "确认报警失败",
	}
	apiErrListDataCacheFailed = APIErrorDef{
		Code:    "E_LIST_DATA_CACHE_FAILED",
		Message: "获取数据缓存失败",
	}
	apiErrGetDeviceDataCacheFailed = APIErrorDef{
		Code:    "E_GET_DEVICE_DATA_CACHE_FAILED",
		Message: "获取设备缓存失败",
	}
	apiErrQueryHistoryDataFailed = APIErrorDef{
		Code:    "E_QUERY_HISTORY_DATA_FAILED",
		Message: "查询历史数据失败",
	}
	apiErrListThresholdsFailed = APIErrorDef{
		Code:    "E_LIST_THRESHOLDS_FAILED",
		Message: "获取阈值列表失败",
	}
	apiErrCreateThresholdFailed = APIErrorDef{
		Code:    "E_CREATE_THRESHOLD_FAILED",
		Message: "创建阈值失败",
	}
	apiErrUpdateThresholdFailed = APIErrorDef{
		Code:    "E_UPDATE_THRESHOLD_FAILED",
		Message: "更新阈值失败",
	}
	apiErrDeleteThresholdFailed = APIErrorDef{
		Code:    "E_DELETE_THRESHOLD_FAILED",
		Message: "删除阈值失败",
	}
	apiErrListDriversFailed = APIErrorDef{
		Code:    "E_LIST_DRIVERS_FAILED",
		Message: "获取驱动列表失败",
	}
	apiErrCreateDriverFailed = APIErrorDef{
		Code:    "E_CREATE_DRIVER_FAILED",
		Message: "创建驱动失败",
	}
	apiErrLoadDriverFailed = APIErrorDef{
		Code:    "E_LOAD_DRIVER_FAILED",
		Message: "加载驱动失败",
	}
	apiErrUpdateDriverFailed = APIErrorDef{
		Code:    "E_UPDATE_DRIVER_FAILED",
		Message: "更新驱动失败",
	}
	apiErrReloadDriverFailed = APIErrorDef{
		Code:    "E_RELOAD_DRIVER_FAILED",
		Message: "重载驱动失败",
	}
	apiErrGetDriverRuntimeFailed = APIErrorDef{
		Code:    "E_GET_DRIVER_RUNTIME_FAILED",
		Message: "获取驱动运行态失败",
	}
	apiErrSyncGatewayIdentityFailed = APIErrorDef{
		Code:    "E_SYNC_GATEWAY_IDENTITY_FAILED",
		Message: "同步网关身份失败",
	}
	apiErrCreateDriversDirFailed = APIErrorDef{
		Code:    "E_CREATE_DRIVERS_DIR_FAILED",
		Message: "创建驱动目录失败",
	}
	apiErrSaveDriverFileFailed = APIErrorDef{
		Code:    "E_SAVE_DRIVER_FILE_FAILED",
		Message: "保存驱动文件失败",
	}
	apiErrWriteDriverFileFailed = APIErrorDef{
		Code:    "E_WRITE_DRIVER_FILE_FAILED",
		Message: "写入驱动文件失败",
	}
	apiErrListDriverFilesFailed = APIErrorDef{
		Code:    "E_LIST_DRIVER_FILES_FAILED",
		Message: "读取驱动目录失败",
	}
	apiErrExecuteDriverFailed = APIErrorDef{
		Code:    "E_EXECUTE_DRIVER_FAILED",
		Message: "执行驱动函数失败",
	}
	apiErrExecuteDriverParamInvalid = APIErrorDef{
		Code:    "E_EXECUTE_DRIVER_PARAM_INVALID",
		Message: "驱动执行参数无效",
	}
	apiErrStartCollectorFailed = APIErrorDef{
		Code:    "E_START_COLLECTOR_FAILED",
		Message: "启动采集器失败",
	}
	apiErrListPaginatedDevicesFailed = APIErrorDef{
		Code:    "E_LIST_PAGINATED_DEVICES_FAILED",
		Message: "分页查询设备失败",
	}
	apiErrListPaginatedDataPointsFailed = APIErrorDef{
		Code:    "E_LIST_PAGINATED_DATA_POINTS_FAILED",
		Message: "分页查询历史数据失败",
	}
)
