function handleAuth(res) {
  if (res.status === 401) {
    window.location.href = '/login';
    throw new Error('unauthorized');
  }
}

export async function getJSON(url, options = {}) {
  const res = await fetch(url, { credentials: 'same-origin', ...options });
  handleAuth(res);
  if (!res.ok) throw new Error(res.statusText);
  return res.json();
}

export async function postJSON(url, body, options = {}) {
  const res = await fetch(url, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...(options.headers || {}) },
    credentials: 'same-origin',
    body: JSON.stringify(body),
    ...options,
  });
  handleAuth(res);
  if (!res.ok) throw new Error(res.statusText);
  return res.json();
}

export async function putJSON(url, body, options = {}) {
  const res = await fetch(url, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json', ...(options.headers || {}) },
    credentials: 'same-origin',
    body: JSON.stringify(body),
    ...options,
  });
  handleAuth(res);
  if (!res.ok) throw new Error(res.statusText);
  return res.json();
}

export async function del(url) {
  const res = await fetch(url, { method: 'DELETE', credentials: 'same-origin' });
  handleAuth(res);
  if (!res.ok) throw new Error(res.statusText);
  return res.json();
}
