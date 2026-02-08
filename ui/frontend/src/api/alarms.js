import { getJSON, postJSON, unwrapData } from '../api';

export async function listAlarms() {
  const res = await getJSON('/api/alarms');
  return unwrapData(res, []);
}

export async function acknowledgeAlarm(id) {
  return postJSON(`/api/alarms/${id}/acknowledge`, {});
}
