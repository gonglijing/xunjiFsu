import { useEffect, useState } from 'preact/hooks';
import { del, getJSON, postJSON } from '../api';
import { useToast } from '../components/Toast';
import { Card } from '../components/cards';

export function Drivers() {
  const toast = useToast();
  const [items, setItems] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const load = () => {
    setLoading(true);
    getJSON('/api/drivers')
      .then((res) => {
        setItems(res.data || res);
        setError('');
      })
      .catch(() => setError('加载驱动失败'))
      .finally(() => setLoading(false));
  };

  useEffect(() => { load(); }, []);

  const toggle = (id, enabled) => {
    postJSON(`/api/drivers/${id}`, { enabled: enabled ? 0 : 1 })
      .then(() => { toast('success', '状态已更新'); load(); })
      .catch(() => toast('error', '更新失败'));
  };

  const remove = (id) => {
    if (!confirm('删除该驱动？')) return;
    del(`/api/drivers/${id}`)
      .then(() => { toast('success', '已删除'); load(); })
      .catch(() => toast('error', '删除失败'));
  };

  const upload = (e) => {
    const file = e.target.files?.[0];
    if (!file) return;
    const fd = new FormData();
    fd.append('file', file);
    fetch('/api/drivers/upload', { method: 'POST', body: fd })
      .then((r) => r.json())
      .then(() => { toast('success', '上传成功'); load(); })
      .catch(() => toast('error', '上传失败'))
      .finally(() => { e.target.value = ''; });
  };

  return (
    <Card
      title="驱动管理"
      extra={<label class="btn btn-primary" style="cursor:pointer;">上传驱动<input type="file" accept=".wasm" style="display:none" onChange={upload} /></label>}
    >
      {loading ? (
        <div class="text-center" style="padding:48px; color:var(--text-muted);">
          <div class="loading-spinner" style="margin:0 auto 16px;"></div><div>加载中...</div>
        </div>
      ) : error ? (
        <div style="color:var(--accent-red); padding:16px 0;">{error}</div>
      ) : (
        <div class="table-container" style="max-height:520px; overflow:auto;">
          <table class="table">
            <thead><tr><th>ID</th><th>名称</th><th>版本</th><th>描述</th><th>状态</th><th>操作</th></tr></thead>
            <tbody>
              {items.map((d) => (
                <tr key={d.id}>
                  <td>{d.id}</td>
                  <td>{d.name}</td>
                  <td>{d.version}</td>
                  <td>{d.description}</td>
                  <td><span class={`badge ${d.enabled === 1 ? 'badge-running' : 'badge-stopped'}`}>{d.enabled === 1 ? '启用' : '禁用'}</span></td>
                  <td class="flex" style="gap:8px;">
                    <button class="btn" onClick={() => toggle(d.id, d.enabled === 1)}>{d.enabled === 1 ? '禁用' : '启用'}</button>
                    <button class="btn btn-danger" onClick={() => remove(d.id)}>删除</button>
                    <a class="btn" href={`/api/drivers/${d.id}/download`}>下载</a>
                  </td>
                </tr>
              ))}
              {!items.length && <tr><td colSpan={6} style="text-align:center; padding:24px; color:var(--text-muted);">暂无驱动</td></tr>}
            </tbody>
          </table>
        </div>
      )}
    </Card>
  );
}
