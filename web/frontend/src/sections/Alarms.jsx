import { useEffect, useState } from 'preact/hooks';
import { getJSON, postJSON } from '../api';
import { useToast } from '../components/Toast';
import { Card } from '../components/cards';

export function Alarms() {
  const toast = useToast();
  const [items, setItems] = useState([]);
  const [loading, setLoading] = useState(true);

  const load = () => {
    setLoading(true);
    getJSON('/api/alarms')
      .then((res) => setItems(res.data || res))
      .catch(() => toast('error','加载报警失败'))
      .finally(() => setLoading(false));
  };

  useEffect(() => { load(); }, []);

  const ack = (id) => {
    postJSON(`/api/alarms/${id}/acknowledge`, {})
      .then(() => { toast('success','已确认'); load(); })
      .catch(() => toast('error','确认失败'));
  };

  return (
    <Card title="报警日志" extra={<button class="btn" onClick={load}>刷新</button>}>
      {loading ? (
        <div class="text-center" style="padding:48px; color:var(--text-muted);">
          <div class="loading-spinner" style="margin:0 auto 16px;"></div><div>加载中...</div>
        </div>
      ) : (
        <div class="table-container" style="max-height:520px; overflow:auto;">
          <table class="table">
            <thead><tr><th>ID</th><th>设备</th><th>字段</th><th>实际值</th><th>阈值</th><th>条件</th><th>严重性</th><th>时间</th><th>操作</th></tr></thead>
            <tbody>
              {items.map((a)=>(
                <tr key={a.id}>
                  <td>{a.id}</td>
                  <td>{a.device_id}</td>
                  <td>{a.field_name}</td>
                  <td>{a.actual_value}</td>
                  <td>{a.threshold_value}</td>
                  <td>{a.operator}</td>
                  <td>{a.severity}</td>
                  <td>{a.triggered_at}</td>
                  <td><button class="btn" onClick={()=>ack(a.id)}>确认</button></td>
                </tr>
              ))}
              {!items.length && <tr><td colSpan={9} style="text-align:center; padding:24px; color:var(--text-muted);">暂无报警</td></tr>}
            </tbody>
          </table>
        </div>
      )}
    </Card>
  );
}
