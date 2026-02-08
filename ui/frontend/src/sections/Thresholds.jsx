import { createSignal, createEffect, For, Show } from 'solid-js';
import { devicesAPI, thresholdsAPI } from '../api/services';
import { useToast } from '../components/Toast';
import Card from '../components/cards';

const empty = { device_id: '', field_name: '', operator: '>', value: 0, severity: 'warning', message: '', enabled: 1 };

export function Thresholds() {
  const toast = useToast();
  const [items, setItems] = createSignal([]);
  const [devices, setDevices] = createSignal([]);
  const [form, setForm] = createSignal(empty);
  const [showModal, setShowModal] = createSignal(false);
  const [saving, setSaving] = createSignal(false);
  const [err, setErr] = createSignal('');

  const load = () => {
    thresholdsAPI.listThresholds()
      .then((res) => setItems(res || []))
      .catch(() => toast.show('error', '加载阈值失败'));
  };

  const loadDevices = () => {
    devicesAPI.listDevices()
      .then((res) => setDevices(res || []))
      .catch(() => {});
  };

  createEffect(() => {
    load();
    loadDevices();
  });

  createEffect(() => {
    if (!showModal()) return;
    const handler = (e) => { if (e.key === 'Escape') setShowModal(false); };
    window.addEventListener('keydown', handler);
    return () => window.removeEventListener('keydown', handler);
  });

  const submit = (e) => {
    e.preventDefault();
    setSaving(true);
    setErr('');
    thresholdsAPI.createThreshold(form())
      .then(() => { 
        toast.show('success', '阈值已创建'); 
        setForm(empty); 
        setShowModal(false); 
        load(); 
      })
      .catch((er) => { 
        setErr(er.message || '创建失败'); 
        toast.show('error', '创建失败'); 
      })
      .finally(() => setSaving(false));
  };

  const remove = (id) => {
    if (!confirm('删除该阈值？')) return;
    thresholdsAPI.deleteThreshold(id)
      .then(() => { toast.show('success', '已删除'); load(); })
      .catch(() => toast.show('error', '删除失败'));
  };

  return (
    <div>
      <Card
        title="阈值配置列表"
        extra={
          <button 
            class="btn btn-primary" 
            onClick={() => { setForm(empty); setErr(''); setShowModal(true); }}
          >
            新增阈值
          </button>
        }
      >
        <div class="table-container" style="max-height:520px; overflow:auto;">
          <table class="table">
            <thead>
              <tr>
                <th>ID</th>
                <th>设备</th>
                <th>字段</th>
                <th>条件</th>
                <th>严重性</th>
                <th>状态</th>
                <th>操作</th>
              </tr>
            </thead>
            <tbody>
              <For each={items()}>
                {(t) => (
                  <tr>
                    <td>{t.id}</td>
                    <td>{t.device_id}</td>
                    <td>{t.field_name}</td>
                    <td>{t.operator} {t.value}</td>
                    <td>{t.severity}</td>
                    <td>
                      <span class={`badge ${t.enabled === 1 ? 'badge-running' : 'badge-stopped'}`}>
                        {t.enabled === 1 ? '启用' : '禁用'}
                      </span>
                    </td>
                    <td>
                      <button class="btn btn-danger" onClick={() => remove(t.id)}>删除</button>
                    </td>
                  </tr>
                )}
              </For>
              <For each={items().length === 0 ? [1] : []}>
                {() => (
                  <tr>
                    <td colSpan={7} style="text-align:center; padding:24px; color:var(--text-muted);">暂无阈值</td>
                  </tr>
                )}
              </For>
            </tbody>
          </table>
        </div>
      </Card>

      <Show when={showModal()}>
        <div
          class="modal-backdrop"
          style="position:fixed; inset:0; background:rgba(0,0,0,0.45); display:flex; align-items:center; justify-content:center; z-index:1000;"
          onClick={(e) => { if (e.target === e.currentTarget) setShowModal(false); }}
        >
          <div class="card" style="width:440px; max-width:92vw;">
            <div class="card-header">
              <h3 class="card-title">新增阈值</h3>
            </div>
            <form class="form" onSubmit={submit} style="padding:12px 16px 16px;">
              <div class="form-group">
                <label class="form-label">设备</label>
                <select 
                  class="form-select" 
                  value={form().device_id} 
                  onChange={(e) => setForm({ ...form(), device_id: +e.target.value })} 
                  required
                >
                  <option value="">选择设备</option>
                  {devices().map((d) => (
                    <option key={d.id} value={d.id}>{d.name || d.id}</option>
                  ))}
                </select>
              </div>
              <div class="form-group">
                <label class="form-label">字段名</label>
                <input 
                  class="form-input" 
                  value={form().field_name} 
                  onInput={(e) => setForm({ ...form(), field_name: e.target.value })} 
                  required 
                />
              </div>
              <div class="grid" style={{ gridTemplateColumns: '1fr 1fr', gap: '12px' }}>
                <div class="form-group">
                  <label class="form-label">运算符</label>
                  <select 
                    class="form-select" 
                    value={form().operator} 
                    onChange={(e) => setForm({ ...form(), operator: e.target.value })}
                  >
                    <option value=">">大于 &gt;</option>
                    <option value=">=">大于等于 &gt;=</option>
                    <option value="<">小于 &lt;</option>
                    <option value="<=">小于等于 &lt;=</option>
                    <option value="==">等于 ==</option>
                  </select>
                </div>
                <div class="form-group">
                  <label class="form-label">阈值</label>
                  <input 
                    class="form-input" 
                    type="number" 
                    value={form().value} 
                    onInput={(e) => setForm({ ...form(), value: +e.target.value })} 
                    required 
                  />
                </div>
              </div>
              <div class="form-group">
                <label class="form-label">严重程度</label>
                <select 
                  class="form-select" 
                  value={form().severity} 
                  onChange={(e) => setForm({ ...form(), severity: e.target.value })}
                >
                  <option value="info">info</option>
                  <option value="warning">warning</option>
                  <option value="error">error</option>
                  <option value="critical">critical</option>
                </select>
              </div>
              <div class="form-group">
                <label class="form-label">报警消息</label>
                <input 
                  class="form-input" 
                  value={form().message} 
                  onInput={(e) => setForm({ ...form(), message: e.target.value })} 
                />
              </div>
              <Show when={err()}>
                <div style="color:var(--accent-red); padding:4px 0;">{err()}</div>
              </Show>
              <div class="flex" style={{ gap: '8px', justifyContent: 'flex-end', marginTop: '8px' }}>
                <button 
                  type="button" 
                  class="btn" 
                  onClick={() => { setShowModal(false); setForm(empty); }} 
                  disabled={saving()}
                >
                  取消
                </button>
                <button type="submit" class="btn btn-primary" disabled={saving()}>
                  {saving() ? '创建中...' : '创建'}
                </button>
              </div>
            </form>
          </div>
        </div>
      </Show>
    </div>
  );
}
