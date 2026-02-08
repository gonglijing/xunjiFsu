const DEV_DEFAULT_AUTH_CHECK_INTERVAL_MS = 600;
const PROD_DEFAULT_AUTH_CHECK_INTERVAL_MS = 1200;
const MIN_AUTH_CHECK_INTERVAL_MS = 200;
const MAX_AUTH_CHECK_INTERVAL_MS = 30000;

const DEV_DEFAULT_DASHBOARD_STATUS_POLL_MS = 3000;
const PROD_DEFAULT_DASHBOARD_STATUS_POLL_MS = 5000;

const DEV_DEFAULT_GATEWAY_METRICS_POLL_MS = 5000;
const PROD_DEFAULT_GATEWAY_METRICS_POLL_MS = 8000;

const DEV_DEFAULT_REALTIME_MINI_POLL_MS = 3000;
const PROD_DEFAULT_REALTIME_MINI_POLL_MS = 4000;

const DEV_DEFAULT_NORTHBOUND_UPLOAD_INTERVAL_MS = 5000;
const PROD_DEFAULT_NORTHBOUND_UPLOAD_INTERVAL_MS = 5000;

const MIN_POLL_INTERVAL_MS = 1000;
const MAX_POLL_INTERVAL_MS = 60000;
const MIN_UPLOAD_INTERVAL_MS = 100;
const MAX_UPLOAD_INTERVAL_MS = 3600000;
let runtimeConfigLogged = false;
const warnedEnvKeys = new Set();
let cachedRuntimeConfig = null;

function clamp(value, min, max) {
  if (value < min) return min;
  if (value > max) return max;
  return value;
}

function parsePositiveInt(raw) {
  const parsed = Number.parseInt(`${raw ?? ''}`, 10);
  if (!Number.isFinite(parsed)) return { ok: false, reason: '不是有效整数' };
  if (parsed <= 0) return { ok: false, reason: '必须大于 0' };
  return { ok: true, value: parsed };
}

function defaultAuthCheckInterval() {
  return import.meta.env.DEV
    ? DEV_DEFAULT_AUTH_CHECK_INTERVAL_MS
    : PROD_DEFAULT_AUTH_CHECK_INTERVAL_MS;
}

function warnInvalidEnvValue(envKey, rawValue, reason, fallback) {
  if (!import.meta.env.DEV) return;
  if (warnedEnvKeys.has(envKey)) return;
  warnedEnvKeys.add(envKey);

  console.warn(
    `[gogw-ui] ${envKey}=${JSON.stringify(rawValue)} 无效（${reason}），已回退默认值 ${fallback}`,
  );
}

function resolveIntervalFromEnv(envKey, fallback, min = MIN_POLL_INTERVAL_MS, max = MAX_POLL_INTERVAL_MS) {
  const rawValue = import.meta.env[envKey];
  const rawText = `${rawValue ?? ''}`.trim();
  if (!rawText) {
    return fallback;
  }

  const parsed = parsePositiveInt(rawText);
  if (!parsed.ok) {
    warnInvalidEnvValue(envKey, rawValue, parsed.reason, fallback);
    return fallback;
  }

  if (parsed.value < min || parsed.value > max) {
    warnInvalidEnvValue(envKey, rawValue, `超出范围 [${min}, ${max}]`, fallback);
    return fallback;
  }

  return parsed.value;
}

function buildRuntimeConfig() {
  const authFallback = defaultAuthCheckInterval();
  const dashboardFallback = import.meta.env.DEV
    ? DEV_DEFAULT_DASHBOARD_STATUS_POLL_MS
    : PROD_DEFAULT_DASHBOARD_STATUS_POLL_MS;
  const gatewayFallback = import.meta.env.DEV
    ? DEV_DEFAULT_GATEWAY_METRICS_POLL_MS
    : PROD_DEFAULT_GATEWAY_METRICS_POLL_MS;
  const realtimeFallback = import.meta.env.DEV
    ? DEV_DEFAULT_REALTIME_MINI_POLL_MS
    : PROD_DEFAULT_REALTIME_MINI_POLL_MS;
  const northboundUploadFallback = import.meta.env.DEV
    ? DEV_DEFAULT_NORTHBOUND_UPLOAD_INTERVAL_MS
    : PROD_DEFAULT_NORTHBOUND_UPLOAD_INTERVAL_MS;

  return {
    mode: import.meta.env.MODE,
    auth_check_interval_ms: resolveIntervalFromEnv(
      'VITE_AUTH_CHECK_INTERVAL_MS',
      authFallback,
      MIN_AUTH_CHECK_INTERVAL_MS,
      MAX_AUTH_CHECK_INTERVAL_MS,
    ),
    dashboard_status_poll_ms: resolveIntervalFromEnv(
      'VITE_DASHBOARD_STATUS_POLL_MS',
      dashboardFallback,
    ),
    gateway_metrics_poll_ms: resolveIntervalFromEnv(
      'VITE_GATEWAY_METRICS_POLL_MS',
      gatewayFallback,
    ),
    realtime_mini_poll_ms: resolveIntervalFromEnv(
      'VITE_REALTIME_MINI_POLL_MS',
      realtimeFallback,
    ),
    northbound_default_upload_interval_ms: resolveIntervalFromEnv(
      'VITE_NORTHBOUND_DEFAULT_UPLOAD_INTERVAL_MS',
      northboundUploadFallback,
      MIN_UPLOAD_INTERVAL_MS,
      MAX_UPLOAD_INTERVAL_MS,
    ),
  };
}

export function validateAndWarnConfig() {
  if (!cachedRuntimeConfig) {
    cachedRuntimeConfig = buildRuntimeConfig();
  }
  return cachedRuntimeConfig;
}

export function getAuthCheckIntervalMs() {
  return validateAndWarnConfig().auth_check_interval_ms;
}

export function getDashboardStatusPollIntervalMs() {
  return validateAndWarnConfig().dashboard_status_poll_ms;
}

export function getGatewayMetricsPollIntervalMs() {
  return validateAndWarnConfig().gateway_metrics_poll_ms;
}

export function getRealtimeMiniPollIntervalMs() {
  return validateAndWarnConfig().realtime_mini_poll_ms;
}

export function getNorthboundDefaultUploadIntervalMs() {
  return validateAndWarnConfig().northbound_default_upload_interval_ms;
}

export function getFrontendRuntimeConfig() {
  return validateAndWarnConfig();
}

export function logFrontendRuntimeConfig() {
  if (!import.meta.env.DEV || runtimeConfigLogged) return;
  runtimeConfigLogged = true;
  console.info('[gogw-ui] runtime config', getFrontendRuntimeConfig());
}
