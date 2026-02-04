import { createSignal, createEffect } from 'solid-js';
import { getJSON } from '../api';
import Card from '../components/cards';
import { useToast } from '../components/Toast';

function Realtime() {
  const toast = useToast();
  const [devices, setDevices] = createSignal([]);
  const [selected, setSelected] = createSignal('');
  const [points, setPoints] = createSignal([]);
  const [loading, setLoading] = createSignal(false);

  createEffect(() => {
    getJSON('/api/devices').then((res) => {
      const list = res.data || res;
      setDevices(list);
      if (list.length) setSelected(String(list[0].id));
    }).catch(() => toast.show('error', '加载设备失败'));
  });

  createEffect(() => {
    if (!selected()) return;
    setLoading(true);
    getJSON(`/api/data/cache/${selected()}`)
      .then((res) => {
        const list = res.data || res || [];
        list.sort((a, b) => String(a.field_name || '').localeCompare(String(b.field_name || '')));
        setPoints(list);
      })
      .catch(() => toast.show('error', '加载实时数据失败'))
      .finally(() => setLoading(false));
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
              </tr>
            </thead>
            <tbody>
              {points().map((p) => {
                const deviceName = devices().find((d) => String(d.id) === String(p.device_id))?.name || p.device_name || '';
                return (
                  <tr key={p.id || `${p.device_id}-${p.field_name}`}>
                    <td>{p.collected_at || p.CollectedAt || ''}</td>
                    <td>{deviceName}</td>
                    <td>{p.field_name || ''}</td>
                    <td>{p.value}</td>
                  </tr>
                );
              })}
              {points().length === 0 && (
                <tr>
                  <td colSpan={4} style="text-align:center; padding:24px; color:var(--text-muted);">暂无数据</td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      )}
    </Card>
  );
}

export default Realtime;
