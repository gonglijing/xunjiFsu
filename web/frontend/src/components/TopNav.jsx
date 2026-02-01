import { isActive } from '../router';

const links = [
  { to: '/', label: '仪表盘' },
  { to: '/resources', label: '资源' },
  { to: '/devices', label: '设备' },
  { to: '/drivers', label: '驱动' },
  { to: '/northbound', label: '北向' },
  { to: '/storage', label: '存储策略' },
  { to: '/thresholds', label: '阈值' },
  { to: '/alarms', label: '报警' },
  { to: '/realtime', label: '实时' },
  { to: '/history', label: '历史' },
];

export function TopNav({ path, onNav }) {
  return (
    <nav class="nav">
      <div class="container nav-inner">
        <div class="nav-brand">
          <div class="nav-logo">☰</div>
          <div>
            <div class="nav-title">HuShu智能网关</div>
            <div class="nav-subtitle">工业物联网网关管理</div>
          </div>
        </div>
        <div class="nav-links">
          {links.map((l) => (
            <a
              key={l.to}
              href={l.to}
              class={`nav-link ${isActive(path, l.to) ? 'active' : ''}`}
              onClick={(e) => {
                e.preventDefault();
                onNav(l.to);
              }}
            >
              {l.label}
            </a>
          ))}
          <a class="btn btn-danger" href="/logout">退出</a>
        </div>
      </div>
    </nav>
  );
}
