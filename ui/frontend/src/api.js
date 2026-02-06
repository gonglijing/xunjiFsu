const TOKEN_KEY = 'gogw_jwt';

function getToken() {
  return localStorage.getItem(TOKEN_KEY) || '';
}

function authHeaders(headers = {}) {
  const token = getToken();
  return token ? { ...headers, Authorization: `Bearer ${token}` } : headers;
}

function handleAuth(res) {
  if (res.status === 401) {
    localStorage.removeItem(TOKEN_KEY);
    window.location.href = '/login';
    throw new Error('unauthorized');
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
      const message = json.error || json.message || 'request failed';
      throw new Error(message);
    }
    return json;
  } catch (e) {
    throw new Error('invalid json');
  }
}

export function storeToken(token) {
  if (token) localStorage.setItem(TOKEN_KEY, token);
}

export async function getJSON(url, options = {}) {
  const res = await authFetch(url, options);
  if (!res.ok) throw new Error(`${res.status} ${res.statusText}`);
  return parseJSON(res);
}

export async function postJSON(url, body, options = {}) {
  const res = await authFetch(url, {
    method: 'POST',
    headers: authHeaders({ 'Content-Type': 'application/json', ...(options.headers || {}) }),
    body: JSON.stringify(body),
    ...options,
  });
  if (!res.ok) throw new Error(`${res.status} ${res.statusText}`);
  return parseJSON(res);
}

export async function post(url, options = {}) {
  const res = await authFetch(url, {
    method: 'POST',
    ...options,
  });
  if (!res.ok) throw new Error(`${res.status} ${res.statusText}`);
  return parseJSON(res);
}

export async function putJSON(url, body, options = {}) {
  const res = await authFetch(url, {
    method: 'PUT',
    headers: authHeaders({ 'Content-Type': 'application/json', ...(options.headers || {}) }),
    body: JSON.stringify(body),
    ...options,
  });
  if (!res.ok) throw new Error(`${res.status} ${res.statusText}`);
  return parseJSON(res);
}

export async function del(url) {
  const res = await authFetch(url, { method: 'DELETE' });
  if (!res.ok && res.status !== 204) throw new Error(`${res.status} ${res.statusText}`);
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
  if (!res.ok) throw new Error(`${res.status} ${res.statusText}`);
  return parseJSON(res);
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
