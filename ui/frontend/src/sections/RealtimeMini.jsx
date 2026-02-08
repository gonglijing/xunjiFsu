import { createSignal, createEffect, onCleanup, For } from 'solid-js';
import api from '../api/services';
import Card from '../components/cards';
import { useToast } from '../components/Toast';
import { formatDateTime } from '../utils/time';
import { getErrorMessage } from '../api/errorMessages';

export function RealtimeMini() {
  const toast = useToast();
  const [points, setPoints] = createSignal([]);
  const [deviceMap, setDeviceMap] = createSignal(new Map());

  const load = () => {
    api.data.listDataCache()
      .then((list) => {
        list.sort((a, b) => {
          const at = new Date(a.collected_at || a.CollectedAt || 0).getTime();
          const bt = new Date(b.collected_at || b.CollectedAt || 0).getTime();
          return bt - at;
        });
        setPoints(list.slice(0, 8));
      })
      .catch((err) => toast.show('error', getErrorMessage(err, '加载实时数据失败')));
  };

  createEffect(() => {
    api.devices.listDevices()
      .then((list) => {
        const map = new Map();
        list.forEach((d) => map.set(String(d.id), d.name || d.id));
        setDeviceMap(map);
      })
      .catch(() => {});
    load();
    const timer = setInterval(load, 4000);
    onCleanup(() => clearInterval(timer));
  });

  return (
    <Card title="最新采集" extra={<button class="btn" onClick={load}>刷新</button>}>
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
                  <td colSpan={4} style="text-align:center; padding:16px; color:var(--text-muted);">暂无数据</td>
                </tr>
              )}
            </For>
          </tbody>
        </table>
      </div>
    </Card>
  );
}
