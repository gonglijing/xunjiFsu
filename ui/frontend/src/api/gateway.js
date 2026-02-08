import { getJSON, putJSON, post, unwrapData } from '../api';

export async function getGatewayConfig() {
  const res = await getJSON('/api/gateway/config');
  return unwrapData(res, {});
}

export async function updateGatewayConfig(payload) {
  return putJSON('/api/gateway/config', payload);
}

export async function syncGatewayIdentityToNorthbound() {
  const res = await post('/api/gateway/northbound/sync-identity');
  return unwrapData(res, {});
}
