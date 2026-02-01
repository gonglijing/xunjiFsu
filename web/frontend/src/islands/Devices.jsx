import { useEffect, useMemo, useState } from 'preact/hooks';

const statusBadge = (enabled) => {
  const text = enabled ? '采集中' : '停止采集';
  const cls = enabled ? 'badge-running' : 'badge-stopped';
  return <span class={`badge ${cls}`}>{text}</span>;
};

const Loading = () => (
  <div class="text-center" style="padding:48px; color:var(--text-muted);">
    <div class="loading-spinner" style="margin:0 auto 16px;"></div>
    <div>加载中...</div>
  </div>
);

export function Devices() {
  const [devices, setDevices] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [search, setSearch] = useState('');

  const fetchDevices = () => {
    setLoading(true);
    fetch('/api/devices')
      .then((r) => r.json())
      .then((data) => {
        setDevices(data.data || []);
        setError('');
      })
      .catch(() => setError('加载失败'))
      .finally(() => setLoading(false));
  };

  useEffect(() => {
    fetchDevices();
  }, []);

  // listen to external refresh event
  useEffect(() => {
    const handler = () => fetchDevices();
    window.addEventListener('island:refresh', handler);
    return () => window.removeEventListener('island:refresh', handler);
  }, []);

  const filtered = useMemo(() => {
    if (!devices) return [];
    if (!search.trim()) return devices;
    const q = search.toLowerCase();
    return devices.filter((d) =>
      [d.name, d.device_address, d.driver_type, d.driver_name]
        .filter(Boolean)
        .some((v) => v.toLowerCase().includes(q))
    );
  }, [devices, search]);

  const toggleEnable = (id) => {
    fetch(`/api/devices/${id}/toggle`, { method: 'POST' })
      .then((r) => r.json())
      .then(() => fetchDevices())
      .catch(() => showToast('error', '切换失败', '请重试'));
  };

  const remove = (id) => {
    if (!confirm('确定删除?')) return;
    fetch(`/api/devices/${id}`, { method: 'DELETE' })
      .then((r) => r.json())
      .then(() => fetchDevices())
      .catch(() => showToast('error', '删除失败', '请重试'));
  };

  if (loading) return <Loading />;
  if (error) return <div style="padding:16px; color:var(--accent-red);">{error}</div>;

  return (
    <div class="card" style="background: transparent; border: none; padding:0;">
      <div class="card-header" style="padding:0 0 12px;">
        <div class="flex gap-3" style="align-items:center;">
          <input
            class="form-input"
            style="max-width:260px;"
            placeholder="搜索设备/地址/驱动"
            value={search}
            onInput={(e) => setSearch(e.target.value)}
          />
          <button class="btn" onClick={fetchDevices}>刷新</button>
        </div>
      </div>
      <div class="table-container" style="max-height:480px; overflow:auto;">
        <table class="table">
          <thead>
            <tr>
              <th>ID</th>
              <th>名称</th>
              <th>驱动类型</th>
              <th>驱动</th>
              <th>地址</th>
              <th>周期(ms)</th>
              <th>状态</th>
              <th style="width:180px;">操作</th>
            </tr>
          </thead>
          <tbody>
            {filtered.map((d) => (
              <tr key={d.id}>
                <td>{d.id}</td>
                <td>{d.name}</td>
                <td>{d.driver_type}</td>
                <td>{d.driver_name || (d.driver_id ? `驱动 #${d.driver_id}` : '-')}</td>
                <td>{d.device_address}</td>
                <td>{d.collect_interval}</td>
                <td>{statusBadge(d.enabled === 1)}</td>
                <td class="flex" style="gap:8px;">
                  <button class={`btn ${d.enabled === 1 ? 'btn-danger' : 'btn-success'}`} onClick={() => toggleEnable(d.id)}>
                    {d.enabled === 1 ? '停止' : '启动'}
                  </button>
                  <button class="btn btn-danger" onClick={() => remove(d.id)}>删除</button>
                </td>
              </tr>
            ))}
            {!filtered.length && (
              <tr>
                <td colSpan={8} style="text-align:center; padding:24px; color:var(--text-muted);">暂无数据</td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}
