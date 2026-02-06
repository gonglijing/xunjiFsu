import { createSignal, createEffect, Show } from 'solid-js';
import { del, getJSON, post, postJSON, putJSON, unwrapData } from '../api';
import { useToast } from '../components/Toast';
import Card from '../components/cards';
import CrudTable from '../components/CrudTable';

const empty = { name: '', type: 'http', upload_interval: 5000, config: '{}', enabled: 1 };

export function Northbound() {
  const toast = useToast();
  const [items, setItems] = createSignal([]);
  const [runtime, setRuntime] = createSignal([]);
  const [loading, setLoading] = createSignal(true);
  const [form, setForm] = createSignal(empty);
  const [editing, setEditing] = createSignal(null);
  const [showModal, setShowModal] = createSignal(false);
  const [saving, setSaving] = createSignal(false);
  const [syncing, setSyncing] = createSignal(false);

  const runtimeByName = () => {
    const map = {};
    for (const item of runtime()) {
      if (item?.name) map[item.name] = item;
    }
    return map;
  };

  const load = () => {
    setLoading(true);
    Promise.all([
      getJSON('/api/northbound'),
      getJSON('/api/northbound/status'),
    ])
      .then(([configs, status]) => {
        setItems(unwrapData(configs, []));
        setRuntime(unwrapData(status, []));
      })
      .catch(() => toast.show('error', '加载北向配置失败'))
      .finally(() => setLoading(false));
  };

  createEffect(() => {
    load();
  });

  const submit = (e) => {
    e.preventDefault();
    setSaving(true);
    const api = editing() ? putJSON(`/api/northbound/${editing()}`, form()) : postJSON('/api/northbound', form());
    api.then(() => { 
      toast.show('success', editing() ? '已更新' : '已创建'); 
      setForm(empty); 
      setEditing(null); 
      setShowModal(false); 
      load(); 
    })
    .catch(() => toast.show('error', '操作失败'))
    .finally(() => setSaving(false));
  };

  const toggle = (id) => {
    postJSON(`/api/northbound/${id}/toggle`, {})
      .then(load)
      .catch(() => toast.show('error', '切换失败'));
  };

  const reload = (id) => {
    post(`/api/northbound/${id}/reload`)
      .then(() => {
        toast.show('success', '重载成功');
        load();
      })
      .catch(() => toast.show('error', '重载失败'));
  };

  const syncGatewayIdentity = () => {
    setSyncing(true);
    post('/api/gateway/northbound/sync-identity')
      .then((res) => {
        const data = unwrapData(res, {});
        const updated = data.updated?.length || 0;
        const failed = data.failed ? Object.keys(data.failed).length : 0;
        toast.show('success', `同步完成：更新 ${updated} 个，失败 ${failed} 个`);
        load();
      })
      .catch((err) => toast.show('error', err?.message || '同步失败'))
      .finally(() => setSyncing(false));
  };

  const remove = (id) => {
    if (!confirm('删除该配置？')) return;
    del(`/api/northbound/${id}`)
      .then(() => { toast.show('success', '已删除'); load(); })
      .catch(() => toast.show('error', '删除失败'));
  };

  const openCreate = () => {
    setForm(empty);
    setEditing(null);
    setShowModal(true);
  };

  const edit = (item) => {
    setEditing(item.id);
    setForm({ name: item.name, type: item.type, upload_interval: item.upload_interval, config: item.config, enabled: item.enabled });
    setShowModal(true);
  };

  return (
    <div>
      <Card
        title="北向配置列表"
        extra={
          <div class="flex" style="gap:8px;">
            <button class="btn" onClick={syncGatewayIdentity} disabled={syncing()}>
              {syncing() ? '同步中...' : '同步网关身份'}
            </button>
            <button class="btn btn-primary" onClick={openCreate}>
              新增配置
            </button>
          </div>
        }
      >
        {loading() ? (
          <div class="text-center" style="padding:48px; color:var(--text-muted);">
            <div class="loading-spinner" style="margin:0 auto 16px;"></div>
            <div>加载中...</div>
          </div>
        ) : (
          <CrudTable
            style="max-height:520px; overflow:auto;"
            loading={loading()}
            items={items()}
            emptyText="暂无配置"
            columns={[
              { key: 'id', title: 'ID' },
              { key: 'name', title: '名称' },
              {
                key: 'type',
                title: '类型',
                render: (n) => (
                  <span class="badge badge-info">{n.type.toUpperCase()}</span>
                ),
              },
              { key: 'upload_interval', title: '上传间隔(ms)' },
              {
                key: 'enabled',
                title: '状态',
                render: (n) => (
                  <span class={`badge ${n.enabled === 1 ? 'badge-running' : 'badge-stopped'}`}>
                    {n.enabled === 1 ? '启用' : '禁用'}
                  </span>
                ),
              },
              {
                key: 'runtime',
                title: '运行态',
                render: (n) => {
                  const rt = runtimeByName()[n.name] || n.runtime || {};
                  const registered = rt.registered ? '已注册' : '未注册';
                  const breaker = rt.breaker_state || 'closed';
                  return (
                    <div style="font-size:12px; color:var(--text-secondary); line-height:1.5;">
                      <div>{registered} / {rt.enabled ? '运行' : '停止'}</div>
                      <div>熔断: {breaker}</div>
                    </div>
                  );
                },
              },
            ]}
            renderActions={(n) => (
              <div class="flex" style="gap:8px;">
                <button class="btn" onClick={() => edit(n)}>编辑</button>
                <button class="btn" onClick={() => toggle(n.id)}>{n.enabled === 1 ? '禁用' : '启用'}</button>
                <button class="btn" onClick={() => reload(n.id)}>重载</button>
                <button class="btn btn-danger" onClick={() => remove(n.id)}>删除</button>
              </div>
            )}
          />
        )}
      </Card>

      <Show when={showModal()}>
        <div class="modal-backdrop" style="position:fixed; inset:0; background:rgba(0,0,0,0.45); display:flex; align-items:center; justify-content:center; z-index:1000;">
          <div class="card" style="width:480px; max-width:90vw;">
            <div class="card-header">
              <h3 class="card-title">{editing() ? '编辑北向配置' : '新增北向配置'}</h3>
              <button class="btn btn-ghost" onClick={() => { setShowModal(false); setEditing(null); setForm(empty); }} style="padding:4px 8px;">✕</button>
            </div>
            <form class="form" onSubmit={submit} style="padding:12px 16px 16px;">
              <div class="form-group">
                <label class="form-label">名称</label>
                <input 
                  class="form-input" 
                  value={form().name} 
                  onInput={(e) => setForm({ ...form(), name: e.target.value })} 
                  placeholder="配置名称" 
                  required 
                />
              </div>
              <div class="grid" style="grid-template-columns: 1fr 1fr; gap:12px;">
                <div class="form-group">
                  <label class="form-label">类型</label>
                  <select 
                    class="form-select" 
                    value={form().type} 
                    onChange={(e) => setForm({ ...form(), type: e.target.value })}
                  >
                    <option value="http">HTTP</option>
                    <option value="mqtt">MQTT</option>
                    <option value="xunji">寻迹</option>
                  </select>
                </div>
                <div class="form-group">
                  <label class="form-label">上传间隔 (ms)</label>
                  <input 
                    class="form-input" 
                    type="number" 
                    value={form().upload_interval} 
                    onInput={(e) => setForm({ ...form(), upload_interval: +e.target.value })} 
                    required 
                  />
                </div>
              </div>
              <div class="form-group">
                <label class="form-label">配置 (JSON)</label>
                <textarea 
                  class="form-input" 
                  rows={5} 
                  value={form().config} 
                  onInput={(e) => setForm({ ...form(), config: e.target.value })} 
                  placeholder='{ "url": "http://...", "method": "POST" }'
                  style="font-family:monospace; font-size:13px;"
                ></textarea>
              </div>
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
