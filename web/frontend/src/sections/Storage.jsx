import { createSignal, createEffect, For } from 'solid-js';
import { del, getJSON, postJSON, putJSON } from '../api';
import { useToast } from '../components/Toast';
import Card from '../components/cards';

const empty = { name: '', storage_days: 30, enabled: 1 };

export function Storage() {
  const toast = useToast();
  const [items, setItems] = createSignal([]);
  const [loading, setLoading] = createSignal(true);
  const [form, setForm] = createSignal(empty);
  const [editing, setEditing] = createSignal(null);
  const [saving, setSaving] = createSignal(false);

  const load = () => {
    setLoading(true);
    getJSON('/api/storage')
      .then((res) => setItems(res.data || res))
      .catch(() => toast.show('error', '加载存储配置失败'))
      .finally(() => setLoading(false));
  };

  createEffect(() => {
    load();
  });

  const submit = (e) => {
    e.preventDefault();
    setSaving(true);
    const api = editing() ? putJSON(`/api/storage/${editing()}`, form()) : postJSON('/api/storage', form());
    api.then(() => { 
      toast.show('success', editing() ? '已更新' : '已创建'); 
      setForm(empty); 
      setEditing(null); 
      load(); 
    })
    .catch(() => toast.show('error', '操作失败'))
    .finally(() => setSaving(false));
  };

  const remove = (id) => {
    if (!confirm('删除该配置？')) return;
    del(`/api/storage/${id}`)
      .then(() => { toast.show('success', '已删除'); load(); })
      .catch(() => toast.show('error', '删除失败'));
  };

  const edit = (item) => {
    setEditing(item.id);
    setForm({ 
      name: item.name, 
      storage_days: item.storage_days, 
      enabled: item.enabled 
    });
  };

  const runCleanup = () => {
    postJSON('/api/storage/run', {})
      .then((res) => toast.show('success', `清理完成，删除 ${res.deleted_count} 条记录`))
      .catch(() => toast.show('error', '清理失败'));
  };

  return (
    <div class="grid" style="grid-template-columns: 3fr 1.4fr; gap:24px;">
      <Card title="存储策略列表" extra={<button class="btn" onClick={runCleanup}>立即清理</button>}>
        {loading() ? (
          <div class="text-center" style="padding:48px; color:var(--text-muted);">
            <div class="loading-spinner" style="margin:0 auto 16px;"></div>
            <div>加载中...</div>
          </div>
        ) : (
          <div class="table-container" style="max-height:520px; overflow:auto;">
            <table class="table">
              <thead>
                <tr>
                  <th>ID</th>
                  <th>名称</th>
                  <th>保留天数</th>
                  <th>状态</th>
                  <th>操作</th>
                </tr>
              </thead>
              <tbody>
                <For each={items()}>
                  {(s) => (
                    <tr>
                      <td>{s.id}</td>
                      <td>{s.name}</td>
                      <td>{s.storage_days}</td>
                      <td>
                        <span class={`badge ${s.enabled === 1 ? 'badge-running' : 'badge-stopped'}`}>
                          {s.enabled === 1 ? '启用' : '禁用'}
                        </span>
                      </td>
                      <td class="flex" style="gap:8px;">
                        <button class="btn" onClick={() => edit(s)}>编辑</button>
                        <button class="btn btn-danger" onClick={() => remove(s.id)}>删除</button>
                      </td>
                    </tr>
                  )}
                </For>
                <For each={items().length === 0 ? [1] : []}>
                  {() => (
                    <tr>
                      <td colSpan={5} style="text-align:center; padding:24px; color:var(--text-muted);">暂无配置</td>
                    </tr>
                  )}
                </For>
              </tbody>
            </table>
          </div>
        )}
      </Card>

      <Card title={editing() ? '编辑存储策略' : '新增存储策略'}>
        <form class="form" onSubmit={submit}>
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
            <label class="form-label">保留天数</label>
            <input 
              class="form-input" 
              type="number" 
              value={form().storage_days} 
              onInput={(e) => setForm({ ...form(), storage_days: +e.target.value })} 
              required 
            />
          </div>
          <div class="flex" style="gap:12px; margin-top:12px;">
            <button class="btn btn-primary" type="submit" disabled={saving()} style="flex:1">
              {saving() ? '保存中...' : (editing() ? '保存' : '创建')}
            </button>
            <button 
              class="btn" 
              type="button" 
              onClick={() => { setForm(empty); setEditing(null); }} 
              style="flex:1"
            >
              重置
            </button>
          </div>
        </form>
      </Card>
    </div>
  );
}
