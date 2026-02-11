import { createSignal, Show } from 'solid-js';
import { isActive } from '../router';

export const mainLinks = [
  { to: '/', label: '拓扑视图', icon: '⛓' },
  { to: '/alarms', label: '报警', icon: '⚠' },
  { to: '/realtime', label: '实时', icon: '◈' },
];

export const settingsLinks = [
  { to: '/gateway', label: '网关设置', icon: '⚙' },
  { to: '/resources', label: '资源', icon: '◐' },
  { to: '/devices', label: '设备', icon: '◑' },
  { to: '/drivers', label: '驱动', icon: '▣' },
  { to: '/northbound', label: '北向', icon: '◉' },
  { to: '/thresholds', label: '阈值', icon: '◐' },
];

export const debugLinks = [
  { to: '/debug/modbus-serial', label: '串口 Modbus', icon: '🧪' },
  { to: '/debug/modbus-tcp', label: 'TCP Modbus', icon: '🌐' },
];

function TopNav(props) {
  const [dropdownOpen, setDropdownOpen] = createSignal(false);
  let dropdownRef;
  const dropdownLinks = [...settingsLinks, ...debugLinks];

  const handleLogout = (e) => {
    localStorage.removeItem('gogw_jwt');
  };

  const closeDropdown = () => {
    setDropdownOpen(false);
  };

  const handleClickOutside = (e) => {
    if (dropdownRef && !dropdownRef.contains(e.target)) {
      setDropdownOpen(false);
    }
  };

  return (
    <nav class="nav">
      <div class="container nav-inner">
        <div class="nav-brand">
          <svg class="nav-brand-icon" viewBox="0 0 48 48" fill="none" xmlns="http://www.w3.org/2000/svg">
            <defs>
              <linearGradient id="brandGradient" x1="0%" y1="0%" x2="100%" y2="100%">
                <stop offset="0%" style="stop-color:#3b82f6"/>
                <stop offset="100%" style="stop-color:#8b5cf6"/>
              </linearGradient>
              <linearGradient id="chipGradient" x1="0%" y1="0%" x2="100%" y2="100%">
                <stop offset="0%" style="stop-color:#06b6d4"/>
                <stop offset="100%" style="stop-color:#22d3ee"/>
              </linearGradient>
            </defs>
            <rect x="4" y="4" width="40" height="40" rx="10" fill="url(#brandGradient)" opacity="0.15"/>
            <rect x="4" y="4" width="40" height="40" rx="10" stroke="url(#brandGradient)" stroke-width="2"/>
            <rect x="14" y="14" width="20" height="20" rx="4" fill="url(#chipGradient)" opacity="0.9"/>
            <circle cx="18" cy="18" r="2" fill="white" opacity="0.9"/>
            <circle cx="18" cy="30" r="2" fill="white" opacity="0.6"/>
            <circle cx="30" cy="18" r="2" fill="white" opacity="0.6"/>
            <circle cx="30" cy="30" r="2" fill="white" opacity="0.9"/>
            <path d="M24 10V14M24 34V38M10 24H14M34 24H38" stroke="url(#brandGradient)" stroke-width="2" stroke-linecap="round"/>
          </svg>
          <div class="nav-brand-text">
            <div class="nav-title">
              <span class="brand-hu">Hu</span>
              <span class="brand-shu">Shu</span>
              <span class="brand-gateway">智能网关</span>
            </div>
            <div class="nav-subtitle">
              <span class="subtitle-icon">◆</span>
              工业物联网网关管理平台
            </div>
          </div>
        </div>
        <div class="nav-links" onClick={handleClickOutside}>
          {mainLinks.map((l) => (
            <a
              key={l.to}
              href={l.to}
              class={`nav-btn ${isActive(props.path, l.to) ? 'active' : ''}`}
              onClick={(e) => {
                e.preventDefault();
                props.onNav(l.to);
                closeDropdown();
              }}
            >
              <span class="nav-btn-icon">{l.icon}</span>
              {l.label}
            </a>
          ))}
          <div class="dropdown" ref={dropdownRef}>
            <button
              class={`nav-btn dropdown-toggle ${isActive(props.path, '/resources') || isActive(props.path, '/devices') || isActive(props.path, '/drivers') || isActive(props.path, '/northbound') || isActive(props.path, '/thresholds') || isActive(props.path, '/debug') ? 'active' : ''}`}
              onClick={() => setDropdownOpen(!dropdownOpen())}
            >
              <span class="nav-btn-icon">⚙</span>
              设置
              <span class="dropdown-arrow">▾</span>
            </button>
            <Show when={dropdownOpen()}>
              <div class="dropdown-menu">
                {dropdownLinks.map((l) => (
                  <a
                    key={l.to}
                    href={l.to}
                    class={`dropdown-item ${isActive(props.path, l.to) ? 'active' : ''}`}
                    onClick={(e) => {
                      e.preventDefault();
                      props.onNav(l.to);
                      closeDropdown();
                    }}
                  >
                    <span class="dropdown-item-icon">{l.icon}</span>
                    {l.label}
                  </a>
                ))}
              </div>
            </Show>
          </div>
          <a
            class="nav-btn nav-btn-danger"
            href="/logout"
            onClick={(e) => {
              e.preventDefault();
              handleLogout(e);
              props.onNav('/login');
            }}
          >
            <span class="nav-btn-icon">⏻</span>
            退出
          </a>
        </div>
      </div>
    </nav>
  );
}

export default TopNav;
