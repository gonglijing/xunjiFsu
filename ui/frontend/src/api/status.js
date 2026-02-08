import { getJSON, unwrapData } from '../api';

export async function getStatus() {
  const res = await getJSON('/api/status');
  return unwrapData(res, null);
}
