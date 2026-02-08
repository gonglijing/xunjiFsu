import { getJSON, post, postJSON, putJSON, del, unwrapData } from '../api';

export async function listNorthboundConfigs() {
  const res = await getJSON('/api/northbound');
  return unwrapData(res, []);
}

export async function listNorthboundStatus() {
  const res = await getJSON('/api/northbound/status');
  return unwrapData(res, []);
}

export async function getNorthboundSchema(type) {
  const res = await getJSON(`/api/northbound/schema?type=${type}`);
  return unwrapData(res, {});
}

export async function createNorthboundConfig(payload) {
  return postJSON('/api/northbound', payload);
}

export async function updateNorthboundConfig(id, payload) {
  return putJSON(`/api/northbound/${id}`, payload);
}

export async function deleteNorthboundConfig(id) {
  return del(`/api/northbound/${id}`);
}

export async function toggleNorthboundConfig(id) {
  return postJSON(`/api/northbound/${id}/toggle`, {});
}

export async function reloadNorthboundConfig(id) {
  return post(`/api/northbound/${id}/reload`);
}

export async function syncGatewayIdentityToNorthbound() {
  const res = await post('/api/gateway/northbound/sync-identity');
  return unwrapData(res, {});
}
