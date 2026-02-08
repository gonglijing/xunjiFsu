import { getJSON, postJSON, putJSON, del, unwrapData } from '../api';

export async function listResources() {
  const res = await getJSON('/api/resources');
  return unwrapData(res, []);
}

export async function createResource(payload) {
  return postJSON('/api/resources', payload);
}

export async function updateResource(id, payload) {
  return putJSON(`/api/resources/${id}`, payload);
}

export async function deleteResource(id) {
  return del(`/api/resources/${id}`);
}

export async function toggleResource(id) {
  return postJSON(`/api/resources/${id}/toggle`, {});
}
