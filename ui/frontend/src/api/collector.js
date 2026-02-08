import { postJSON } from '../api';

export async function startCollector() {
  return postJSON('/api/collector/start', {});
}

export async function stopCollector() {
  return postJSON('/api/collector/stop', {});
}
