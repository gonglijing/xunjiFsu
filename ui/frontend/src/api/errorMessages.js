const COMMON_ERROR_MESSAGES = {
  E_BAD_REQUEST: '请求参数错误',
  E_UNAUTHORIZED: '登录已过期，请重新登录',
  E_NOT_FOUND: '请求的资源不存在',
  E_SERVER_ERROR: '服务端处理失败',
};

const REQUEST_ERROR_MESSAGES = {
  E_INVALID_ID: '无效的 ID 参数',
  E_INVALID_REQUEST_BODY: '请求体格式错误',
  E_INVALID_DEVICE_ID: '无效的设备 ID',
  E_HISTORY_DATA_QUERY_INVALID: '历史数据查询参数无效',
};

const DEVICE_ERROR_MESSAGES = {
  E_DEVICE_NOT_FOUND: '设备不存在',
  E_DEVICE_HAS_NO_DRIVER: '设备未绑定驱动',
  E_DEVICE_NAME_REQUIRED: '设备名称不能为空',
  E_LIST_DEVICES_FAILED: '获取设备列表失败',
  E_CREATE_DEVICE_FAILED: '创建设备失败',
  E_UPDATE_DEVICE_FAILED: '更新设备失败',
  E_DELETE_DEVICE_FAILED: '删除设备失败',
  E_TOGGLE_DEVICE_FAILED: '切换设备状态失败',
  E_LIST_PAGINATED_DEVICES_FAILED: '分页查询设备失败',
};

const DRIVER_ERROR_MESSAGES = {
  E_DRIVER_NOT_FOUND: '驱动不存在',
  E_DRIVER_NOT_LOADED: '驱动未加载',
  E_DRIVER_LOOKUP_FAILED: '驱动查找失败',
  E_DRIVER_NAME_REQUIRED: '驱动名称不能为空',
  E_DRIVER_WASM_NOT_FOUND: '驱动文件不存在',
  E_DRIVER_CONFIG_SCHEMA_INVALID: '驱动配置模型无效',
  E_LIST_DRIVERS_FAILED: '获取驱动列表失败',
  E_CREATE_DRIVER_FAILED: '创建驱动失败',
  E_LOAD_DRIVER_FAILED: '加载驱动失败',
  E_UPDATE_DRIVER_FAILED: '更新驱动失败',
  E_RELOAD_DRIVER_FAILED: '重载驱动失败',
  E_GET_DRIVER_RUNTIME_FAILED: '获取驱动运行态失败',
  E_CREATE_DRIVERS_DIR_FAILED: '创建驱动目录失败',
  E_SAVE_DRIVER_FILE_FAILED: '保存驱动文件失败',
  E_WRITE_DRIVER_FILE_FAILED: '写入驱动文件失败',
  E_LIST_DRIVER_FILES_FAILED: '读取驱动目录失败',
  E_EXECUTE_DRIVER_FAILED: '执行驱动函数失败',
  E_EXECUTE_DRIVER_PARAM_INVALID: '驱动执行参数无效',
};

const RESOURCE_ERROR_MESSAGES = {
  E_RESOURCE_NOT_FOUND: '资源不存在',
  E_LIST_RESOURCES_FAILED: '获取资源列表失败',
  E_CREATE_RESOURCE_FAILED: '创建资源失败',
  E_UPDATE_RESOURCE_FAILED: '更新资源失败',
  E_DELETE_RESOURCE_FAILED: '删除资源失败',
  E_TOGGLE_RESOURCE_FAILED: '切换资源状态失败',
};

const GATEWAY_ERROR_MESSAGES = {
  E_GET_GATEWAY_CONFIG_FAILED: '获取网关配置失败',
  E_UPDATE_GATEWAY_CONFIG_FAILED: '更新网关配置失败',
  E_START_COLLECTOR_FAILED: '启动采集器失败',
};

const NORTHBOUND_ERROR_MESSAGES = {
  E_NORTHBOUND_NOT_FOUND: '北向配置不存在',
  E_NORTHBOUND_CONFIG_INVALID: '北向配置参数无效',
  E_NORTHBOUND_INITIALIZE_FAILED: '北向初始化失败',
  E_NORTHBOUND_RELOAD_FAILED: '北向重载失败',
  E_LIST_NORTHBOUND_CONFIGS_FAILED: '获取北向配置失败',
  E_CREATE_NORTHBOUND_CONFIG_FAILED: '创建北向配置失败',
  E_UPDATE_NORTHBOUND_CONFIG_FAILED: '更新北向配置失败',
  E_DELETE_NORTHBOUND_CONFIG_FAILED: '删除北向配置失败',
  E_TOGGLE_NORTHBOUND_FAILED: '切换北向状态失败',
  E_LIST_NORTHBOUND_STATUS_FAILED: '获取北向运行态失败',
};

const STORAGE_ERROR_MESSAGES = {
  E_CLEANUP_BY_POLICY_FAILED: '按保留天数清理失败',
  E_CLEANUP_DATA_FAILED: '数据清理失败',
};

const USER_ERROR_MESSAGES = {
  E_LIST_USERS_FAILED: '获取用户列表失败',
  E_CREATE_USER_FAILED: '创建用户失败',
  E_UPDATE_USER_FAILED: '更新用户失败',
  E_DELETE_USER_FAILED: '删除用户失败',
};

const ALARM_ERROR_MESSAGES = {
  E_LIST_ALARM_LOGS_FAILED: '获取报警日志失败',
  E_ACKNOWLEDGE_ALARM_FAILED: '确认报警失败',
  E_DELETE_ALARM_FAILED: '删除报警失败',
  E_BATCH_DELETE_ALARM_FAILED: '批量删除报警失败',
  E_CLEAR_ALARM_LOGS_FAILED: '清空报警失败',
  E_ALARM_IDS_REQUIRED: '请选择要删除的告警',
};

const DATA_ERROR_MESSAGES = {
  E_LIST_DATA_CACHE_FAILED: '获取数据缓存失败',
  E_GET_DEVICE_DATA_CACHE_FAILED: '获取设备缓存失败',
  E_QUERY_HISTORY_DATA_FAILED: '查询历史数据失败',
  E_LIST_PAGINATED_DATA_POINTS_FAILED: '分页查询历史数据失败',
};

const THRESHOLD_ERROR_MESSAGES = {
  E_LIST_THRESHOLDS_FAILED: '获取阈值列表失败',
  E_CREATE_THRESHOLD_FAILED: '创建阈值失败',
  E_UPDATE_THRESHOLD_FAILED: '更新阈值失败',
  E_DELETE_THRESHOLD_FAILED: '删除阈值失败',
  E_GET_ALARM_REPEAT_INTERVAL_FAILED: '获取报警重复触发间隔失败',
  E_UPDATE_ALARM_REPEAT_INTERVAL_FAILED: '更新报警重复触发间隔失败',
  E_INVALID_ALARM_REPEAT_INTERVAL: '重复触发间隔必须大于 0 分钟',
};

const ERROR_CODE_MESSAGES = Object.freeze({
  ...COMMON_ERROR_MESSAGES,
  ...REQUEST_ERROR_MESSAGES,
  ...DEVICE_ERROR_MESSAGES,
  ...DRIVER_ERROR_MESSAGES,
  ...RESOURCE_ERROR_MESSAGES,
  ...GATEWAY_ERROR_MESSAGES,
  ...NORTHBOUND_ERROR_MESSAGES,
  ...STORAGE_ERROR_MESSAGES,
  ...USER_ERROR_MESSAGES,
  ...ALARM_ERROR_MESSAGES,
  ...DATA_ERROR_MESSAGES,
  ...THRESHOLD_ERROR_MESSAGES,
});

export function getErrorMessage(error, fallback = '操作失败') {
  if (!error) return fallback;
  if (typeof error === 'string') return error;

  if (typeof error.userMessage === 'string' && error.userMessage.trim()) {
    return error.userMessage.trim();
  }
  if (typeof error.message === 'string' && error.message.trim()) {
    return error.message.trim();
  }

  const code = typeof error.code === 'string' ? error.code.trim() : '';
  return ERROR_CODE_MESSAGES[code] || fallback;
}

export function resolveAPIErrorMessage(code, message, fallback = '操作失败') {
  const normalizedMessage = typeof message === 'string' ? message.trim() : '';
  if (normalizedMessage) return normalizedMessage;

  const normalizedCode = typeof code === 'string' ? code.trim() : '';
  if (normalizedCode && ERROR_CODE_MESSAGES[normalizedCode]) {
    return ERROR_CODE_MESSAGES[normalizedCode];
  }

  return fallback;
}
