import { createSignal, createEffect } from 'solid-js';
import { getJSON } from '../api';
import Card from '../components/cards';
import { useToast } from '../components/Toast';

function History() {
  const toast = useToast();
  const [deviceID, setDeviceID] = createSignal('');
  const [data, setData] = createSignal([]);
  const [loading, setLoading] = createSignal(false);
  const [devices, setDevices] = createSignal([]);

  createEffect(() => {
    getJSON('/api/devices').then((res) => setDevices(res.data || res)).catch(() => {});
  });

  const load = () => {
    setLoading(true);
    const url = deviceID() ? `/api/data/history?device_id=${deviceID()}` : '/api/data/history';
    getJSON(url)
      .then((res) => setData(res.data || res))
      .catch(() => toast.show('error', '加载历史数据失败'))
      .finally(() => setLoading(false));
  };

  createEffect(() => {
    load();
  });

  return (
    <Card
      title="历史数据"
      extra={
        <div class="flex gap-3">
          <select class="form-select" value={deviceID()} onChange={(e) => setDeviceID(e.target.value)}>
            <option value="">全部设备</option>
            {devices().map((d) => (
              <option key={d.id} value={d.id}>{d.name || d.id}</option>
            ))}
          </select>
          <button class="btn" onClick={load}>刷新</button>
        </div>
      }
    >
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
              </tr>
            </thead>
            <tbody>
              {data().map((p) => (
                <tr key={p.id}>
                  <td>{p.collected_at?.slice(0, 19) || p.CollectedAt}</td>
                  <td>{p.device_name || p.DeviceName}</td>
                  <td>{p.field_name || p.FieldName}</td>
                  <td>{p.value || p.Value}</td>
                </tr>
              ))}
              {data().length === 0 && (
                <tr>
                  <td colSpan={4} style="text-align:center; padding:24px; color:var(--text-muted);">暂无历史数据</td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      )}
    </Card>
  );
}

export default History;
