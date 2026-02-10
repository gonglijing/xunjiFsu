import { createSignal, onMount, onCleanup, For } from 'solid-js';
import api from '../api/services';
import Card from '../components/cards';
import { useToast } from '../components/Toast';
import { formatDateTime } from '../utils/time';
import { showErrorToast } from '../utils/errors';
import { usePageLoader } from '../utils/pageLoader';
import LoadErrorHint from '../components/LoadErrorHint';
import { getRealtimeMiniPollIntervalMs } from '../utils/runtimeConfig';

const REALTIME_MINI_POLL_INTERVAL_MS = getRealtimeMiniPollIntervalMs();
const SYSTEM_DEVICE_ID = '-1';
const SYSTEM_DEVICE_LABEL = '系统设备';

export function RealtimeMini() {
  const toast = useToast();
  const [points, setPoints] = createSignal([]);
  const [deviceMap, setDeviceMap] = createSignal(new Map());
  const {
    loading,
    error: loadError,
    setError: setLoadError,
    run: runRealtimeLoad,
  } = usePageLoader(async () => {
    const list = await api.data.listDataCache();
    list.sort((a, b) => {
      const at = new Date(a.collected_at || a.CollectedAt || 0).getTime();
      const bt = new Date(b.collected_at || b.CollectedAt || 0).getTime();
      return bt - at;
    });
    setPoints(list.slice(0, 8));
  }, {
    errorMessage: '加载实时数据失败',
    onError: (err) => showErrorToast(toast, err, '加载实时数据失败'),
  });

  const load = () => {
    setLoadError('');
    runRealtimeLoad();
  };

  onMount(() => {
    api.devices.listDevices()
      .then((list) => {
        const map = new Map();
        list.forEach((d) => map.set(String(d.id), d.name || d.id));
        map.set(SYSTEM_DEVICE_ID, SYSTEM_DEVICE_LABEL);
        setDeviceMap(map);
      })
      .catch(() => {});
    load();
    const timer = setInterval(load, REALTIME_MINI_POLL_INTERVAL_MS);
    onCleanup(() => clearInterval(timer));
  });

  return (
    <Card title="最新采集" extra={<div class="toolbar-actions"><button class="btn btn-ghost btn-sm" onClick={load}>刷新</button></div>}>
      <LoadErrorHint error={loadError()} onRetry={load} />
      <div class="table-container" style="max-height:320px; overflow:auto;">
        <table class="table">
          <thead>
            <tr>
              <th>时间</th>
              <th>设备</th>
              <th>字段</th>
              <th>值</th>
            </tr>
          </thead>
          <tbody>
            <For each={points()}>
              {(p) => {
                const name = deviceMap().get(String(p.device_id)) || p.device_name || p.DeviceName || p.device_id;
                return (
                  <tr key={p.id || `${p.device_id}-${p.field_name}`}>
                    <td>{formatDateTime(p.collected_at || p.CollectedAt)}</td>
                    <td>{name}</td>
                    <td>{p.field_name || p.FieldName}</td>
                    <td>{p.value || p.Value}</td>
                  </tr>
                );
              }}
            </For>
            <For each={points().length === 0 ? [1] : []}>
              {() => (
                <tr>
                  <td colSpan={4} style="text-align:center; padding:16px; color:var(--text-muted);">
                    {loading() ? '加载中...' : '暂无数据'}
                  </td>
                </tr>
              )}
            </For>
          </tbody>
        </table>
      </div>
    </Card>
  );
}
