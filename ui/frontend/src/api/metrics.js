import { getJSON } from '../api';

export async function getMetrics() {
  return getJSON('/metrics');
}
