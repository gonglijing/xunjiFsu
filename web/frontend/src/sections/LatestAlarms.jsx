import { useEffect, useState } from 'preact/hooks';
import { getJSON, postJSON } from '../api';
import { useToast } from '../components/Toast';
import { Card } from '../components/cards';

export function LatestAlarms() {
  const toast = useToast();
  const [items, setItems] = useState([]);

  const load = () => {
    getJSON('/api/alarms')
      .then((res) => setItems((res.data || res).slice(0, 8)))
      .catch(() => toast('error', '加载告警失败'));
  };

  useEffect(() => { load(); }, []);

  const ack = (id) => {
    postJSON(`/api/alarms/${id}/acknowledge`, {})
      .then(() => { toast('success', '已确认'); load(); })
      .catch(() => toast('error', '确认失败'));
  };

  return (
    <Card title="最近告警" extra={<button class="btn" onClick={load}>刷新</button>}>
      <div class="table-container" style="max-height:320px; overflow:auto;">
        <table class="table">
          <thead><tr><th>时间</th><th>设备</th><th>字段</th><th>值</th><th>阈值</th><th>级别</th><th></th></tr></thead>
          <tbody>
            {items.map((a) => (
              <tr key={a.id}>
                <td>{a.triggered_at?.slice(5, 19) || a.triggered_at}</td>
                <td>{a.device_id}</td>
                <td>{a.field_name}</td>
                <td>{a.actual_value}</td>
                <td>{`${a.operator} ${a.threshold_value}`}</td>
                <td><span class={`badge ${a.severity === 'critical' ? 'badge-critical' : 'badge-running'}`}>{a.severity || 'warn'}</span></td>
                <td>
                  {a.acknowledged === 1 ? (
                    <span class="text-muted text-xs">已确认</span>
                  ) : (
                    <button class="btn btn-primary" onClick={() => ack(a.id)} style="padding:6px 10px;">确认</button>
                  )}
                </td>
              </tr>
            ))}
            {!items.length && <tr><td colSpan={7} style="text-align:center; padding:16px; color:var(--text-muted);">暂无告警</td></tr>}
          </tbody>
        </table>
      </div>
    </Card>
  );
}
