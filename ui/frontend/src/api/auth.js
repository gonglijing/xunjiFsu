import { postJSON, unwrapData } from '../api';

export async function login(payload) {
  const res = await postJSON('/login', payload);
  return unwrapData(res, {});
}
