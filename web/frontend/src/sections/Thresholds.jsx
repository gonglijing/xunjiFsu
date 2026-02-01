import { useEffect, useState } from 'preact/hooks';
import { del, getJSON, postJSON } from '../api';
import { useToast } from '../components/Toast';
import { Card } from '../components/cards';

const empty = { device_id: '', field_name: '', operator: '>', value: 0, severity: 'warning', message: '', enabled: 1 };

export function Thresholds() {
  const toast = useToast();
  const [items, setItems] = useState([]);
  const [devices, setDevices] = useState([]);
  const [form, setForm] = useState(empty);

  const load = () => {
    getJSON('/api/thresholds').then((res) => setItems(res.data || res)).catch(()=>toast('error','加载阈值失败'));
  };

  const loadDevices = () => {
    getJSON('/api/devices').then((res)=> setDevices(res.data || res)).catch(()=>{});
  };

  useEffect(() => { load(); loadDevices(); }, []);

  const submit = (e) => {
    e.preventDefault();
    postJSON('/api/thresholds', form)
      .then(() => { toast('success','阈值已创建'); setForm(empty); load(); })
      .catch(() => toast('error','创建失败'));
  };

  const remove = (id) => {
    if (!confirm('删除该阈值？')) return;
    del(`/api/thresholds/${id}`).then(()=>{toast('success','已删除'); load();}).catch(()=>toast('error','删除失败'));
  };

  return (
    <div class="grid" style={{ gridTemplateColumns: '3fr 1.4fr', gap: '24px' }}>
      <Card title="阈值配置列表">
        <div class="table-container" style="max-height:520px; overflow:auto;">
          <table class="table">
            <thead><tr><th>ID</th><th>设备</th><th>字段</th><th>条件</th><th>严重性</th><th>状态</th><th>操作</th></tr></thead>
            <tbody>
              {items.map((t)=>(
                <tr key={t.id}>
                  <td>{t.id}</td>
                  <td>{t.device_id}</td>
                  <td>{t.field_name}</td>
                  <td>{t.operator} {t.value}</td>
                  <td>{t.severity}</td>
                  <td><span class={`badge ${t.enabled===1?'badge-running':'badge-stopped'}`}>{t.enabled===1?'启用':'禁用'}</span></td>
                  <td><button class="btn btn-danger" onClick={()=>remove(t.id)}>删除</button></td>
                </tr>
              ))}
              {!items.length && <tr><td colSpan={7} style="text-align:center; padding:24px; color:var(--text-muted);">暂无阈值</td></tr>}
            </tbody>
          </table>
        </div>
      </Card>

      <Card title="新增阈值">
        <form class="form" onSubmit={submit}>
          <div class="form-group">
            <label class="form-label">设备</label>
            <select class="form-select" value={form.device_id} onChange={(e)=>setForm({...form,device_id:+e.target.value})} required>
              <option value="">选择设备</option>
              {devices.map((d)=>(<option key={d.id} value={d.id}>{d.name || d.id}</option>))}
            </select>
          </div>
          <div class="form-group">
            <label class="form-label">字段名</label>
            <input class="form-input" value={form.field_name} onInput={(e)=>setForm({...form,field_name:e.target.value})} required />
          </div>
          <div class="grid" style={{ gridTemplateColumns:'1fr 1fr', gap:'12px' }}>
            <div class="form-group">
              <label class="form-label">运算符</label>
              <select class="form-select" value={form.operator} onChange={(e)=>setForm({...form,operator:e.target.value})}>
                <option value=">">大于 &gt;</option>
                <option value=">=">大于等于 &gt;=</option>
                <option value="<">小于 &lt;</option>
                <option value="<=">小于等于 &lt;=</option>
                <option value="==">等于 ==</option>
              </select>
            </div>
            <div class="form-group">
              <label class="form-label">阈值</label>
              <input class="form-input" type="number" value={form.value} onInput={(e)=>setForm({...form,value:+e.target.value})} required />
            </div>
          </div>
          <div class="form-group">
            <label class="form-label">严重程度</label>
            <select class="form-select" value={form.severity} onChange={(e)=>setForm({...form,severity:e.target.value})}>
              <option value="warning">warning</option>
              <option value="critical">critical</option>
            </select>
          </div>
          <div class="form-group">
            <label class="form-label">报警消息</label>
            <input class="form-input" value={form.message} onInput={(e)=>setForm({...form,message:e.target.value})} />
          </div>
          <div class="flex" style={{ gap:'12px', marginTop:'12px' }}>
            <button class="btn btn-primary" type="submit" style={{ flex:1 }}>创建</button>
            <button class="btn" type="button" onClick={()=>setForm(empty)} style={{ flex:1 }}>重置</button>
          </div>
        </form>
      </Card>
    </div>
  );
}
