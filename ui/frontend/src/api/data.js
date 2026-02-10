import { del, getJSON, unwrapData } from '../api';

export async function listDataCache() {
  const res = await getJSON('/api/data');
  return unwrapData(res, []);
}

export async function getDataCacheByDevice(deviceId) {
  const res = await getJSON(`/api/data/cache/${deviceId}`);
  return unwrapData(res, []);
}

export async function getHistoryData(params) {
  const query = new URLSearchParams(params || {}).toString();
  const suffix = query ? `?${query}` : '';
  const res = await getJSON(`/api/data/history${suffix}`);
  return unwrapData(res, []);
}

export async function clearHistoryData(params) {
  const query = new URLSearchParams(params || {}).toString();
  const suffix = query ? `?${query}` : '';
  return del(`/api/data/history${suffix}`);
}
