import { createSignal, createEffect, Show, onCleanup } from 'solid-js';
import api from '../api/services';
import Card from '../components/cards';
import { useToast } from '../components/Toast';
import { formatDateTime } from '../utils/time';
import { getErrorMessage } from '../api/errorMessages';
import { showErrorToast, withErrorToast } from '../utils/errors';
import { getRealtimeMiniPollIntervalMs } from '../utils/runtimeConfig';

const SYSTEM_DEVICE_ID = '-1';
const SYSTEM_DEVICE_RAW_NAME = '__system__';
const SYSTEM_DEVICE_LABEL = '系统设备';
const REALTIME_POLL_INTERVAL_MS = getRealtimeMiniPollIntervalMs();

function Realtime() {
  const toast = useToast();
  const showLoadError = withErrorToast(toast, '加载实时数据失败');
  const [devices, setDevices] = createSignal([]);
  const [selected, setSelected] = createSignal('');
  const [points, setPoints] = createSignal([]);
  const [loading, setLoading] = createSignal(false);
  const [historyOpen, setHistoryOpen] = createSignal(false);
  const [historyLoading, setHistoryLoading] = createSignal(false);
  const [historyError, setHistoryError] = createSignal('');
  const [historyView, setHistoryView] = createSignal('chart');
  const [historyData, setHistoryData] = createSignal([]);
  const [historyField, setHistoryField] = createSignal('');
  const [historyDeviceID, setHistoryDeviceID] = createSignal('');
  const [historyDeviceName, setHistoryDeviceName] = createSignal('');
  const [historyStart, setHistoryStart] = createSignal('');
  const [historyEnd, setHistoryEnd] = createSignal('');

  let pollTimer;

  const withSystemDevice = (list) => {
    const normalized = Array.isArray(list)
      ? list.filter((d) => String(d?.id) !== SYSTEM_DEVICE_ID)
      : [];
    return [{ id: SYSTEM_DEVICE_ID, name: SYSTEM_DEVICE_LABEL, enabled: 1 }, ...normalized];
  };

  const getDeviceName = (deviceID, fallbackName = '') => {
    const name = devices().find((d) => String(d.id) === String(deviceID))?.name || fallbackName || '';
    if (String(name) === SYSTEM_DEVICE_RAW_NAME) return SYSTEM_DEVICE_LABEL;
    return name;
  };

  const pad = (n) => String(n).padStart(2, '0');
  const formatLocal = (date) => (
    `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`
  );

  const toISO = (value) => {
    const d = new Date(value);
    if (Number.isNaN(d.getTime())) return '';
    return d.toISOString();
  };

  const openHistory = (p) => {
    const now = new Date();
    const start = new Date(now.getTime() - 24 * 60 * 60 * 1000);
    const deviceName = getDeviceName(p.device_id, p.device_name);
    setHistoryDeviceID(String(p.device_id));
    setHistoryDeviceName(deviceName);
    setHistoryField(p.field_name || '');
    setHistoryStart(formatLocal(start));
    setHistoryEnd(formatLocal(now));
    setHistoryView('chart');
    setHistoryError('');
    setHistoryData([]);
    setHistoryOpen(true);
    fetchHistory(String(p.device_id), p.field_name || '', formatLocal(start), formatLocal(now));
  };

  const fetchHistory = (deviceID = historyDeviceID(), fieldName = historyField(), startVal = historyStart(), endVal = historyEnd()) => {
    if (!deviceID || !fieldName) return;
    const startISO = toISO(startVal);
    const endISO = toISO(endVal);
    if (!startISO || !endISO) {
      setHistoryError('时间格式不正确');
      return;
    }
    if (new Date(startISO) > new Date(endISO)) {
      setHistoryError('开始时间不能晚于结束时间');
      return;
    }
    setHistoryLoading(true);
    setHistoryError('');
    const params = {
      device_id: deviceID,
      field_name: fieldName,
      start: startVal,
      end: endVal,
    };
    api.data.getHistoryData(params)
      .then((list) => {
        list.sort((a, b) => {
          const at = new Date(a.collected_at || a.CollectedAt || 0).getTime();
          const bt = new Date(b.collected_at || b.CollectedAt || 0).getTime();
          return at - bt;
        });
        setHistoryData(list);
      })
      .catch((err) => setHistoryError(getErrorMessage(err, '加载历史数据失败')))
      .finally(() => setHistoryLoading(false));
  };

  const series = () => historyData()
    .map((p) => ({
      t: new Date(p.collected_at || p.CollectedAt || 0).getTime(),
      v: Number.parseFloat(p.value || p.Value),
    }))
    .filter((p) => !Number.isNaN(p.t) && !Number.isNaN(p.v))
    .sort((a, b) => a.t - b.t);

  const buildPath = (list, width, height, padding) => {
    if (list.length === 0) return '';
    const minT = list[0].t;
    const maxT = list[list.length - 1].t;
    let minV = list[0].v;
    let maxV = list[0].v;
    list.forEach((p) => {
      if (p.v < minV) minV = p.v;
      if (p.v > maxV) maxV = p.v;
    });
    const rangeT = maxT - minT || 1;
    const rangeV = maxV - minV || 1;
    return list.map((p, i) => {
      const x = padding + ((p.t - minT) / rangeT) * (width - padding * 2);
      const y = height - padding - ((p.v - minV) / rangeV) * (height - padding * 2);
      return `${i === 0 ? 'M' : 'L'}${x.toFixed(2)} ${y.toFixed(2)}`;
    }).join(' ');
  };

  const buildTicks = (list, width, height, padding) => {
    if (list.length === 0) return { xTicks: [], yTicks: [] };
    const minT = list[0].t;
    const maxT = list[list.length - 1].t;
    let minV = list[0].v;
    let maxV = list[0].v;
    list.forEach((p) => {
      if (p.v < minV) minV = p.v;
      if (p.v > maxV) maxV = p.v;
    });
    const rangeT = maxT - minT || 1;
    const rangeV = maxV - minV || 1;

    const ySteps = 4;
    const yTicks = [];
    for (let i = 0; i <= ySteps; i += 1) {
      const ratio = i / ySteps;
      const value = maxV - rangeV * ratio;
      const y = padding + ratio * (height - padding * 2);
      yTicks.push({ y, value: value.toFixed(2) });
    }

    const xSteps = 4;
    const xTicks = [];
    for (let i = 0; i <= xSteps; i += 1) {
      const ratio = i / xSteps;
      const t = minT + rangeT * ratio;
      const x = padding + ratio * (width - padding * 2);
      const d = new Date(t);
      const label = `${pad(d.getHours())}:${pad(d.getMinutes())}`;
      xTicks.push({ x, label });
    }

    return { xTicks, yTicks };
  };

  const chartTicks = () => buildTicks(series(), 700, 260, 28);

  createEffect(() => {
    api.devices.listDevices().then((list) => {
      const merged = withSystemDevice(list);
      setDevices(merged);
      if (merged.length) {
        setSelected((prev) => prev || String(merged[0].id));
      }
    }).catch((err) => showErrorToast(toast, err, '加载设备失败'));
  });

  createEffect(() => {
    if (!selected()) return;

    if (pollTimer) {
      clearInterval(pollTimer);
      pollTimer = undefined;
    }

    const loadPoints = (isBackground = false) => {
      if (!isBackground) {
        setLoading(true);
      }

      api.data.getDataCacheByDevice(selected())
        .then((list) => {
          list.sort((a, b) => String(a.field_name || '').localeCompare(String(b.field_name || '')));
          setPoints(list);
        })
        .catch(showLoadError)
        .finally(() => {
          if (!isBackground) {
            setLoading(false);
          }
        });
    };

    loadPoints(false);
    pollTimer = setInterval(() => loadPoints(true), REALTIME_POLL_INTERVAL_MS);
  });

  onCleanup(() => {
    if (pollTimer) {
      clearInterval(pollTimer);
      pollTimer = undefined;
    }
  });

  return (
    <Card title="实时数据">
      <div class="tabs" style="flex-wrap:wrap; gap:6px; margin-bottom:16px;">
        {devices().map((d) => (
          <button
            class={`tab-btn ${selected() === String(d.id) ? 'active' : ''}`}
            onClick={() => setSelected(String(d.id))}
          >
            {d.name || d.id}
          </button>
        ))}
        {devices().length === 0 && (
          <div style="color:var(--text-muted); padding:8px 4px;">暂无设备</div>
        )}
      </div>
      {loading() ? (
        <div style="padding:24px;">加载中...</div>
      ) : (
        <div class="table-container" style="max-height:520px; overflow:auto;">
          <table class="table">
            <thead>
              <tr>
                <th>时间</th>
                <th>设备</th>
                <th>字段</th>
                <th>值</th>
                <th>操作</th>
              </tr>
            </thead>
            <tbody>
              {points().map((p) => {
                const deviceName = getDeviceName(p.device_id, p.device_name);
                return (
                  <tr key={p.id || `${p.device_id}-${p.field_name}`}>
                    <td>{formatDateTime(p.collected_at || p.CollectedAt)}</td>
                    <td>{deviceName}</td>
                    <td>{p.field_name || ''}</td>
                    <td>{p.value}</td>
                    <td>
                      <div class="table-actions">
                        <button class="btn btn-soft-primary btn-sm" onClick={() => openHistory(p)}>历史数据</button>
                      </div>
                    </td>
                  </tr>
                );
              })}
              {points().length === 0 && (
                <tr>
                  <td colSpan={5} style="text-align:center; padding:24px; color:var(--text-muted);">暂无数据</td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      )}

      <Show when={historyOpen()}>
        <div
          class="modal-backdrop"
          style="position:fixed; inset:0; background:rgba(0,0,0,0.45); display:flex; align-items:center; justify-content:center; z-index:1001;"
          onClick={(e) => { if (e.target === e.currentTarget) setHistoryOpen(false); }}
        >
          <div class="card" style="width:780px; max-width:95vw;">
            <div class="card-header">
              <h3 class="card-title">历史数据 - {historyDeviceName()} / {historyField()}</h3>
              <button class="btn btn-ghost btn-no-icon btn-only-icon btn-close-lite" onClick={() => setHistoryOpen(false)}>✕</button>
            </div>
            <div class="card-body">
              <div class="grid" style="grid-template-columns: 1.4fr 1.4fr 1fr; gap:12px; margin-bottom:12px;">
                <div class="form-group">
                  <label class="form-label">开始时间</label>
                  <input
                    class="form-input"
                    type="datetime-local"
                    value={historyStart()}
                    onInput={(e) => setHistoryStart(e.target.value)}
                  />
                </div>
                <div class="form-group">
                  <label class="form-label">结束时间</label>
                  <input
                    class="form-input"
                    type="datetime-local"
                    value={historyEnd()}
                    onInput={(e) => setHistoryEnd(e.target.value)}
                  />
                </div>
                <div class="form-group">
                  <label class="form-label">展现方式</label>
                  <div class="toolbar-actions">
                    <button
                      class={`btn btn-sm ${historyView() === 'chart' ? 'btn-primary' : 'btn-outline-primary'}`}
                      onClick={() => setHistoryView('chart')}
                    >
                      折线图
                    </button>
                    <button
                      class={`btn btn-sm ${historyView() === 'list' ? 'btn-primary' : 'btn-outline-primary'}`}
                      onClick={() => setHistoryView('list')}
                    >
                      列表
                    </button>
                  </div>
                </div>
              </div>

              <div class="modal-actions" style="margin-bottom:12px;">
                <button
                  class="btn btn-soft-primary btn-sm"
                  onClick={() => fetchHistory()}
                  disabled={historyLoading()}
                >
                  {historyLoading() ? '加载中...' : '查询'}
                </button>
              </div>

              <Show when={historyError()}>
                <div style="color:var(--accent-red); margin-bottom:12px;">{historyError()}</div>
              </Show>

              <Show when={historyView() === 'chart'}>
                <div style="border:1px solid var(--border-color); border-radius:12px; padding:12px;">
                  <Show when={series().length > 1} fallback={<div style="color:var(--text-muted); padding:16px; text-align:center;">暂无可绘制数据</div>}>
                    <svg viewBox="0 0 700 260" style="width:100%; height:260px;">
                      {chartTicks().yTicks.map((t, idx) => (
                        <g key={`y-${idx}`}>
                          <line x1="28" x2="680" y1={t.y} y2={t.y} stroke="rgba(255,255,255,0.06)" />
                          <text x="8" y={t.y + 4} font-size="10" fill="var(--text-muted)">{t.value}</text>
                        </g>
                      ))}
                      {chartTicks().xTicks.map((t, idx) => (
                        <g key={`x-${idx}`}>
                          <line x1={t.x} x2={t.x} y1="24" y2="232" stroke="rgba(255,255,255,0.04)" />
                          <text x={t.x - 12} y="248" font-size="10" fill="var(--text-muted)">{t.label}</text>
                        </g>
                      ))}
                      <path
                        d={buildPath(series(), 700, 260, 28)}
                        fill="none"
                        stroke="var(--accent-blue)"
                        stroke-width="2"
                      />
                    </svg>
                  </Show>
                </div>
              </Show>

              <Show when={historyView() === 'list'}>
                <div class="table-container" style="max-height:360px; overflow:auto;">
                  <table class="table">
                    <thead>
                      <tr>
                        <th>时间</th>
                        <th>值</th>
                      </tr>
                    </thead>
                    <tbody>
                      {historyData().map((p) => (
                        <tr key={p.id || `${p.device_id}-${p.field_name}-${p.collected_at}`}>
                          <td>{formatDateTime(p.collected_at || p.CollectedAt)}</td>
                          <td>{p.value || p.Value}</td>
                        </tr>
                      ))}
                      {historyData().length === 0 && (
                        <tr>
                          <td colSpan={2} style="text-align:center; padding:16px; color:var(--text-muted);">暂无数据</td>
                        </tr>
                      )}
                    </tbody>
                  </table>
                </div>
              </Show>
            </div>
          </div>
        </div>
      </Show>
    </Card>
  );
}

export default Realtime;
