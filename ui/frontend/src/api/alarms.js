import { del, getJSON, postJSON, unwrapData } from '../api';

export async function listAlarms() {
  const res = await getJSON('/api/alarms');
  return unwrapData(res, []);
}

export async function acknowledgeAlarm(id) {
  return postJSON(`/api/alarms/${id}/acknowledge`, {});
}

export async function deleteAlarm(id) {
  return del(`/api/alarms/${id}`);
}

export async function batchDeleteAlarms(ids) {
  return postJSON('/api/alarms/batch-delete', { ids });
}

export async function clearAlarms() {
  return del('/api/alarms');
}
