import { createSignal, createEffect, For, Show } from 'solid-js';
import { del, getJSON, postJSON, putJSON } from '../api';
import { useToast } from '../components/Toast';
import Card from '../components/cards';

const defaultForm = {
  name: '',
  description: '',
  product_key: '',
  device_key: '',
  driver_type: 'modbus_rtu',
  serial_port: '/dev/ttyS0',
  baud_rate: 9600,
  data_bits: 8,
  stop_bits: 1,
  parity: 'N',
  ip_address: '',
  port_num: 502,
  device_address: '1',
  collect_interval: 1000,
  timeout: 1000,
  resource_id: null,
  enabled: 1,
};

export function Devices() {
  const toast = useToast();
  const [items, setItems] = createSignal([]);
  const [resources, setResources] = createSignal([]);
  const [loading, setLoading] = createSignal(true);
  const [error, setError] = createSignal('');
  const [search, setSearch] = createSignal('');
  const [form, setForm] = createSignal(defaultForm);
  const [editing, setEditing] = createSignal(null);
  const [showModal, setShowModal] = createSignal(false);
  const [submitting, setSubmitting] = createSignal(false);
  const [showWriteModal, setShowWriteModal] = createSignal(false);
  const [writeMeta, setWriteMeta] = createSignal([]);
  const [writeForm, setWriteForm] = createSignal({ field: '', value: '' });
  const [writeError, setWriteError] = createSignal('');
  const [writeTarget, setWriteTarget] = createSignal(null);
  let modalRoot;

  const load = () => {
    setLoading(true);
    Promise.all([getJSON('/api/devices'), getJSON('/api/resources')])
      .then(([devRes, resRes]) => {
        setItems(devRes.data || devRes);
        setResources(resRes.data || resRes);
        setError('');
      })
      .catch(() => setError('加载设备或资源失败'))
      .finally(() => setLoading(false));
  };

  createEffect(() => {
    load();
  });

  // ESC 关闭弹窗
  createEffect(() => {
    if (!showModal()) return;
    const handler = (e) => { if (e.key === 'Escape') setShowModal(false); };
    window.addEventListener('keydown', handler);
    return () => window.removeEventListener('keydown', handler);
  });

  const filtered = () => {
    const q = search().trim().toLowerCase();
    if (!q) return items();
    return items().filter((d) =>
      [d.name, d.device_address, d.driver_type, d.driver_name]
        .filter(Boolean)
        .some((v) => v.toLowerCase().includes(q))
    );
  };

  const submit = (e) => {
    e.preventDefault();
    setSubmitting(true);
    const api = editing() ? putJSON(`/api/devices/${editing()}`, form()) : postJSON('/api/devices', form());
    api.then(() => {
      toast.show('success', editing() ? '设备已更新' : '设备已创建');
      setForm(defaultForm);
      setEditing(null);
      setShowModal(false);
      load();
    })
    .catch(() => toast.show('error', '操作失败'))
    .finally(() => setSubmitting(false));
  };

  const toggle = (id) => {
    postJSON(`/api/devices/${id}/toggle`, {})
      .then(load)
      .catch(() => toast.show('error', '切换失败'));
  };

  const remove = (id) => {
    if (!confirm('确定删除该设备？')) return;
    if (!confirm('删除后将无法恢复，继续吗？')) return;
    del(`/api/devices/${id}`)
      .then(() => {
        toast.show('success', '已删除');
        load();
      })
      .catch(() => toast.show('error', '删除失败'));
  };

  const openWrite = (device) => {
    setWriteError('');
    setWriteMeta([]);
    setWriteForm({ field: '', value: '' });
    setWriteTarget(device.id);
    setShowWriteModal(true);
    getJSON(`/api/devices/${device.id}/writables`)
      .then((meta) => {
        const list = meta?.data || meta || [];
        setWriteMeta(list);
        if (list.length) setWriteForm({ field: list[0].field || '', value: '' });
      })
      .catch(() => setWriteError('加载可写寄存器失败'));
  };

  const submitWrite = (e) => {
    e.preventDefault();
    setWriteError('');
    const field = writeForm().field;
    const value = writeForm().value;
    if (!field) {
      setWriteError('请选择字段');
      return;
    }
    postJSON(`/api/devices/${writeTarget()}/execute`, {
      function: 'handle',
      params: { func_name: 'write', field, value },
    })
      .then(() => {
        toast.show('success', '写入成功');
        setShowWriteModal(false);
      })
      .catch((err) => setWriteError(err.message || '写入失败'));
  };

  const openCreate = () => {
    setForm(defaultForm);
    setEditing(null);
    setShowModal(true);
  };

  const edit = (item) => {
    setEditing(item.id);
    setForm({
      name: item.name,
      description: item.description || '',
      product_key: item.product_key || '',
      device_key: item.device_key || '',
      driver_type: item.driver_type,
      serial_port: item.serial_port || '/dev/ttyS0',
      baud_rate: item.baud_rate || 9600,
      data_bits: item.data_bits || 8,
      stop_bits: item.stop_bits || 1,
      parity: item.parity || 'N',
      ip_address: item.ip_address || '',
      port_num: item.port_num || 502,
      device_address: item.device_address || '1',
      collect_interval: item.collect_interval || 1000,
      timeout: item.timeout || 1000,
      resource_id: item.resource_id,
      enabled: item.enabled,
    });
    setShowModal(true);
  };

  const filteredResources = () => {
    if (!resources().length) return [];
    if (form().driver_type === 'modbus_rtu') return resources().filter((r) => r.type === 'serial');
    if (form().driver_type === 'modbus_tcp') return resources().filter((r) => r.type === 'net');
    return resources();
  };

  return (
    <div>
      <Card
        title="设备列表"
        extra={
          <div class="flex gap-3">
            <input
              class="form-input"
              style="max-width:240px;"
              placeholder="搜索设备/地址/驱动"
              value={search()}
              onInput={(e) => setSearch(e.target.value)}
            />
            <button class="btn" onClick={load}>刷新</button>
            <button class="btn btn-primary" onClick={openCreate}>新增设备</button>
          </div>
        }
      >
        {loading() ? (
          <div class="text-center" style="padding:48px; color:var(--text-muted);">
            <div class="loading-spinner" style="margin:0 auto 16px;"></div>
            <div>加载中...</div>
          </div>
        ) : error() ? (
          <div style="color:var(--accent-red); padding:16px 0;">{error()}</div>
        ) : (
          <div class="table-container" style="max-height:520px; overflow:auto;">
            <table class="table">
              <thead>
                <tr>
                  <th>ID</th>
                  <th>名称</th>
                  <th>产品/设备Key</th>
                  <th>驱动类型</th>
                  <th>驱动</th>
                  <th>资源</th>
                  <th>周期(ms)</th>
                  <th>状态</th>
                  <th>操作</th>
                </tr>
              </thead>
              <tbody>
                <For each={filtered()}>
                  {(d) => (
                    <tr>
                      <td>{d.id}</td>
                      <td>{d.name}</td>
                      <td style="min-width:140px;">
                        <div class="text-sm">{d.product_key || '-'}</div>
                        <div class="text-muted text-xs">{d.device_key || ''}</div>
                      </td>
                      <td>{d.driver_type}</td>
                      <td>{d.driver_name || (d.driver_id ? `驱动 #${d.driver_id}` : '-')}</td>
                      <td>
                        {d.resource_name ? (
                          <div>
                            <div>{d.resource_name}</div>
                            <div class="text-muted text-xs">{d.resource_path}</div>
                          </div>
                        ) : (
                          <span class="text-muted">未绑定</span>
                        )}
                      </td>
                      <td>{d.collect_interval}</td>
                      <td>
                        <span class={`badge ${d.enabled === 1 ? 'badge-running' : 'badge-stopped'}`}>
                          {d.enabled === 1 ? '采集中' : '停止'}
                        </span>
                      </td>
                      <td class="flex" style="gap:8px;">
                        <button class="btn" onClick={() => edit(d)}>编辑</button>
                        <button class="btn" onClick={() => openWrite(d)}>写</button>
                        <button 
                          class={`btn ${d.enabled === 1 ? 'btn-danger' : 'btn-success'}`} 
                          onClick={() => toggle(d.id)}
                        >
                          {d.enabled === 1 ? '停止' : '启动'}
                        </button>
                        <button class="btn btn-danger" onClick={() => remove(d.id)}>删除</button>
                      </td>
                    </tr>
                  )}
                </For>
                <Show when={filtered().length === 0}>
                  <tr>
                    <td colSpan={9} style="text-align:center; padding:24px; color:var(--text-muted);">暂无设备</td>
                  </tr>
                </Show>
              </tbody>
            </table>
          </div>
        )}
      </Card>

      <Show when={showModal()}>
        <div
          ref={modalRoot}
          class="modal-backdrop"
          style="position:fixed; inset:0; background:rgba(0,0,0,0.45); display:flex; align-items:center; justify-content:center; z-index:1000; overflow:auto; padding:24px;"
          onClick={(e) => { if (e.target === e.currentTarget) setShowModal(false); }}
        >
          <div class="card" style="width:640px; max-width:100%;">
            <div class="card-header">
              <h3 class="card-title">{editing() ? '编辑设备' : '新增设备'}</h3>
              <button class="btn btn-ghost" onClick={() => { setShowModal(false); setEditing(null); setForm(defaultForm); }} style="padding:4px 8px;">✕</button>
            </div>
            <form class="form" onSubmit={submit} style="padding:0 4px;">
              <div class="grid" style="grid-template-columns: 1fr 1fr; gap:12px;">
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
                  <label class="form-label">描述</label>
                  <input 
                    class="form-input" 
                    value={form().description} 
                    onInput={(e) => setForm({ ...form(), description: e.target.value })} 
                  />
                </div>
              </div>
              <div class="grid" style="grid-template-columns: repeat(2, 1fr); gap:12px;">
                <div class="form-group">
                  <label class="form-label">ProductKey</label>
                  <input 
                    class="form-input" 
                    value={form().product_key} 
                    onInput={(e) => setForm({ ...form(), product_key: e.target.value })} 
                  />
                </div>
                <div class="form-group">
                  <label class="form-label">DeviceKey</label>
                  <input 
                    class="form-input" 
                    value={form().device_key} 
                    onInput={(e) => setForm({ ...form(), device_key: e.target.value })} 
                  />
                </div>
              </div>
              <div class="form-group">
                <label class="form-label">驱动类型</label>
                <select 
                  class="form-select" 
                  value={form().driver_type} 
                  onChange={(e) => setForm({ ...form(), driver_type: e.target.value, resource_id: null })}
                >
                  <option value="modbus_rtu">Modbus RTU (串口)</option>
                  <option value="modbus_tcp">Modbus TCP</option>
                </select>
              </div>

              <Show when={form().driver_type === 'modbus_rtu'}>
                <div class="grid" style="grid-template-columns: repeat(3, 1fr); gap:12px;">
                  <div class="form-group">
                    <label class="form-label">串口</label>
                    <input 
                      class="form-input" 
                      value={form().serial_port} 
                      onInput={(e) => setForm({ ...form(), serial_port: e.target.value })} 
                      required 
                    />
                  </div>
                  <div class="form-group">
                    <label class="form-label">波特率</label>
                    <input 
                      class="form-input" 
                      type="number" 
                      value={form().baud_rate} 
                      onInput={(e) => setForm({ ...form(), baud_rate: +e.target.value })} 
                      required 
                    />
                  </div>
                  <div class="form-group">
                    <label class="form-label">校验位</label>
                    <select 
                      class="form-select" 
                      value={form().parity} 
                      onChange={(e) => setForm({ ...form(), parity: e.target.value })}
                    >
                      <option value="N">None</option>
                      <option value="E">Even</option>
                      <option value="O">Odd</option>
                    </select>
                  </div>
                  <div class="form-group">
                    <label class="form-label">数据位</label>
                    <input 
                      class="form-input" 
                      type="number" 
                      value={form().data_bits} 
                      onInput={(e) => setForm({ ...form(), data_bits: +e.target.value })} 
                      required 
                    />
                  </div>
                  <div class="form-group">
                    <label class="form-label">停止位</label>
                    <input 
                      class="form-input" 
                      type="number" 
                      value={form().stop_bits} 
                      onInput={(e) => setForm({ ...form(), stop_bits: +e.target.value })} 
                      required 
                    />
                  </div>
                </div>
              </Show>

              <Show when={form().driver_type === 'modbus_tcp'}>
                <div class="grid" style="grid-template-columns: 1fr 1fr; gap:12px;">
                  <div class="form-group">
                    <label class="form-label">IP 地址</label>
                    <input 
                      class="form-input" 
                      value={form().ip_address} 
                      onInput={(e) => setForm({ ...form(), ip_address: e.target.value })} 
                      required 
                    />
                  </div>
                  <div class="form-group">
                    <label class="form-label">端口</label>
                    <input 
                      class="form-input" 
                      type="number" 
                      value={form().port_num} 
                      onInput={(e) => setForm({ ...form(), port_num: +e.target.value })} 
                      required 
                    />
                  </div>
                </div>
              </Show>

              <div class="grid" style="grid-template-columns: repeat(3, 1fr); gap:12px;">
                <div class="form-group">
                  <label class="form-label">设备地址</label>
                  <input 
                    class="form-input" 
                    value={form().device_address} 
                    onInput={(e) => setForm({ ...form(), device_address: e.target.value })} 
                    required 
                  />
                </div>
                <div class="form-group">
                  <label class="form-label">采集周期 (ms)</label>
                  <input 
                    class="form-input" 
                    type="number" 
                    value={form().collect_interval} 
                    onInput={(e) => setForm({ ...form(), collect_interval: +e.target.value })} 
                    required 
                  />
                </div>
                <div class="form-group">
                  <label class="form-label">超时 (ms)</label>
                  <input 
                    class="form-input" 
                    type="number" 
                    value={form().timeout} 
                    onInput={(e) => setForm({ ...form(), timeout: +e.target.value })} 
                  />
                </div>
              </div>

              <div class="form-group">
                <label class="form-label">绑定资源</label>
                <select 
                  class="form-select" 
                  value={form().resource_id ?? ''} 
                  onChange={(e) => setForm({ ...form(), resource_id: e.target.value ? Number(e.target.value) : null })}
                >
                  <option value="">不绑定</option>
                  {filteredResources().map((r) => (
                    <option key={r.id} value={r.id}>{`${r.name} (${r.path})`}</option>
                  ))}
                </select>
                <div class="text-muted text-xs" style="margin-top:4px;">同一资源串行访问，建议按驱动类型匹配。</div>
              </div>

              <div class="flex" style={{ gap: '8px', justifyContent: 'flex-end', marginTop: '16px' }}>
                <button 
                  type="button" 
                  class="btn" 
                  onClick={() => { setShowModal(false); setEditing(null); setForm(defaultForm); }} 
                  disabled={submitting()}
                >
                  取消
                </button>
                <button type="submit" class="btn btn-primary" disabled={submitting()}>
                  {submitting() ? '保存中...' : (editing() ? '保存' : '创建')}
                </button>
              </div>
            </form>
          </div>
        </div>
      </Show>

      <Show when={showWriteModal()}>
        <div
          class="modal-backdrop"
          style="position:fixed; inset:0; background:rgba(0,0,0,0.45); display:flex; align-items:center; justify-content:center; z-index:1000; overflow:auto; padding:24px;"
          onClick={(e) => { if (e.target === e.currentTarget) setShowWriteModal(false); }}
        >
          <div class="card" style="width:420px; max-width:90vw;">
            <div class="card-header">
              <h3 class="card-title">写寄存器</h3>
              <button class="btn btn-ghost" onClick={() => setShowWriteModal(false)} style="padding:4px 8px;">✕</button>
            </div>
            <form class="form" onSubmit={submitWrite} style="padding:12px 16px 16px;">
              <div class="form-group">
                <label class="form-label">字段</label>
                <select
                  class="form-select"
                  value={writeForm().field}
                  onChange={(e) => setWriteForm({ ...writeForm(), field: e.target.value })}
                  required
                >
                  <option value="">选择字段</option>
                  <For each={writeMeta()}>
                    {(w) => <option value={w.field || w.name}>{w.label || w.field || w.name}</option>}
                  </For>
                </select>
                <Show when={writeMeta().length === 0}>
                  <div style="color:var(--text-muted); font-size:12px; margin-top:4px;">驱动未提供可写元数据</div>
                </Show>
              </div>
              <div class="form-group">
                <label class="form-label">值</label>
                <input
                  class="form-input"
                  value={writeForm().value}
                  onInput={(e) => setWriteForm({ ...writeForm(), value: e.target.value })}
                  required
                />
              </div>
              <Show when={writeError()}>
                <div style="color:var(--accent-red); padding:4px 0;">{writeError()}</div>
              </Show>
              <div class="flex" style="gap:8px; justify-content:flex-end; margin-top:8px;">
                <button type="button" class="btn" onClick={() => setShowWriteModal(false)}>取消</button>
                <button type="submit" class="btn btn-primary">写入</button>
              </div>
            </form>
          </div>
        </div>
      </Show>
    </div>
  );
}
