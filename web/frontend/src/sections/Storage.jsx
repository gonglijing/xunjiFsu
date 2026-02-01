import { useEffect, useState } from 'preact/hooks';
import { del, getJSON, postJSON, putJSON } from '../api';
import { Card } from '../components/cards';
import { useToast } from '../components/Toast';

const empty = { name: '', product_key: '', device_key: '', storage_days: 30, enabled: 1 };

export function Storage() {
  const toast = useToast();
  const [items, setItems] = useState([]);
  const [form, setForm] = useState(empty);
  const [editing, setEditing] = useState(null);
  const [loading, setLoading] = useState(true);
  const [cleaning, setCleaning] = useState(false);

  const load = () => {
    setLoading(true);
    getJSON('/api/storage')
      .then((res) => setItems(res.data || res))
      .catch(() => toast('error', '加载存储策略失败'))
      .finally(() => setLoading(false));
  };

  useEffect(() => { load(); }, []);

  const submit = (e) => {
    e.preventDefault();
    const payload = { ...form, storage_days: Number(form.storage_days), enabled: Number(form.enabled) };
    const req = editing ? putJSON(`/api/storage/${editing}`, payload) : postJSON('/api/storage', payload);
    req.then(() => {
      toast('success', editing ? '策略已更新' : '策略已创建');
      setForm(empty); setEditing(null); load();
    }).catch(() => toast('error', '保存失败'));
  };

  const remove = (id) => {
    if (!confirm('删除该策略？')) return;
    del(`/api/storage/${id}`)
      .then(() => { toast('success', '已删除'); load(); })
      .catch(() => toast('error', '删除失败'));
  };

  const cleanup = () => {
    setCleaning(true);
    postJSON('/api/storage/run', {})
      .then((res) => toast('success', `已清理 ${res.deleted_count ?? res.data?.deleted_count ?? 0} 条`))
      .catch(() => toast('error', '清理失败'))
      .finally(() => setCleaning(false));
  };

  return (
    <div class="grid" style="grid-template-columns: 3fr 1.4fr; gap:24px;">
      <Card
        title="存储策略"
        extra={<button class="btn" onClick={cleanup} disabled={cleaning}>{cleaning ? '清理中...' : '立即按策略清理'}</button>}
      >
        {loading ? (
          <div class="text-center" style="padding:48px; color:var(--text-muted);">
            <div class="loading-spinner" style="margin:0 auto 16px;"></div><div>加载中...</div>
          </div>
        ) : (
          <div class="table-container" style="max-height:520px; overflow:auto;">
            <table class="table">
              <thead>
                <tr>
                  <th>ID</th><th>名称</th><th>ProductKey</th><th>DeviceKey</th><th>存储天数</th><th>状态</th><th>操作</th>
                </tr>
              </thead>
              <tbody>
                {items.map((s) => (
                  <tr key={s.id}>
                    <td>{s.id}</td>
                    <td>{s.name}</td>
                    <td>{s.product_key || '-'}</td>
                    <td>{s.device_key || '-'}</td>
                    <td>{s.storage_days}</td>
                    <td><span class={`badge ${s.enabled === 1 ? 'badge-running' : 'badge-stopped'}`}>{s.enabled === 1 ? '启用' : '禁用'}</span></td>
                    <td class="flex" style="gap:8px;">
                      <button class="btn" onClick={() => { setEditing(s.id); setForm({ name: s.name, product_key: s.product_key, device_key: s.device_key, storage_days: s.storage_days, enabled: s.enabled }); }}>编辑</button>
                      <button class="btn btn-danger" onClick={() => remove(s.id)}>删除</button>
                    </td>
                  </tr>
                ))}
                {!items.length && <tr><td colSpan={7} style="text-align:center; padding:24px; color:var(--text-muted);">暂无策略</td></tr>}
              </tbody>
            </table>
          </div>
        )}
      </Card>

      <Card title={editing ? '编辑存储策略' : '新增存储策略'}>
        <form class="form" onSubmit={submit}>
          <div class="form-group">
            <label class="form-label">名称</label>
            <input class="form-input" value={form.name} onInput={(e)=>setForm({...form,name:e.target.value})} required />
          </div>
          <div class="grid" style="grid-template-columns: repeat(2, 1fr); gap:12px;">
            <div class="form-group">
              <label class="form-label">ProductKey</label>
              <input class="form-input" value={form.product_key} onInput={(e)=>setForm({...form,product_key:e.target.value})} />
            </div>
            <div class="form-group">
              <label class="form-label">DeviceKey</label>
              <input class="form-input" value={form.device_key} onInput={(e)=>setForm({...form,device_key:e.target.value})} />
            </div>
          </div>
          <div class="form-group">
            <label class="form-label">存储天数</label>
            <input class="form-input" type="number" value={form.storage_days} onInput={(e)=>setForm({...form,storage_days:+e.target.value})} min="1" />
          </div>
          <div class="form-group">
            <label class="form-label">状态</label>
            <select class="form-select" value={form.enabled} onChange={(e)=>setForm({...form,enabled:+e.target.value})}>
              <option value={1}>启用</option>
              <option value={0}>禁用</option>
            </select>
          </div>
          <div class="flex" style="gap:12px; margin-top:12px;">
            <button class="btn btn-primary" type="submit" style="flex:1">{editing ? '保存' : '创建'}</button>
            <button class="btn" type="button" style="flex:1" onClick={()=>{ setForm(empty); setEditing(null); }}>重置</button>
          </div>
        </form>
      </Card>
    </div>
  );
}
