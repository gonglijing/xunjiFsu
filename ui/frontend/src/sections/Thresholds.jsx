import { createSignal, createEffect, onMount, For, Show, createMemo } from 'solid-js';
import api from '../api/services';
import { useToast } from '../components/Toast';
import Card from '../components/cards';
import { getErrorMessage } from '../api/errorMessages';
import { showErrorToast } from '../utils/errors';
import { usePageLoader } from '../utils/pageLoader';
import LoadErrorHint from '../components/LoadErrorHint';

const DEFAULT_REPEAT_INTERVAL_MINUTES = 1;
const empty = {
  device_id: '',
  field_name: '',
  operator: '>',
  value: 0,
  severity: 'warning',
  message: '',
  enabled: 1,
  shielded: 0,
};

export function Thresholds() {
  const toast = useToast();
  const [items, setItems] = createSignal([]);
  const [devices, setDevices] = createSignal([]);
  const [repeatIntervalMinutes, setRepeatIntervalMinutes] = createSignal(DEFAULT_REPEAT_INTERVAL_MINUTES);
  const [savingRepeatInterval, setSavingRepeatInterval] = createSignal(false);
  const {
    loading,
    error: loadError,
    setError: setLoadError,
    run: runThresholdsLoad,
  } = usePageLoader(async () => {
    const [thresholds, devicesList, repeatCfg] = await Promise.all([
      api.thresholds.listThresholds(),
      api.devices.listDevices(),
      api.thresholds.getAlarmRepeatInterval(),
    ]);
    setItems(thresholds || []);
    setDevices(devicesList || []);
    const seconds = Number(repeatCfg?.seconds);
    setRepeatIntervalMinutes(Number.isFinite(seconds) && seconds > 0 ? Math.max(1, Math.ceil(seconds / 60)) : DEFAULT_REPEAT_INTERVAL_MINUTES);
  }, {
    errorMessage: '加载阈值失败',
    onError: (err) => showErrorToast(toast, err, '加载阈值失败'),
  });
  const [form, setForm] = createSignal(empty);
  const [showModal, setShowModal] = createSignal(false);
  const [saving, setSaving] = createSignal(false);
  const [err, setErr] = createSignal('');

  const deviceNameMap = createMemo(() => {
    const map = {};
    for (const device of devices()) {
      map[String(device.id)] = device.name || `设备#${device.id}`;
    }
    return map;
  });

  const load = () => {
    setLoadError('');
    runThresholdsLoad();
  };

  onMount(load);

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
    api.thresholds.createThreshold(form())
      .then(() => {
        toast.show('success', '阈值已创建');
        setForm(empty);
        setShowModal(false);
        load();
      })
      .catch((er) => {
        const msg = getErrorMessage(er, '创建失败');
        setErr(msg);
        toast.show('error', msg);
      })
      .finally(() => setSaving(false));
  };

  const saveRepeatInterval = () => {
    const minutes = Number.parseInt(String(repeatIntervalMinutes() || 0), 10);
    if (!Number.isFinite(minutes) || minutes <= 0) {
      toast.show('warning', '重复触发间隔必须大于 0 分钟');
      return;
    }

    const seconds = minutes * 60;
    setSavingRepeatInterval(true);
    api.thresholds.updateAlarmRepeatInterval(seconds)
      .then(() => {
        setRepeatIntervalMinutes(minutes);
        toast.show('success', '重复触发间隔已更新');
      })
      .catch((error) => showErrorToast(toast, error, '更新重复触发间隔失败'))
      .finally(() => setSavingRepeatInterval(false));
  };

  const toggleShield = (threshold) => {
    if (!threshold) return;

    const nextShielded = threshold.shielded === 1 ? 0 : 1;
    const payload = {
      device_id: threshold.device_id,
      field_name: threshold.field_name,
      operator: threshold.operator,
      value: threshold.value,
      severity: threshold.severity,
      enabled: threshold.enabled,
      message: threshold.message,
      shielded: nextShielded,
    };

    api.thresholds.updateThreshold(threshold.id, payload)
      .then(() => {
        toast.show('success', nextShielded === 1 ? '阈值已屏蔽' : '已取消屏蔽');
        load();
      })
      .catch((error) => showErrorToast(toast, error, '更新屏蔽状态失败'));
  };

  const remove = (id) => {
    if (!confirm('删除该阈值？')) return;
    api.thresholds.deleteThreshold(id)
      .then(() => { toast.show('success', '已删除'); load(); })
      .catch((error) => showErrorToast(toast, error, '删除失败'));
  };

  return (
    <div>
      <Card
        title="阈值配置列表"
        extra={(
          <div style="display:flex; gap:8px; align-items:center; flex-wrap:wrap;">
            <span class="text-muted text-xs">重复触发间隔(分钟)</span>
            <input
              class="form-input"
              type="number"
              min="1"
              style="width:110px;"
              value={repeatIntervalMinutes()}
              onInput={(e) => setRepeatIntervalMinutes(Number(e.target.value || 0))}
            />
            <button
              class="btn btn-soft-primary btn-sm"
              onClick={saveRepeatInterval}
              disabled={savingRepeatInterval() || loading()}
            >
              {savingRepeatInterval() ? '保存中...' : '保存间隔'}
            </button>
            <button
              class="btn btn-primary btn-pill btn-sm"
              onClick={() => { setForm(empty); setErr(''); setShowModal(true); }}
            >
              新增阈值
            </button>
          </div>
        )}
      >
        <LoadErrorHint error={loadError()} onRetry={load} />
        <div class="table-container" style="max-height:520px; overflow:auto;">
          <table class="table">
            <thead>
              <tr>
                <th>ID</th>
                <th>设备名</th>
                <th>字段</th>
                <th>条件</th>
                <th>严重性</th>
                <th>状态</th>
                <th>操作</th>
              </tr>
            </thead>
            <tbody>
              <For each={items()}>
                {(item) => (
                  <tr>
                    <td>{item.id}</td>
                    <td>
                      <div>{deviceNameMap()[String(item.device_id)] || `设备#${item.device_id}`}</div>
                      <div class="text-muted text-xs">ID: {item.device_id}</div>
                    </td>
                    <td>{item.field_name}</td>
                    <td>{item.operator} {item.value}</td>
                    <td>{item.severity}</td>
                    <td>
                      <span class={`badge ${item.shielded === 1 ? 'badge-stopped' : 'badge-running'}`}>
                        {item.shielded === 1 ? '已屏蔽' : '生效中'}
                      </span>
                    </td>
                    <td>
                      <div style="display:flex; gap:8px; align-items:center;">
                        <button class="btn btn-outline-primary btn-sm" onClick={() => toggleShield(item)}>
                          {item.shielded === 1 ? '取消屏蔽' : '屏蔽'}
                        </button>
                        <button class="btn btn-outline-danger btn-sm" onClick={() => remove(item.id)}>删除</button>
                      </div>
                    </td>
                  </tr>
                )}
              </For>
              <For each={items().length === 0 ? [1] : []}>
                {() => (
                  <tr>
                    <td colSpan={7} style="text-align:center; padding:24px; color:var(--text-muted);">
                      {loading() ? '加载中...' : '暂无阈值'}
                    </td>
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
                  {devices().map((device) => (
                    <option key={device.id} value={device.id}>{device.name || device.id}</option>
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
