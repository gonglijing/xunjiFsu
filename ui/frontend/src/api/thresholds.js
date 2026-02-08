import { getJSON, postJSON, del, unwrapData } from '../api';

export async function listThresholds() {
  const res = await getJSON('/api/thresholds');
  return unwrapData(res, []);
}

export async function createThreshold(payload) {
  return postJSON('/api/thresholds', payload);
}

export async function deleteThreshold(id) {
  return del(`/api/thresholds/${id}`);
}
