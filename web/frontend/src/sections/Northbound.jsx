import { useEffect, useState } from 'preact/hooks';
import { del, getJSON, postJSON, putJSON } from '../api';
import { useToast } from '../components/Toast';
import { Card } from '../components/cards';

const empty = { name: '', type: 'http', upload_interval: 5000, config: '{}', enabled: 1 };

export function Northbound() {
  const toast = useToast();
  const [items, setItems] = useState([]);
  const [loading, setLoading] = useState(true);
  const [form, setForm] = useState(empty);

  const load = () => {
    setLoading(true);
    getJSON('/api/northbound')
      .then((res) => setItems(res.data || res))
      .catch(() => toast('error', '加载北向配置失败'))
      .finally(() => setLoading(false));
  };

  useEffect(() => { load(); }, []);

  const submit = (e) => {
    e.preventDefault();
    postJSON('/api/northbound', form)
      .then(() => { toast('success', '已创建'); setForm(empty); load(); })
      .catch(() => toast('error', '创建失败'));
  };

  const toggle = (id, enabled) => {
    postJSON(`/api/northbound/${id}/toggle`, {})
      .then(() => load())
      .catch(() => toast('error', '切换失败'));
  };

  const remove = (id) => {
    if (!confirm('删除该配置？')) return;
    del(`/api/northbound/${id}`)
      .then(() => { toast('success', '已删除'); load(); })
      .catch(() => toast('error', '删除失败'));
  };

  return (
    <div class="grid" style="grid-template-columns: 3fr 1.4fr; gap:24px;">
      <Card title="北向配置列表">
        {loading ? (
          <div class="text-center" style="padding:48px; color:var(--text-muted);">
            <div class="loading-spinner" style="margin:0 auto 16px;"></div><div>加载中...</div>
          </div>
        ) : (
          <div class="table-container" style="max-height:520px; overflow:auto;">
            <table class="table">
              <thead><tr><th>ID</th><th>名称</th><th>类型</th><th>上传间隔(ms)</th><th>状态</th><th>操作</th></tr></thead>
              <tbody>
                {items.map((n) => (
                  <tr key={n.id}>
                    <td>{n.id}</td><td>{n.name}</td><td>{n.type}</td><td>{n.upload_interval}</td>
                    <td><span class={`badge ${n.enabled === 1 ? 'badge-running' : 'badge-stopped'}`}>{n.enabled === 1 ? '启用' : '禁用'}</span></td>
                    <td class="flex" style="gap:8px;">
                      <button class="btn" onClick={() => toggle(n.id, n.enabled === 1)}>{n.enabled === 1 ? '禁用' : '启用'}</button>
                      <button class="btn btn-danger" onClick={() => remove(n.id)}>删除</button>
                    </td>
                  </tr>
                ))}
                {!items.length && <tr><td colSpan={6} style="text-align:center; padding:24px; color:var(--text-muted);">暂无配置</td></tr>}
              </tbody>
            </table>
          </div>
        )}
      </Card>

      <Card title="新增北向配置">
        <form class="form" onSubmit={submit}>
          <div class="form-group">
            <label class="form-label">名称</label>
            <input class="form-input" value={form.name} onInput={(e)=>setForm({...form,name:e.target.value})} required />
          </div>
          <div class="form-group">
            <label class="form-label">类型</label>
            <select class="form-select" value={form.type} onChange={(e)=>setForm({...form,type:e.target.value})}>
              <option value="http">HTTP</option>
              <option value="mqtt">MQTT</option>
              <option value="xunji">寻迹</option>
            </select>
          </div>
          <div class="form-group">
            <label class="form-label">上传间隔 (ms)</label>
            <input class="form-input" type="number" value={form.upload_interval} onInput={(e)=>setForm({...form,upload_interval:+e.target.value})} required />
          </div>
          <div class="form-group">
            <label class="form-label">配置 (JSON)</label>
            <textarea class="form-input" rows={6} value={form.config} onInput={(e)=>setForm({...form,config:e.target.value})} placeholder='{ "url": "http://..." }'></textarea>
          </div>
          <div class="flex" style="gap:12px; margin-top:12px;">
            <button class="btn btn-primary" type="submit" style="flex:1">创建</button>
            <button class="btn" type="button" onClick={()=>setForm(empty)} style="flex:1">重置</button>
          </div>
        </form>
      </Card>
    </div>
  );
}
