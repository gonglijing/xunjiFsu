import { getJSON, del, post, upload as uploadWithAuth, unwrapData } from '../api';

export async function listDrivers() {
  const res = await getJSON('/api/drivers');
  return unwrapData(res, []);
}

export async function deleteDriver(id) {
  return del(`/api/drivers/${id}`);
}

export async function reloadDriver(id) {
  return post(`/api/drivers/${id}/reload`);
}

export async function uploadDriver(file) {
  const fd = new FormData();
  fd.append('file', file);
  return uploadWithAuth('/api/drivers/upload', fd);
}

