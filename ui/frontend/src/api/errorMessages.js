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
};

const DEVICE_ERROR_MESSAGES = {
  E_DEVICE_NOT_FOUND: '设备不存在',
  E_DEVICE_HAS_NO_DRIVER: '设备未绑定驱动',
  E_LIST_DEVICES_FAILED: '获取设备列表失败',
  E_CREATE_DEVICE_FAILED: '创建设备失败',
  E_UPDATE_DEVICE_FAILED: '更新设备失败',
  E_DELETE_DEVICE_FAILED: '删除设备失败',
  E_TOGGLE_DEVICE_FAILED: '切换设备状态失败',
};

const DRIVER_ERROR_MESSAGES = {
  E_DRIVER_NOT_FOUND: '驱动不存在',
  E_DRIVER_NOT_LOADED: '驱动未加载',
  E_DRIVER_WASM_NOT_FOUND: '驱动文件不存在',
  E_DRIVER_CONFIG_SCHEMA_INVALID: '驱动配置模型无效',
  E_LIST_DRIVERS_FAILED: '获取驱动列表失败',
  E_CREATE_DRIVER_FAILED: '创建驱动失败',
  E_UPDATE_DRIVER_FAILED: '更新驱动失败',
  E_RELOAD_DRIVER_FAILED: '重载驱动失败',
  E_EXECUTE_DRIVER_FAILED: '执行驱动函数失败',
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
  E_GATEWAY_IDENTITY_REQUIRED: '请先配置网关 product_key 和 device_key',
  E_SYNC_GATEWAY_IDENTITY_FAILED: '同步网关身份失败',
};

const NORTHBOUND_ERROR_MESSAGES = {
  E_NORTHBOUND_NOT_FOUND: '北向配置不存在',
  E_NORTHBOUND_CONFIG_INVALID: '北向配置参数无效',
  E_NORTHBOUND_INITIALIZE_FAILED: '北向初始化失败',
  E_NORTHBOUND_RELOAD_FAILED: '北向重载失败',
};

const STORAGE_ERROR_MESSAGES = {
  E_LIST_STORAGE_CONFIGS_FAILED: '获取存储策略失败',
  E_CREATE_STORAGE_CONFIG_FAILED: '创建存储策略失败',
  E_UPDATE_STORAGE_CONFIG_FAILED: '更新存储策略失败',
  E_DELETE_STORAGE_CONFIG_FAILED: '删除存储策略失败',
  E_CLEANUP_BY_POLICY_FAILED: '按策略清理失败',
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
