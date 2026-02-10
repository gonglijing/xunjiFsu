import { getJSON, putJSON, post, unwrapData } from '../api';

export async function getGatewayConfig() {
  const res = await getJSON('/api/gateway/config');
  return unwrapData(res, {});
}

export async function updateGatewayConfig(payload) {
  return putJSON('/api/gateway/config', payload);
}

export async function getGatewayRuntimeConfig() {
  const res = await getJSON('/api/gateway/runtime');
  return unwrapData(res, {});
}

export async function updateGatewayRuntimeConfig(payload) {
  const res = await putJSON('/api/gateway/runtime', payload);
  return unwrapData(res, {});
}

export async function getGatewayRuntimeAudits(limit = 20) {
  const res = await getJSON(`/api/gateway/runtime/audits?limit=${encodeURIComponent(limit)}`);
  return unwrapData(res, []);
}
