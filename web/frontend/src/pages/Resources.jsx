import { useEffect, useState } from 'preact/hooks';
import { del, getJSON, postJSON, putJSON } from '../api';
import { Card } from '../components/cards';
import { useToast } from '../components/Toast';

const empty = { name: '', type: 'serial', path: '', enabled: 1 };

export function Resources() {
  const toast = useToast();
  const [items, setItems] = useState([]);
  const [form, setForm] = useState(empty);
  const [editing, setEditing] = useState(null);

  const load = () => {
    getJSON('/api/resources').then((res) => setItems(res.data || res)).catch(() => toast('error', '加载资源失败'));
  };

  useEffect(() => { load(); }, []);

  const submit = (e) => {
    e.preventDefault();
    const api = editing ? putJSON(`/api/resources/${editing}`, form) : postJSON('/api/resources', form);
    api.then(() => {
      toast('success', editing ? '资源已更新' : '资源已创建');
      setForm(empty); setEditing(null); load();
    }).catch(() => toast('error', '保存失败'));
  };

  const remove = (id) => {
    if (!confirm('删除该资源？')) return;
    del(`/api/resources/${id}`).then(() => { toast('success','已删除'); load(); }).catch(()=>toast('error','删除失败'));
  };

  const toggle = (item) => {
    postJSON(`/api/resources/${item.id}/toggle`, {}).then(load).catch(()=>toast('error','切换失败'));
  };

  return (
    <div class="grid" style={{ gridTemplateColumns: '3fr 1.4fr', gap: '24px' }}>
      <Card title="资源列表">
        <div class="table-container" style="max-height:520px; overflow:auto;">
          <table class="table">
            <thead><tr><th>ID</th><th>名称</th><th>类型</th><th>路径</th><th>状态</th><th>操作</th></tr></thead>
            <tbody>
              {items.map((r)=> (
                <tr key={r.id}>
                  <td>{r.id}</td><td>{r.name}</td><td>{r.type}</td><td>{r.path}</td>
                  <td><span class={`badge ${r.enabled===1?'badge-running':'badge-stopped'}`}>{r.enabled===1?'启用':'禁用'}</span></td>
                  <td class="flex" style="gap:8px;">
                    <button class="btn" onClick={()=>{setEditing(r.id); setForm({name:r.name,type:r.type,path:r.path,enabled:r.enabled});}}>编辑</button>
                    <button class="btn" onClick={()=>toggle(r)}>{r.enabled===1?'禁用':'启用'}</button>
                    <button class="btn btn-danger" onClick={()=>remove(r.id)}>删除</button>
                  </td>
                </tr>
              ))}
              {!items.length && <tr><td colSpan={6} style="text-align:center; padding:24px; color:var(--text-muted);">暂无资源</td></tr>}
            </tbody>
          </table>
        </div>
      </Card>

      <Card title={editing ? '编辑资源' : '新增资源'}>
        <form class="form" onSubmit={submit}>
          <div class="form-group">
            <label class="form-label">名称</label>
            <input class="form-input" value={form.name} onInput={(e)=>setForm({...form,name:e.target.value})} required />
          </div>
          <div class="form-group">
            <label class="form-label">类型</label>
            <select class="form-select" value={form.type} onChange={(e)=>setForm({...form,type:e.target.value})}>
              <option value="serial">串口</option>
              <option value="net">网口</option>
              <option value="di">DI</option>
              <option value="do">DO</option>
            </select>
          </div>
          <div class="form-group">
            <label class="form-label">路径</label>
            <input class="form-input" value={form.path} onInput={(e)=>setForm({...form,path:e.target.value})} placeholder="如 /dev/ttyUSB0 或 eth0" required />
          </div>
          <div class="flex" style={{ gap:'12px', marginTop:'12px' }}>
            <button class="btn btn-primary" type="submit" style={{ flex:1 }}>{editing ? '保存' : '创建'}</button>
            <button class="btn" type="button" onClick={()=>{setForm(empty); setEditing(null);}} style={{ flex:1 }}>重置</button>
          </div>
        </form>
      </Card>
    </div>
  );
}
