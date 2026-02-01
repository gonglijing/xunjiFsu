import { useState } from 'preact/hooks';
import { postJSON } from '../api';

export function Login({ onSuccess }) {
  const [username, setUsername] = useState('admin');
  const [password, setPassword] = useState('123456');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const submit = (e) => {
    e.preventDefault();
    setLoading(true);
    setError('');
    postJSON('/login', { username, password })
      .then(() => onSuccess())
      .catch(() => setError('用户名或密码错误'))
      .finally(() => setLoading(false));
  };

  return (
    <div class="container" style="max-width:420px; margin-top:48px;">
      <div class="card">
        <div class="card-header"><h3 class="card-title">登录</h3></div>
        <form class="form" onSubmit={submit}>
          <div class="form-group">
            <label class="form-label">用户名</label>
            <input class="form-input" value={username} onInput={(e)=>setUsername(e.target.value)} required />
          </div>
          <div class="form-group">
            <label class="form-label">密码</label>
            <input class="form-input" type="password" value={password} onInput={(e)=>setPassword(e.target.value)} required />
          </div>
          {error && <div style="color:var(--accent-red); padding:8px 0;">{error}</div>}
          <button class="btn btn-primary" type="submit" disabled={loading} style="width:100%; margin-top:8px;">
            {loading ? '登录中...' : '登录'}
          </button>
        </form>
      </div>
    </div>
  );
}
