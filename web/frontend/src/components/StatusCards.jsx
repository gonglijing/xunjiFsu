import { useEffect, useState } from 'preact/hooks';
import { getJSON } from '../api';

const skeleton = (label, iconClass) => (
  <div class="stat-card">
    <div class={`stat-card-icon ${iconClass}`}>⏳</div>
    <div class="stat-card-label">{label}</div>
    <div class="stat-card-value" style="opacity:0.6">--</div>
  </div>
);

export function StatusCards() {
  const [data, setData] = useState(null);
  const [error, setError] = useState('');

  const load = () => {
    getJSON('/api/status')
      .then((res) => setData(res.data || res))
      .catch(() => setError('状态获取失败'));
  };

  useEffect(() => {
    load();
    const t = setInterval(load, 5000);
    return () => clearInterval(t);
  }, []);

  if (error) return <div style="color:var(--accent-red);padding:8px 0;">{error}</div>;
  if (!data) {
    return (
      <div class="grid grid-cols-4 mb-8">
        {skeleton('设备总数', 'blue')}
        {skeleton('采集使能', 'green')}
        {skeleton('北向启用', 'purple')}
        {skeleton('未确认告警', 'orange')}
      </div>
    );
  }

  return (
    <div class="grid grid-cols-4 mb-8">
      <div class="stat-card">
        <div class="stat-card-icon blue">📊</div>
        <div class="stat-card-label">设备总数</div>
        <div class="stat-card-value">{data.devices?.total ?? 0}</div>
      </div>
      <div class="stat-card">
        <div class="stat-card-icon green">✓</div>
        <div class="stat-card-label">采集使能</div>
        <div class="stat-card-value">{data.devices?.enabled ?? 0}</div>
      </div>
      <div class="stat-card">
        <div class="stat-card-icon purple">⚡</div>
        <div class="stat-card-label">北向启用</div>
        <div class="stat-card-value">{data.northbound?.enabled ?? 0}</div>
      </div>
      <div class="stat-card">
        <div class="stat-card-icon orange">⚠</div>
        <div class="stat-card-label">未确认告警</div>
        <div class="stat-card-value">{data.alarms?.unacked ?? 0}</div>
      </div>
    </div>
  );
}
