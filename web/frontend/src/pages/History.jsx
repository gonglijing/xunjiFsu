import { useEffect, useState } from 'preact/hooks';
import { getJSON } from '../api';
import { Card } from '../components/cards';
import { useToast } from '../components/Toast';

export function History() {
  const toast = useToast();
  const [devices, setDevices] = useState([]);
  const [selected, setSelected] = useState('');
  const [rows, setRows] = useState([]);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    getJSON('/api/devices').then((res)=>{
      const list = res.data || res; setDevices(list); if (list.length) setSelected(String(list[0].id));
    }).catch(()=>toast('error','加载设备失败'));
  }, []);

  const load = () => {
    if (!selected) return;
    setLoading(true);
    getJSON(`/api/data/history?device_id=${selected}`).then((res)=>{
      setRows(res.data || res);
    }).catch(()=>toast('error','加载历史数据失败')).finally(()=>setLoading(false));
  };

  useEffect(() => { load(); }, [selected]);

  return (
    <Card title="历史数据" extra={<select class="form-select" value={selected} onChange={(e)=>setSelected(e.target.value)}>
      {devices.map((d)=>(<option key={d.id} value={d.id}>{d.name || d.id}</option>))}
    </select>}>
      {loading ? <div style="padding:24px;">加载中...</div> : (
        <div class="table-container" style="max-height:520px; overflow:auto;">
          <table class="table">
            <thead><tr><th>时间</th><th>设备</th><th>字段</th><th>值</th></tr></thead>
            <tbody>
              {rows.map((p)=>(
                <tr key={`${p.device_id}-${p.field_name}-${p.collected_at}`}>
                  <td>{p.collected_at || p.CollectedAt || ''}</td>
                  <td>{p.device_name || ''}</td>
                  <td>{p.field_name || ''}</td>
                  <td>{p.value}</td>
                </tr>
              ))}
              {!rows.length && <tr><td colSpan={4} style="text-align:center; padding:24px; color:var(--text-muted);">暂无数据</td></tr>}
            </tbody>
          </table>
        </div>
      )}
    </Card>
  );
}
