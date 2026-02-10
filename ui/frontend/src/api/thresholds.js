import { getJSON, postJSON, putJSON, del, unwrapData } from '../api';

export async function listThresholds() {
  const res = await getJSON('/api/thresholds');
  return unwrapData(res, []);
}

export async function createThreshold(payload) {
  return postJSON('/api/thresholds', payload);
}

export async function updateThreshold(id, payload) {
  return putJSON(`/api/thresholds/${id}`, payload);
}

export async function deleteThreshold(id) {
  return del(`/api/thresholds/${id}`);
}

export async function getAlarmRepeatInterval() {
  const res = await getJSON('/api/thresholds/repeat-interval');
  return unwrapData(res, { seconds: 60 });
}

export async function updateAlarmRepeatInterval(seconds) {
  return postJSON('/api/thresholds/repeat-interval', { seconds });
}
