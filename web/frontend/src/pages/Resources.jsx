import { createSignal, createEffect, onCleanup, Show, For } from 'solid-js';
import { getJSON, del, postJSON, putJSON } from '../api';
import Card, { SectionTabs } from '../components/cards';
import { useToast } from '../components/Toast';

const empty = { name: '', type: 'serial', path: '', enabled: 1 };

function Resources() {
  const toast = useToast();
  const [items, setItems] = createSignal([]);
  const [form, setForm] = createSignal(empty);
  const [editing, setEditing] = createSignal(null);
  const [showModal, setShowModal] = createSignal(false);
  const [saving, setSaving] = createSignal(false);
  const [err, setErr] = createSignal('');

  const load = () => {
    getJSON('/api/resources')
      .then((res) => setItems(res.data || res))
      .catch(() => toast.show('error', '加载资源失败'));
  };

  createEffect(() => {
    load();
  });

  const submit = (e) => {
    e.preventDefault();
    setSaving(true);
    setErr('');
    const api = editing() ? putJSON(`/api/resources/${editing()}`, form()) : postJSON('/api/resources', form());
    api.then(() => {
      toast.show('success', editing() ? '资源已更新' : '资源已创建');
      setForm(empty);
      setEditing(null);
      setShowModal(false);
      load();
    }).catch((er) => {
      setErr(er.message || '保存失败');
      toast.show('error', '保存失败');
    }).finally(() => setSaving(false));
  };

  const remove = (id) => {
    if (!confirm('删除该资源？')) return;
    del(`/api/resources/${id}`)
      .then(() => { toast.show('success', '已删除'); load(); })
      .catch(() => toast.show('error', '删除失败'));
  };

  const toggle = (item) => {
    postJSON(`/api/resources/${item.id}/toggle`, {})
      .then(load)
      .catch(() => toast.show('error', '切换失败'));
  };

  return (
    <div>
      <Card
        title="资源列表"
        extra={
          <button class="btn btn-primary" onClick={() => { setForm(empty); setEditing(null); setErr(''); setShowModal(true); }}>
            新增资源
          </button>
        }
      >
        <div class="table-container" style="max-height:520px; overflow:auto;">
          <table class="table">
            <thead>
              <tr>
                <th>ID</th>
                <th>名称</th>
                <th>类型</th>
                <th>路径</th>
                <th>状态</th>
                <th>操作</th>
              </tr>
            </thead>
            <tbody>
              <For each={items()}>
                {(r) => (
                  <tr>
                    <td>{r.id}</td>
                    <td>{r.name}</td>
                    <td>{r.type}</td>
                    <td>{r.path}</td>
                    <td>
                      <span class={`badge ${r.enabled === 1 ? 'badge-running' : 'badge-stopped'}`}>
                        {r.enabled === 1 ? '启用' : '禁用'}
                      </span>
                    </td>
                    <td class="flex" style="gap:8px;">
                      <button 
                        class="btn" 
                        onClick={() => { setEditing(r.id); setForm({ name: r.name, type: r.type, path: r.path, enabled: r.enabled }); setErr(''); setShowModal(true); }}
                      >
                        编辑
                      </button>
                      <button class="btn" onClick={() => toggle(r)}>{r.enabled === 1 ? '禁用' : '启用'}</button>
                      <button class="btn btn-danger" onClick={() => remove(r.id)}>删除</button>
                    </td>
                  </tr>
                )}
              </For>
              <Show when={items().length === 0}>
                <tr>
                  <td colSpan={6} style="text-align:center; padding:24px; color:var(--text-muted);">暂无资源</td>
                </tr>
              </Show>
            </tbody>
          </table>
        </div>
      </Card>

      <Show when={showModal()}>
        <div class="modal-backdrop" style="position:fixed; inset:0; background:rgba(0,0,0,0.45); display:flex; align-items:center; justify-content:center; z-index:1000;">
          <div class="card" style="width:420px; max-width:90vw;">
            <div class="card-header">
              <h3 class="card-title">{editing() ? '编辑资源' : '新增资源'}</h3>
            </div>
            <form class="form" onSubmit={submit} style="padding:12px 16px 16px;">
              <div class="form-group">
                <label class="form-label">名称</label>
                <input 
                  class="form-input" 
                  value={form().name} 
                  onInput={(e) => setForm({ ...form(), name: e.target.value })} 
                  required 
                />
              </div>
              <div class="form-group">
                <label class="form-label">类型</label>
                <select 
                  class="form-select" 
                  value={form().type} 
                  onChange={(e) => setForm({ ...form(), type: e.target.value })}
                >
                  <option value="serial">串口</option>
                  <option value="net">网口</option>
                  <option value="di">DI</option>
                  <option value="do">DO</option>
                </select>
              </div>
              <div class="form-group">
                <label class="form-label">路径</label>
                <input 
                  class="form-input" 
                  value={form().path} 
                  onInput={(e) => setForm({ ...form(), path: e.target.value })} 
                  placeholder="如 /dev/ttyUSB0 或 eth0" 
                  required 
                />
              </div>
              <Show when={err()}>
                <div style="color:var(--accent-red); padding:4px 0;">{err()}</div>
              </Show>
              <div class="flex" style={{ gap: '8px', justifyContent: 'flex-end', marginTop: '8px' }}>
                <button 
                  type="button" 
                  class="btn" 
                  onClick={() => { setShowModal(false); setEditing(null); setForm(empty); }} 
                  disabled={saving()}
                >
                  取消
                </button>
                <button type="submit" class="btn btn-primary" disabled={saving()}>
                  {saving() ? '保存中...' : (editing() ? '保存' : '创建')}
                </button>
              </div>
            </form>
          </div>
        </div>
      </Show>
    </div>
  );
}

export default Resources;
