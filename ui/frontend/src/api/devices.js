import { getJSON, postJSON, putJSON, del, unwrapData } from '../api';

export async function listDevices() {
  const res = await getJSON('/api/devices');
  return unwrapData(res, []);
}

export async function listResources() {
  const res = await getJSON('/api/resources');
  return unwrapData(res, []);
}

export async function listDrivers() {
  const res = await getJSON('/api/drivers');
  return unwrapData(res, []);
}

export async function createDevice(payload) {
  return postJSON('/api/devices', payload);
}

export async function updateDevice(id, payload) {
  return putJSON(`/api/devices/${id}`, payload);
}

export async function deleteDevice(id) {
  return del(`/api/devices/${id}`);
}

export async function toggleDevice(id) {
  return postJSON(`/api/devices/${id}/toggle`, {});
}

export async function listWritables(deviceId) {
  return getJSON(`/api/devices/${deviceId}/writables`);
}

export async function executeDeviceFunction(deviceId, payload) {
  return postJSON(`/api/devices/${deviceId}/execute`, payload);
}

