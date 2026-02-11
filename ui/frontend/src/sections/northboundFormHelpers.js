import { NORTHBOUND_TYPE, normalizeNorthboundType } from '../utils/northboundType.js';

const MS_PER_SECOND = 1000;

function toInt(value, fallback = 0) {
  const n = Number.parseInt(`${value ?? ''}`, 10);
  return Number.isFinite(n) ? n : fallback;
}

function toBool(value) {
  if (typeof value === 'boolean') return value;
  if (typeof value === 'string') {
    const v = value.trim().toLowerCase();
    return v === 'true' || v === '1' || v === 'yes';
  }
  return !!value;
}

export function toUploadIntervalSeconds(uploadIntervalMs, fallbackSeconds = 5) {
  const milliseconds = toInt(uploadIntervalMs, fallbackSeconds * MS_PER_SECOND);
  const seconds = Math.round(milliseconds / MS_PER_SECOND);
  return seconds > 0 ? seconds : fallbackSeconds;
}

export function toUploadIntervalMs(uploadIntervalSeconds, fallbackMs = 5000) {
  const fallbackSeconds = Math.max(1, Math.round(fallbackMs / MS_PER_SECOND));
  const seconds = toInt(uploadIntervalSeconds, fallbackSeconds);
  if (seconds <= 0) return fallbackMs;
  return seconds * MS_PER_SECOND;
}

export function resolveConfigString(config, ...keys) {
  for (const key of keys) {
    const value = `${config?.[key] ?? ''}`.trim();
    if (value) return value;
  }
  return '';
}

export function safeParseJSON(value, fallback = {}) {
  try {
    return JSON.parse(value || '{}');
  } catch {
    return fallback;
  }
}

export function parseConfigFromItem(item, nbType, uploadIntervalMs, defaultUploadIntervalMs = 5000) {
  const parsed = safeParseJSON(item?.config, {});
  const merged = parsed && typeof parsed === 'object' ? { ...parsed } : {};
  const normalizedType = normalizeNorthboundType(nbType);

  const serverURL = `${item?.server_url || item?.connection?.server_url || ''}`.trim();
  if (serverURL) {
    if (!resolveConfigString(merged, 'serverUrl', 'server_url', 'broker')) {
      if (normalizedType === NORTHBOUND_TYPE.MQTT) merged.broker = serverURL;
      else merged.serverUrl = serverURL;
    }
  }

  if (!resolveConfigString(merged, 'clientId', 'client_id') && `${item?.client_id || ''}`.trim()) {
    merged.clientId = `${item.client_id}`.trim();
  }
  if (!resolveConfigString(merged, 'username') && `${item?.username || ''}`.trim()) {
    merged.username = `${item.username}`.trim();
  }
  if (!resolveConfigString(merged, 'topic') && `${item?.topic || ''}`.trim()) {
    merged.topic = `${item.topic}`.trim();
  }
  if (!resolveConfigString(merged, 'alarmTopic', 'alarm_topic') && `${item?.alarm_topic || ''}`.trim()) {
    merged.alarmTopic = `${item.alarm_topic}`.trim();
  }
  if (!resolveConfigString(merged, 'productKey', 'product_key') && `${item?.product_key || ''}`.trim()) {
    merged.productKey = `${item.product_key}`.trim();
  }
  if (!resolveConfigString(merged, 'deviceKey', 'device_key') && `${item?.device_key || ''}`.trim()) {
    merged.deviceKey = `${item.device_key}`.trim();
  }

  if (merged.qos === undefined && item?.qos !== undefined && item?.qos !== null) {
    merged.qos = toInt(item.qos, 0);
  }
  if (merged.retain === undefined && item?.retain !== undefined && item?.retain !== null) {
    merged.retain = toBool(item.retain);
  }
  if (merged.keepAlive === undefined && item?.keep_alive !== undefined && item?.keep_alive !== null) {
    merged.keepAlive = toInt(item.keep_alive, 60);
  }
  if (merged.connectTimeout === undefined && item?.timeout !== undefined && item?.timeout !== null) {
    merged.connectTimeout = toInt(item.timeout, 10);
  }

  if (toInt(merged.uploadIntervalMs, 0) <= 0) {
    merged.uploadIntervalMs = toInt(uploadIntervalMs, defaultUploadIntervalMs);
  }

  return merged;
}

export function fillPayloadFromConfig(payload, configValue) {
  const cfg = configValue && typeof configValue === 'object' ? configValue : {};
  const serverURL = resolveConfigString(cfg, 'serverUrl', 'server_url', 'broker');
  if (!`${payload.server_url || ''}`.trim() && serverURL) {
    payload.server_url = serverURL;
  }

  const username = resolveConfigString(cfg, 'username');
  if (!`${payload.username || ''}`.trim() && username) {
    payload.username = username;
  }

  const clientID = resolveConfigString(cfg, 'clientId', 'client_id');
  if (!`${payload.client_id || ''}`.trim() && clientID) {
    payload.client_id = clientID;
  }

  const topic = resolveConfigString(cfg, 'topic');
  if (!`${payload.topic || ''}`.trim() && topic) {
    payload.topic = topic;
  }

  const alarmTopic = resolveConfigString(cfg, 'alarmTopic', 'alarm_topic');
  if (!`${payload.alarm_topic || ''}`.trim() && alarmTopic) {
    payload.alarm_topic = alarmTopic;
  }

  const productKey = resolveConfigString(cfg, 'productKey', 'product_key');
  if (!`${payload.product_key || ''}`.trim() && productKey) {
    payload.product_key = productKey;
  }

  const deviceKey = resolveConfigString(cfg, 'deviceKey', 'device_key');
  if (!`${payload.device_key || ''}`.trim() && deviceKey) {
    payload.device_key = deviceKey;
  }
}

export function getNorthboundServerAddress(item) {
  const direct = `${item?.server_url || item?.connection?.server_url || ''}`.trim();
  if (direct) return direct;
  const parsed = safeParseJSON(item?.config, {});
  return resolveConfigString(parsed, 'serverUrl', 'server_url', 'broker') || '-';
}
