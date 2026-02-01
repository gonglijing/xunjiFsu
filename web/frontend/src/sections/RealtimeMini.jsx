import { useEffect, useState } from 'preact/hooks';
import { getJSON } from '../api';
import { Card } from '../components/cards';
import { useToast } from '../components/Toast';

export function RealtimeMini() {
  const toast = useToast();
  const [points, setPoints] = useState([]);

  const load = () => {
    getJSON('/api/data/points')
      .then((res) => setPoints((res.data || res).slice(0, 8)))
      .catch(() => toast('error', '加载实时数据失败'));
  };

  useEffect(() => { load(); const t = setInterval(load, 4000); return () => clearInterval(t); }, []);

  return (
    <Card title="最新采集" extra={<button class="btn" onClick={load}>刷新</button>}>
      <div class="table-container" style="max-height:320px; overflow:auto;">
        <table class="table">
          <thead><tr><th>时间</th><th>设备</th><th>字段</th><th>值</th></tr></thead>
          <tbody>
            {points.map((p) => (
              <tr key={p.id || `${p.device_id}-${p.field_name}-${p.collected_at}`}>
                <td>{p.collected_at?.slice(5, 19) || p.CollectedAt}</td>
                <td>{p.device_name || p.DeviceName}</td>
                <td>{p.field_name || p.FieldName}</td>
                <td>{p.value || p.Value}</td>
              </tr>
            ))}
            {!points.length && <tr><td colSpan={4} style="text-align:center; padding:16px; color:var(--text-muted);">暂无数据</td></tr>}
          </tbody>
        </table>
      </div>
    </Card>
  );
}
