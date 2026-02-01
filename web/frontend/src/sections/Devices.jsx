import { useEffect, useMemo, useState } from 'preact/hooks';
import { del, getJSON, postJSON } from '../api';
import { useToast } from '../components/Toast';
import { Card } from '../components/cards';

const defaultForm = {
  name: '',
  description: '',
  product_key: '',
  device_key: '',
  driver_type: 'modbus_rtu',
  serial_port: '/dev/ttyS0',
  baud_rate: 9600,
  data_bits: 8,
  stop_bits: 1,
  parity: 'N',
  ip_address: '',
  port_num: 502,
  device_address: '1',
  collect_interval: 1000,
  timeout: 1000,
  resource_id: null,
};

export function Devices() {
  const toast = useToast();
  const [items, setItems] = useState([]);
  const [resources, setResources] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [search, setSearch] = useState('');
  const [form, setForm] = useState(defaultForm);
  const [submitting, setSubmitting] = useState(false);

  const load = () => {
    setLoading(true);
    Promise.all([getJSON('/api/devices'), getJSON('/api/resources')])
      .then(([devRes, resRes]) => {
        setItems(devRes.data || devRes);
        setResources(resRes.data || resRes);
        setError('');
      })
      .catch(() => setError('加载设备或资源失败'))
      .finally(() => setLoading(false));
  };

  useEffect(() => {
    load();
  }, []);

  const filtered = useMemo(() => {
    const q = search.trim().toLowerCase();
    if (!q) return items;
    return items.filter((d) =>
      [d.name, d.device_address, d.driver_type, d.driver_name]
        .filter(Boolean)
        .some((v) => v.toLowerCase().includes(q))
    );
  }, [items, search]);

  const submit = (e) => {
    e.preventDefault();
    setSubmitting(true);
    postJSON('/api/devices', form)
      .then(() => {
        toast('success', '设备已创建');
        setForm(defaultForm);
        load();
      })
      .catch(() => toast('error', '创建失败'))
      .finally(() => setSubmitting(false));
  };

  const toggle = (id) => {
    postJSON(`/api/devices/${id}/toggle`, {})
      .then(() => load())
      .catch(() => toast('error', '切换失败'));
  };

  const remove = (id) => {
    if (!confirm('确定删除该设备？')) return;
    del(`/api/devices/${id}`)
      .then(() => {
        toast('success', '已删除');
        load();
      })
      .catch(() => toast('error', '删除失败'));
  };

  const filteredResources = useMemo(() => {
    if (!resources.length) return [];
    if (form.driver_type === 'modbus_rtu') return resources.filter((r) => r.type === 'serial');
    if (form.driver_type === 'modbus_tcp') return resources.filter((r) => r.type === 'net');
    return resources;
  }, [resources, form.driver_type]);

  return (
    <div class="grid" style={{ gridTemplateColumns: '3fr 1.4fr', gap: '24px' }}>
      <Card
        title="设备列表"
        extra={
          <div class="flex gap-3">
            <input
              class="form-input"
              style="max-width:240px;"
              placeholder="搜索设备/地址/驱动"
              value={search}
              onInput={(e) => setSearch(e.target.value)}
            />
            <button class="btn" onClick={load}>刷新</button>
          </div>
        }
      >
        {loading ? (
          <div class="text-center" style="padding:48px; color:var(--text-muted);">
            <div class="loading-spinner" style="margin:0 auto 16px;"></div>
            <div>加载中...</div>
          </div>
        ) : error ? (
          <div style="color:var(--accent-red); padding:16px 0;">{error}</div>
        ) : (
          <div class="table-container" style="max-height:520px; overflow:auto;">
            <table class="table">
              <thead>
                <tr>
                  <th>ID</th><th>名称</th><th>产品/设备Key</th><th>驱动类型</th><th>驱动</th><th>资源</th><th>周期(ms)</th><th>状态</th><th>操作</th>
                </tr>
              </thead>
              <tbody>
                {filtered.map((d) => (
                  <tr key={d.id}>
                    <td>{d.id}</td>
                    <td>{d.name}</td>
                    <td style="min-width:140px;">
                      <div class="text-sm">{d.product_key || '-'}</div>
                      <div class="text-muted text-xs">{d.device_key || ''}</div>
                    </td>
                    <td>{d.driver_type}</td>
                    <td>{d.driver_name || (d.driver_id ? `驱动 #${d.driver_id}` : '-')}</td>
                    <td>
                      {d.resource_name ? (
                        <div>
                          <div>{d.resource_name}</div>
                          <div class="text-muted text-xs">{d.resource_path}</div>
                        </div>
                      ) : (
                        <span class="text-muted">未绑定</span>
                      )}
                    </td>
                    <td>{d.collect_interval}</td>
                    <td><span class={`badge ${d.enabled === 1 ? 'badge-running' : 'badge-stopped'}`}>{d.enabled === 1 ? '采集中' : '停止'}</span></td>
                    <td class="flex" style="gap:8px;">
                      <button class={`btn ${d.enabled === 1 ? 'btn-danger' : 'btn-success'}`} onClick={() => toggle(d.id)}>
                        {d.enabled === 1 ? '停止' : '启动'}
                      </button>
                      <button class="btn btn-danger" onClick={() => remove(d.id)}>删除</button>
                    </td>
                  </tr>
                ))}
                {!filtered.length && (
                  <tr><td colSpan={8} style="text-align:center; padding:24px; color:var(--text-muted);">暂无设备</td></tr>
                )}
              </tbody>
            </table>
          </div>
        )}
      </Card>

      <Card title="新增设备">
        <form class="form" onSubmit={submit}>
          <div class="form-group">
            <label class="form-label">名称</label>
            <input class="form-input" value={form.name} onInput={(e)=>setForm({...form,name:e.target.value})} required />
          </div>
          <div class="form-group">
            <label class="form-label">描述</label>
            <input class="form-input" value={form.description} onInput={(e)=>setForm({...form,description:e.target.value})} />
          </div>
          <div class="grid" style={{ gridTemplateColumns: 'repeat(2, 1fr)', gap: '12px' }}>
            <div class="form-group">
              <label class="form-label">ProductKey</label>
              <input class="form-input" value={form.product_key} onInput={(e)=>setForm({...form,product_key:e.target.value})} />
            </div>
            <div class="form-group">
              <label class="form-label">DeviceKey</label>
              <input class="form-input" value={form.device_key} onInput={(e)=>setForm({...form,device_key:e.target.value})} />
            </div>
          </div>
          <div class="form-group">
            <label class="form-label">驱动类型</label>
            <select class="form-select" value={form.driver_type} onChange={(e)=>setForm({...form,driver_type:e.target.value, resource_id: null})}>
              <option value="modbus_rtu">Modbus RTU (串口)</option>
              <option value="modbus_tcp">Modbus TCP</option>
            </select>
          </div>

          {form.driver_type === 'modbus_rtu' ? (
            <div class="grid" style={{ gridTemplateColumns: 'repeat(2, 1fr)', gap: '12px' }}>
              <div class="form-group">
                <label class="form-label">串口</label>
                <input class="form-input" value={form.serial_port} onInput={(e)=>setForm({...form,serial_port:e.target.value})} required />
              </div>
              <div class="form-group">
                <label class="form-label">波特率</label>
                <input class="form-input" type="number" value={form.baud_rate} onInput={(e)=>setForm({...form,baud_rate:+e.target.value})} required />
              </div>
              <div class="form-group">
                <label class="form-label">数据位</label>
                <input class="form-input" type="number" value={form.data_bits} onInput={(e)=>setForm({...form,data_bits:+e.target.value})} required />
              </div>
              <div class="form-group">
                <label class="form-label">停止位</label>
                <input class="form-input" type="number" value={form.stop_bits} onInput={(e)=>setForm({...form,stop_bits:+e.target.value})} required />
              </div>
              <div class="form-group">
                <label class="form-label">校验位</label>
                <select class="form-select" value={form.parity} onChange={(e)=>setForm({...form,parity:e.target.value})}>
                  <option value="N">None</option>
                  <option value="E">Even</option>
                  <option value="O">Odd</option>
                </select>
              </div>
            </div>
          ) : (
            <div class="grid" style={{ gridTemplateColumns: 'repeat(2, 1fr)', gap: '12px' }}>
              <div class="form-group">
                <label class="form-label">IP 地址</label>
                <input class="form-input" value={form.ip_address} onInput={(e)=>setForm({...form,ip_address:e.target.value})} required />
              </div>
              <div class="form-group">
                <label class="form-label">端口</label>
                <input class="form-input" type="number" value={form.port_num} onInput={(e)=>setForm({...form,port_num:+e.target.value})} required />
              </div>
            </div>
          )}

          <div class="grid" style={{ gridTemplateColumns: 'repeat(2, 1fr)', gap: '12px' }}>
            <div class="form-group">
              <label class="form-label">设备地址</label>
              <input class="form-input" value={form.device_address} onInput={(e)=>setForm({...form,device_address:e.target.value})} required />
            </div>
            <div class="form-group">
              <label class="form-label">采集周期 (ms)</label>
              <input class="form-input" type="number" value={form.collect_interval} onInput={(e)=>setForm({...form,collect_interval:+e.target.value})} required />
            </div>
            <div class="form-group">
              <label class="form-label">超时 (ms)</label>
              <input class="form-input" type="number" value={form.timeout} onInput={(e)=>setForm({...form,timeout:+e.target.value})} />
            </div>
          </div>

          <div class="form-group" style="margin-top:8px;">
            <label class="form-label">绑定资源</label>
            <select class="form-select" value={form.resource_id ?? ''} onChange={(e)=>setForm({...form,resource_id: e.target.value ? Number(e.target.value) : null})}>
              <option value="">不绑定</option>
              {filteredResources.map((r)=>(
                <option key={r.id} value={r.id}>{`${r.name} (${r.path})`}</option>
              ))}
            </select>
            <div class="text-muted text-xs" style="margin-top:4px;">同一资源串行访问，建议按驱动类型匹配。</div>
          </div>

          <div class="flex" style={{ gap: '12px', marginTop: '16px' }}>
            <button class="btn btn-primary" type="submit" disabled={submitting} style={{ flex: 1 }}>{submitting ? '提交中...' : '创建设备'}</button>
            <button class="btn" type="button" onClick={()=>setForm(defaultForm)} style={{ flex: 1 }}>重置</button>
          </div>
        </form>
      </Card>
    </div>
  );
}
