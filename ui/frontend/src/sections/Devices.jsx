import { createSignal, createEffect, onMount, onCleanup, For, Show } from 'solid-js';
import api from '../api/services';
import { useToast } from '../components/Toast';
import Card from '../components/cards';
import ConfirmDialog from '../components/ConfirmDialog';
import CrudTable from '../components/CrudTable';
import DeviceDetailDrawer from '../components/DeviceDetailDrawer';
import Modal from '../components/Modal';
import { getErrorMessage } from '../api/errorMessages';
import { showErrorToast, withErrorToast } from '../utils/errors';
import { formatDateTime } from '../utils/time';

const defaultForm = {
  name: '',
  description: '',
  product_key: '',
  device_key: '',
  driver_id: null,
  driver_type: 'modbus_rtu_wasm',
  serial_port: '',
  baud_rate: 9600,
  data_bits: 8,
  stop_bits: 1,
  parity: 'N',
  ip_address: '',
  port_num: 502,
  device_address: '1',
  collect_interval: 1000,
  storage_interval: 300,
  timeout: 1000,
  resource_id: null,
  enabled: 1,
};

function includesWriteAccess(rw) {
  return String(rw || '').toUpperCase().includes('W');
}

function normalizeWritableItem(item) {
  if (!item) return null;
  if (typeof item === 'string') {
    const field = item.trim();
    if (!field) return null;
    return { field, label: field, unit: '', rw: 'RW' };
  }
  if (typeof item !== 'object') return null;

  const field = String(item.field || item.field_name || item.fieldName || item.name || '').trim();
  if (!field) return null;

  return {
    field,
    label: String(item.label || item.title || field).trim() || field,
    unit: String(item.unit || '').trim(),
    rw: String(item.rw || item.RW || 'RW').trim() || 'RW',
  };
}

function normalizeDriverPoint(point) {
  if (!point || typeof point !== 'object') return null;

  const field = String(point.field_name || point.FieldName || '').trim();
  if (!field) return null;

  return {
    field,
    label: String(point.label || point.Label || field).trim() || field,
    unit: String(point.unit || point.Unit || '').trim(),
    rw: String(point.rw || point.RW || '').trim(),
    value: String(point.value || point.Value || '').trim(),
  };
}

function buildWritableOptions(schemaItems, driverPoints) {
  const optionByField = new Map();

  const upsertOption = (item) => {
    if (!item || !item.field) return;
    const key = item.field.toLowerCase();
    const current = optionByField.get(key) || {};
    optionByField.set(key, {
      field: item.field,
      label: item.label || current.label || item.field,
      unit: item.unit || current.unit || '',
      rw: item.rw || current.rw || 'RW',
      value: item.value || current.value || '',
    });
  };

  (Array.isArray(schemaItems) ? schemaItems : [])
    .map(normalizeWritableItem)
    .forEach(upsertOption);

  (Array.isArray(driverPoints) ? driverPoints : [])
    .map(normalizeDriverPoint)
    .filter((point) => point && includesWriteAccess(point.rw))
    .forEach(upsertOption);

  return Array.from(optionByField.values()).sort((left, right) => left.field.localeCompare(right.field));
}

export function Devices() {
  const toast = useToast();
  const [items, setItems] = createSignal([]);
  const [resources, setResources] = createSignal([]);
  const [drivers, setDrivers] = createSignal([]);
  const [loading, setLoading] = createSignal(true);
  const [error, setError] = createSignal('');
  const [confirmState, setConfirmState] = createSignal(null);
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
  const [writeLoading, setWriteLoading] = createSignal(false);
  const [writeSubmitting, setWriteSubmitting] = createSignal(false);
  const [detailVisible, setDetailVisible] = createSignal(false);
  const [detailDevice, setDetailDevice] = createSignal(null);
  const [detailCache, setDetailCache] = createSignal([]);
  const [detailAlarms, setDetailAlarms] = createSignal([]);
  const [detailLoading, setDetailLoading] = createSignal(false);
  const showSaveError = withErrorToast(toast, '保存失败');
  let modalRoot;

  const normalizeList = (res) => {
    if (Array.isArray(res)) return res;
    if (res && Array.isArray(res.data)) return res.data;
    return [];
  };

  const formatCollectRuntime = (runtime) => {
    if (!runtime || typeof runtime !== 'object') {
      return {
        registered: false,
        failures: 0,
        nextRunAt: '',
        lastError: '',
      };
    }
    const failures = Number(runtime.consecutive_failures || 0);
    const lastErrorKind = runtime.last_error_kind ? `[${runtime.last_error_kind}] ` : '';
    return {
      registered: !!runtime.registered,
      failures,
      nextRunAt: formatDateTime(runtime.next_run_at),
      lastError: runtime.last_error ? `${lastErrorKind}${runtime.last_error}` : '',
    };
  };

  const mergeRuntimeStatuses = (list) => {
    const statuses = normalizeList(list);
    if (!statuses.length) return;

    const statusMap = new Map();
    statuses.forEach((status) => {
      if (!status || typeof status !== 'object') return;
      const id = Number(status.device_id);
      if (!id) return;
      statusMap.set(id, status);
    });
    if (!statusMap.size) return;

    setItems((prev) =>
      prev.map((item) => {
        if (!item) return item;
        const status = statusMap.get(Number(item.id));
        if (!status) return item;
        return { ...item, collect_runtime: status };
      })
    );
  };

  const refreshRuntimeStatuses = () => {
    api.devices.listDeviceRuntimeStatuses()
      .then(mergeRuntimeStatuses)
      .catch(() => {});
  };

  const load = () => {
    setLoading(true);
    Promise.allSettled([
      api.devices.listDevices(),
      api.devices.listResources(),
      api.devices.listDrivers(),
    ])
      .then(([devRes, resRes, drvRes]) => {
        if (devRes.status === 'fulfilled') {
          setItems(normalizeList(devRes.value));
        }
        if (resRes.status === 'fulfilled') {
          setResources(normalizeList(resRes.value));
        }
        if (drvRes.status === 'fulfilled') {
          const list = normalizeList(drvRes.value);
          setDrivers(list.filter((d) => Number(d.enabled ?? 1) === 1));
        }

        const hasError = [devRes, resRes, drvRes].some((r) => r.status === 'rejected');
        setError(hasError ? '加载设备、资源或驱动失败' : '');
      })
      .finally(() => setLoading(false));
  };

  onMount(() => {
    load();
    const timer = setInterval(refreshRuntimeStatuses, 5000);
    onCleanup(() => clearInterval(timer));
  });

  // ESC 关闭弹窗
  createEffect(() => {
    if (!showModal()) return;
    const handler = (e) => { if (e.key === 'Escape') setShowModal(false); };
    window.addEventListener('keydown', handler);
    return () => window.removeEventListener('keydown', handler);
  });

  // ESC 关闭详情抽屉
  createEffect(() => {
    if (!detailVisible()) return;
    const handler = (e) => { if (e.key === 'Escape') closeDetail(); };
    window.addEventListener('keydown', handler);
    return () => window.removeEventListener('keydown', handler);
  });

  const filtered = () => {
    const q = search().trim().toLowerCase();
    if (!q) return items();
    return items().filter((d) =>
      [d.name, d.device_address, d.driver_type, d.driver_name, d.collect_runtime?.last_error]
        .filter(Boolean)
        .some((v) => v.toLowerCase().includes(q))
    );
  };

  const submit = (e) => {
    e.preventDefault();
    setSubmitting(true);
    const request = editing()
      ? api.devices.updateDevice(editing(), form())
      : api.devices.createDevice(form());
    request.then(() => {
      toast.show('success', editing() ? '设备已更新' : '设备已创建');
      setForm(defaultForm);
      setEditing(null);
      setShowModal(false);
      load();
    })
    .catch(showSaveError)
    .finally(() => setSubmitting(false));
  };

  const toggle = (id) => {
    api.devices.toggleDevice(id)
      .then(load)
      .catch((err) => showErrorToast(toast, err, '切换失败'));
  };

  const remove = (id) => {
    setConfirmState({
      title: '删除设备',
      message: '确定删除该设备？',
      confirmText: '删除',
      variant: 'danger',
      onConfirm: () => {
        setConfirmState(null);
        api.devices.deleteDevice(id)
          .then(() => { toast.show('success', '已删除'); load(); })
          .catch((err) => showErrorToast(toast, err, '删除失败'));
      },
    });
  };

  const closeWrite = () => {
    setShowWriteModal(false);
    setWriteTarget(null);
    setWriteMeta([]);
    setWriteForm({ field: '', value: '' });
    setWriteError('');
    setWriteLoading(false);
    setWriteSubmitting(false);
  };

  const openWrite = async (device) => {
    setWriteTarget(device);
    setWriteMeta([]);
    setWriteForm({ field: '', value: '' });
    setWriteError('');
    setWriteLoading(true);
    setShowWriteModal(true);

    const [schemaResult, liveResult] = await Promise.allSettled([
      api.devices.listWritables(device.id),
      api.devices.executeDeviceFunction(device.id, { function: 'collect', params: {} }),
    ]);

    const schemaItems = schemaResult.status === 'fulfilled'
      ? (Array.isArray(schemaResult.value) ? schemaResult.value : schemaResult.value?.writable || [])
      : [];
    const livePoints =
      liveResult.status === 'fulfilled' && Array.isArray(liveResult.value?.points)
        ? liveResult.value.points
        : [];

    const options = buildWritableOptions(schemaItems, livePoints);
    setWriteMeta(options);
    setWriteForm((prev) => ({
      field: options[0]?.field || prev.field || '',
      value: '',
    }));

    if (!options.length && liveResult.status === 'rejected') {
      setWriteError(getErrorMessage(liveResult.reason, '未获取到可写测点'));
    }

    setWriteLoading(false);
  };

  const submitWrite = (e) => {
    e.preventDefault();
    const target = writeTarget();
    const field = writeForm().field.trim();
    const value = writeForm().value.trim();

    setWriteError('');
    if (!target) {
      setWriteError('未选择设备');
      return;
    }
    if (!field) {
      setWriteError('请输入字段名');
      return;
    }
    if (!value) {
      setWriteError('请输入写入值');
      return;
    }

    setWriteSubmitting(true);
    api.devices.executeDeviceFunction(target.id, {
      function: 'write',
      params: {
        field_name: field,
        value,
      },
    })
      .then(() => {
        toast.show('success', '写入成功');
        closeWrite();
      })
      .catch((err) => setWriteError(getErrorMessage(err, '写入失败')))
      .finally(() => setWriteSubmitting(false));
  };

  const openDetail = async (device) => {
    setDetailDevice(device);
    setDetailVisible(true);
    setDetailLoading(true);
    try {
      const [cacheRes, alarmsRes] = await Promise.all([
        api.data.getDataCacheByDevice(device.id),
        api.alarms.listAlarms(),
      ]);
      const cacheVal = Array.isArray(cacheRes) ? cacheRes : cacheRes?.data || [];
      const allAlarms = Array.isArray(alarmsRes) ? alarmsRes : alarmsRes?.data || [];
      setDetailCache(cacheVal);
      setDetailAlarms(allAlarms.filter((a) => String(a.device_id) === String(device.id)));
    } finally {
      setDetailLoading(false);
    }
  };

  const closeDetail = () => {
    setDetailVisible(false);
    setDetailDevice(null);
    setDetailCache([]);
    setDetailAlarms([]);
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
      driver_id: item.driver_id,
      driver_type: item.driver_type || 'modbus_rtu_wasm',
      serial_port: item.serial_port || '',
      baud_rate: item.baud_rate || 9600,
      data_bits: item.data_bits || 8,
      stop_bits: item.stop_bits || 1,
      parity: item.parity || 'N',
      ip_address: item.ip_address || '',
      port_num: item.port_num || 502,
      device_address: item.device_address || '1',
      collect_interval: item.collect_interval || 1000,
      storage_interval: item.storage_interval || 300,
      timeout: item.timeout || 1000,
      resource_id: item.resource_id,
      enabled: item.enabled,
    });
    setShowModal(true);
  };

  const filteredResources = () => {
    if (!resources().length) return [];
    return resources();
  };

  return (
    <div>
      <Show when={confirmState()}>
        {(dialog) => (
          <ConfirmDialog
            title={dialog().title}
            message={dialog().message}
            confirmText={dialog().confirmText}
            variant={dialog().variant}
            onCancel={() => setConfirmState(null)}
            onConfirm={dialog().onConfirm}
          />
        )}
      </Show>
      <Card
        title="设备列表"
        extra={
          <div class="toolbar-actions">
            <input
              class="form-input"
              style="max-width:240px;"
              placeholder="搜索设备/地址/驱动"
              value={search()}
              onInput={(e) => setSearch(e.target.value)}
            />
            <button class="btn btn-ghost btn-sm" onClick={load}>刷新</button>
            <button class="btn btn-primary btn-sm" onClick={openCreate}>新增设备</button>
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
          <CrudTable
            style="max-height:520px; overflow:auto;"
            loading={loading()}
            items={filtered()}
            emptyText="暂无设备"
            columns={[
              { key: 'id', title: 'ID' },
              { key: 'name', title: '名称' },
              { key: 'driver_type', title: '驱动类型' },
              {
                key: 'driver_name',
                title: '驱动',
                render: (d) => d.driver_name || (d.driver_id ? `驱动 #${d.driver_id}` : '-'),
              },
              {
                key: 'resource',
                title: '资源',
                render: (d) =>
                  d.resource_name ? (
                    <div>
                      <div>{d.resource_name}</div>
                      <div class="text-muted text-xs">{d.resource_path}</div>
                    </div>
                  ) : (
                    <span style="color:var(--text-muted);">-</span>
                  ),
              },
              { key: 'collect_interval', title: '周期(ms)' },
              {
                key: 'collect_runtime',
                title: '采集运行时',
                render: (d) => {
                  const runtime = formatCollectRuntime(d.collect_runtime);
                  return (
                    <div>
                      <div>
                        <span class={`badge ${runtime.registered ? 'badge-running' : 'badge-stopped'}`}>
                          {runtime.registered ? '已注册' : '未注册'}
                        </span>
                        <Show when={runtime.failures > 0}>
                          <span style="margin-left:8px; color:var(--accent-red);">
                            连续失败 {runtime.failures}
                          </span>
                        </Show>
                      </div>
                      <Show when={runtime.nextRunAt}>
                        <div class="text-muted text-xs">下次采集: {runtime.nextRunAt}</div>
                      </Show>
                      <Show when={runtime.lastError}>
                        <div class="text-muted text-xs" style="max-width:360px; overflow:hidden; text-overflow:ellipsis; white-space:nowrap;">
                          最后错误: {runtime.lastError}
                        </div>
                      </Show>
                    </div>
                  );
                },
              },
              {
                key: 'enabled',
                title: '状态',
                render: (d) => (
                  <span class={`badge ${d.enabled === 1 ? 'badge-running' : 'badge-stopped'}`}>
                    {d.enabled === 1 ? '启用' : '禁用'}
                  </span>
                ),
              },
            ]}
            renderActions={(d) => (
              <div class="table-actions">
                <button class="btn btn-outline-primary" onClick={() => openDetail(d)}>详情</button>
                <button class="btn btn-outline-primary" onClick={() => edit(d)}>编辑</button>
                <button class="btn btn-soft-primary" onClick={() => toggle(d.id)}>
                  {d.enabled === 1 ? '禁用' : '启用'}
                </button>
                <button class="btn btn-outline-danger" onClick={() => remove(d.id)}>删除</button>
                <button class="btn btn-soft-primary" onClick={() => openWrite(d)}>写入</button>
              </div>
            )}
          />
        )}
      </Card>

      <Show when={showModal()}>
        <Modal
          title={editing() ? '编辑设备' : '新增设备'}
          onClose={() => { setShowModal(false); setEditing(null); setForm(defaultForm); }}
          contentStyle="width:720px; max-width:100%;"
          backdropStyle="overflow:auto; padding:24px;"
        >
            <form class="form" onSubmit={submit} style="padding:0 4px;">
              <div class="grid" style="grid-template-columns: repeat(2, 1fr); gap:12px;">
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
                <div class="form-group">
                  <label class="form-label">ProductKey</label>
                  <input 
                    class="form-input" 
                    value={form().product_key} 
                    onInput={(e) => setForm({ ...form(), product_key: e.target.value })} 
                    placeholder="子设备 productKey"
                  />
                </div>
                <div class="form-group">
                  <label class="form-label">DeviceKey</label>
                  <input 
                    class="form-input" 
                    value={form().device_key} 
                    onInput={(e) => setForm({ ...form(), device_key: e.target.value })} 
                    placeholder="子设备 deviceKey"
                  />
                </div>
                <div class="form-group">
                  <label class="form-label">选择驱动 <span style="color:var(--accent-red);">*</span></label>
                  <select
                    class="form-select"
                    value={form().driver_id ?? ''}
                    onChange={(e) => setForm({ ...form(), driver_id: e.target.value ? +e.target.value : null })}
                    required
                  >
                    <option value="">请选择驱动</option>
                    <Show when={!drivers().length}>
                      <option value="" disabled>暂无可用驱动</option>
                    </Show>
                    <For each={drivers()}>
                      {(d) => (
                        <option value={d.id}>
                          {d.name} (v{d.version || '1.0'})
                        </option>
                      )}
                    </For>
                  </select>
                </div>
                <div class="form-group">
                  <label class="form-label">绑定资源</label>
                  <select
                    class="form-select"
                    value={form().resource_id ?? ''}
                    onChange={(e) => setForm({ ...form(), resource_id: e.target.value ? +e.target.value : null })}
                  >
                    <option value="">请选择资源</option>
                    <Show when={!filteredResources().length}>
                      <option value="" disabled>暂无可用资源</option>
                    </Show>
                    <For each={filteredResources()}>
                      {(r) => (
                        <option value={r.id}>
                          {r.name} ({r.type}) {r.path ? `- ${r.path}` : ''}
                        </option>
                      )}
                    </For>
                  </select>
                </div>
              </div>
                <div class="form-group">
                  <label class="form-label">驱动类型</label>
                  <select 
                    class="form-select" 
                    value={form().driver_type} 
                    onChange={(e) => setForm({ ...form(), driver_type: e.target.value, resource_id: null })}
                  >
                    <option value="modbus_rtu_wasm">ModbusRtu / Wasm</option>
                    <option value="modbus_tcp_wasm">ModbusTcp / Wasm</option>
                    <option value="modbus_rtu_excel">ModbusRtu / Excel</option>
                    <option value="modbus_tcp_excel">ModbusTcp / Excel</option>
                  </select>
                </div>

              <Show when={form().driver_type?.startsWith('modbus_rtu')}>
                <div class="grid" style="grid-template-columns: repeat(3, 1fr); gap:12px;">
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
                  <div class="form-group">
                    <label class="form-label">设备地址</label>
                    <input 
                      class="form-input" 
                      value={form().device_address} 
                      onInput={(e) => setForm({ ...form(), device_address: e.target.value })} 
                      required 
                    />
                  </div>
                </div>
              </Show>

              <Show when={form().driver_type?.startsWith('modbus_tcp')}>
                <div class="grid" style="grid-template-columns: repeat(1, 1fr); gap:12px;">
                  <div class="form-group">
                    <label class="form-label">设备地址</label>
                    <input 
                      class="form-input" 
                      value={form().device_address} 
                      onInput={(e) => setForm({ ...form(), device_address: e.target.value })} 
                      required 
                    />
                  </div>
                </div>
              </Show>

              <div class="grid" style="grid-template-columns: repeat(3, 1fr); gap:12px;">
                <div class="form-group">
                  <label class="form-label">采集周期(ms)</label>
                  <input 
                    class="form-input" 
                    type="number" 
                    value={form().collect_interval} 
                    onInput={(e) => setForm({ ...form(), collect_interval: +e.target.value })} 
                    required 
                  />
                </div>
                <div class="form-group">
                  <label class="form-label">存储周期(s)</label>
                  <input 
                    class="form-input" 
                    type="number" 
                    value={form().storage_interval} 
                    onInput={(e) => setForm({ ...form(), storage_interval: +e.target.value })} 
                    required 
                  />
                </div>
                <div class="form-group">
                  <label class="form-label">超时(ms)</label>
                  <input 
                    class="form-input" 
                    type="number" 
                    value={form().timeout} 
                    onInput={(e) => setForm({ ...form(), timeout: +e.target.value })} 
                    required 
                  />
                </div>
              </div>

              <div class="form-group" style="margin-top:12px;">
                <label class="form-label">状态</label>
                <select 
                  class="form-select" 
                  value={form().enabled} 
                  onChange={(e) => setForm({ ...form(), enabled: +e.target.value })}
                >
                  <option value={1}>启用</option>
                  <option value={0}>禁用</option>
                </select>
              </div>
              
              <div class="modal-actions" style={{ marginTop: '16px' }}>
                <button
                  type="button"
                  class="btn btn-outline-primary btn-sm"
                  onClick={() => { setShowModal(false); setEditing(null); setForm(defaultForm); }}
                  disabled={submitting()}
                >
                  取消
                </button>
                <button type="submit" class="btn btn-primary btn-sm" disabled={submitting()}>
                  {submitting() ? '保存中...' : (editing() ? '保存' : '创建')}
                </button>
              </div>
            </form>
        </Modal>
      </Show>

      <Show when={showWriteModal()}>
        <Modal
          title={`写入数据 - ${writeTarget()?.name}`}
          onClose={closeWrite}
          backdropStyle="z-index:1001;"
          contentStyle="width:400px; max-width:90vw;"
        >
            <form class="form" onSubmit={submitWrite} style="padding:16px;">
              <div class="text-muted text-xs" style="margin-bottom:12px;">
                单次调用只写一个测点。优先使用驱动实时返回的可写字段。
              </div>
              <Show when={writeLoading()}>
                <div class="loading-state" style="padding:12px 0;">
                  <div class="loading-spinner"></div>
                  <div>正在加载可写测点...</div>
                </div>
              </Show>
              <div class="form-group">
                <label class="form-label">字段</label>
                <Show
                  when={writeMeta().length > 0}
                  fallback={
                    <input
                      class="form-input"
                      value={writeForm().field}
                      onInput={(e) => setWriteForm({ ...writeForm(), field: e.target.value })}
                      placeholder="例如 TEMSET"
                      disabled={writeLoading() || writeSubmitting()}
                    />
                  }
                >
                  <select
                    class="form-select"
                    value={writeForm().field}
                    onChange={(e) => setWriteForm({ ...writeForm(), field: e.target.value })}
                    disabled={writeLoading() || writeSubmitting()}
                  >
                    <For each={writeMeta()}>
                      {(w) => (
                        <option value={w.field}>
                          {w.label || w.field}{w.unit ? ` (${w.unit})` : ''}
                        </option>
                      )}
                    </For>
                  </select>
                </Show>
                <Show when={!writeLoading() && writeMeta().length === 0}>
                  <div class="text-muted text-xs" style="margin-top:6px;">
                    当前未发现驱动声明的可写点，可手工输入字段名后单点写入。
                  </div>
                </Show>
              </div>
              <div class="form-group">
                <label class="form-label">值</label>
                <input 
                  class="form-input" 
                  value={writeForm().value} 
                  onInput={(e) => setWriteForm({ ...writeForm(), value: e.target.value })} 
                  placeholder="请输入要写入的值"
                  disabled={writeLoading() || writeSubmitting()}
                  required 
                />
              </div>
              <Show when={writeError()}>
                <div style="color:var(--accent-red); padding:4px 0;">{writeError()}</div>
              </Show>
              <div class="modal-actions modal-actions-fill" style={{ marginTop: '12px' }}>
                <button type="button" class="btn btn-outline-primary btn-sm" onClick={closeWrite} disabled={writeSubmitting()}>取消</button>
                <button type="submit" class="btn btn-primary btn-sm" disabled={writeLoading() || writeSubmitting()}>
                  {writeSubmitting() ? '写入中...' : '写入'}
                </button>
              </div>
            </form>
        </Modal>
      </Show>

      <DeviceDetailDrawer
        visible={detailVisible()}
        device={detailDevice}
        cache={detailCache}
        alarms={detailAlarms}
        loading={detailLoading}
        onWrite={openWrite}
        onClose={closeDetail}
      />
    </div>
  );
}
