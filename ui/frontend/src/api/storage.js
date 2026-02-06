import { getJSON, postJSON, putJSON, del, unwrapData } from '../api';

export async function listStorageConfigs() {
  const res = await getJSON('/api/storage');
  return unwrapData(res, []);
}

export async function createStorageConfig(payload) {
  return postJSON('/api/storage', payload);
}

export async function updateStorageConfig(id, payload) {
  return putJSON(`/api/storage/${id}`, payload);
}

export async function deleteStorageConfig(id) {
  return del(`/api/storage/${id}`);
}

export async function cleanupByPolicy() {
  return postJSON('/api/storage/run', {});
}

