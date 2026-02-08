const TOKEN_KEY = 'gogw_jwt';
import { resolveAPIErrorMessage } from './api/errorMessages';

const unauthorizedListeners = new Set();

function getToken() {
  return localStorage.getItem(TOKEN_KEY) || '';
}

function emitUnauthorized() {
  let handled = false;
  unauthorizedListeners.forEach((listener) => {
    try {
      if (listener() === true) handled = true;
    } catch {
      // ignore listener errors
    }
  });
  return handled;
}

function authHeaders(headers = {}) {
  const token = getToken();
  return token ? { ...headers, Authorization: `Bearer ${token}` } : headers;
}

function handleAuth(res) {
  if (res.status === 401) {
    localStorage.removeItem(TOKEN_KEY);
    const handled = emitUnauthorized();
    if (!handled) {
      window.location.href = '/login';
    }
    const err = new Error('unauthorized');
    err.status = 401;
    err.code = 'E_UNAUTHORIZED';
    throw err;
  }
}

async function authFetch(url, options = {}) {
  const res = await fetch(url, {
    credentials: 'same-origin',
    headers: authHeaders(options.headers),
    ...options,
  });
  handleAuth(res);
  return res;
}

async function parseJSON(res) {
  if (res.status === 204) return null;
  const text = await res.text();
  if (!text) return null;
  try {
    const json = JSON.parse(text);
    // 统一处理 APIResponse 包装
    if (json && typeof json === 'object' && 'success' in json) {
      if (json.success) return json.data ?? null;
      const err = new Error(resolveAPIErrorMessage(json.code, json.error || json.message, 'request failed'));
      err.code = json.code || '';
      err.status = res.status;
      throw err;
    }
    return json;
  } catch (e) {
    if (e instanceof Error && e.message !== 'invalid json') {
      throw e;
    }
    throw new Error('invalid json');
  }
}

function extractAPIError(payload, fallbackMessage) {
  if (!payload || typeof payload !== 'object') {
    return { message: fallbackMessage, code: '' };
  }

  return {
    message: resolveAPIErrorMessage(payload.code, payload.error || payload.message, fallbackMessage),
    code: payload.code || '',
  };
}

async function parseError(res) {
  const fallbackMessage = `${res.status} ${res.statusText}`;
  if (res.status === 204) {
    const err = new Error(fallbackMessage);
    err.status = res.status;
    err.code = '';
    return err;
  }

  const text = await res.text();
  if (!text) {
    const err = new Error(fallbackMessage);
    err.status = res.status;
    err.code = '';
    return err;
  }

  try {
    const json = JSON.parse(text);
    const parsed = extractAPIError(json, fallbackMessage);
    const err = new Error(parsed.message);
    err.status = res.status;
    err.code = parsed.code;
    return err;
  } catch (_) {
    const err = new Error(fallbackMessage);
    err.status = res.status;
    err.code = '';
    return err;
  }
}

async function ensureOK(res) {
  if (res.ok || res.status === 204) return;
  throw await parseError(res);
}

export function storeToken(token) {
  if (token) localStorage.setItem(TOKEN_KEY, token);
}

export function clearToken() {
  localStorage.removeItem(TOKEN_KEY);
}

export function onUnauthorized(listener) {
  if (typeof listener !== 'function') {
    return () => {};
  }
  unauthorizedListeners.add(listener);
  return () => unauthorizedListeners.delete(listener);
}

export async function getJSON(url, options = {}) {
  const res = await authFetch(url, options);
  await ensureOK(res);
  return parseJSON(res);
}

export async function postJSON(url, body, options = {}) {
  const res = await authFetch(url, {
    method: 'POST',
    headers: authHeaders({ 'Content-Type': 'application/json', ...(options.headers || {}) }),
    body: JSON.stringify(body),
    ...options,
  });
  await ensureOK(res);
  return parseJSON(res);
}

export async function post(url, options = {}) {
  const res = await authFetch(url, {
    method: 'POST',
    ...options,
  });
  await ensureOK(res);
  return parseJSON(res);
}

export async function putJSON(url, body, options = {}) {
  const res = await authFetch(url, {
    method: 'PUT',
    headers: authHeaders({ 'Content-Type': 'application/json', ...(options.headers || {}) }),
    body: JSON.stringify(body),
    ...options,
  });
  await ensureOK(res);
  return parseJSON(res);
}

export async function del(url) {
  const res = await authFetch(url, { method: 'DELETE' });
  if (!res.ok && res.status !== 204) throw await parseError(res);
  if (res.status === 204) return null;
  return parseJSON(res);
}

// 用于上传等非 JSON 请求，仍然自动附带 JWT
export async function upload(url, formData, options = {}) {
  const res = await authFetch(url, {
    method: 'POST',
    body: formData,
    ...options,
  });
  await ensureOK(res);
  return parseJSON(res);
}

export function unwrapData(value, fallback = null) {
  if (value === null || value === undefined) return fallback;
  if (value && typeof value === 'object' && 'data' in value) {
    return value.data ?? fallback;
  }
  return value;
}

// SolidJS 资源获取辅助函数
import { createResource, createSignal } from 'solid-js';

export function useFetch(url, options = {}) {
  const [data, { mutate, refetch }] = createResource(
    () => url,
    (url) => getJSON(url, options)
  );
  return { data, mutate, refetch, loading: () => data.loading, error: () => data.error };
}

export function useMutation(fn) {
  const [loading, setLoading] = createSignal(false);
  const [error, setError] = createSignal(null);

  async function mutate(...args) {
    try {
      setLoading(true);
      setError(null);
      const result = await fn(...args);
      return result;
    } catch (e) {
      setError(e);
      throw e;
    } finally {
      setLoading(false);
    }
  }

  return { mutate, loading, error };
}
