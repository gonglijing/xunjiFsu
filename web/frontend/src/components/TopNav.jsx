import { createSignal } from 'solid-js';
import { isActive } from '../router';

const links = [
  { to: '/', label: '仪表盘' },
  { to: '/alarms', label: '报警' },
  { to: '/realtime', label: '实时' },
  { to: '/history', label: '历史' },
];

const settingsLinks = [
  { to: '/resources', label: '资源' },
  { to: '/devices', label: '设备' },
  { to: '/drivers', label: '驱动' },
  { to: '/northbound', label: '北向' },
  { to: '/storage', label: '存储策略' },
  { to: '/thresholds', label: '阈值' },
];

function TopNav(props) {
  const [dropdownOpen, setDropdownOpen] = createSignal(false);
  let dropdownRef;

  const handleLogout = (e) => {
    localStorage.removeItem('gogw_jwt');
  };

  const closeDropdown = () => {
    setDropdownOpen(false);
  };

  // 点击外部关闭下拉菜单
  const handleClickOutside = (e) => {
    if (dropdownRef && !dropdownRef.contains(e.target)) {
      setDropdownOpen(false);
    }
  };

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
        <div class="nav-links" onClick={handleClickOutside}>
          {links.map((l) => (
            <a
              key={l.to}
              href={l.to}
              class={`nav-link ${isActive(props.path, l.to) ? 'active' : ''}`}
              onClick={(e) => {
                e.preventDefault();
                props.onNav(l.to);
                closeDropdown();
              }}
            >
              {l.label}
            </a>
          ))}
          <div class="dropdown" ref={dropdownRef}>
            <button
              class={`nav-link dropdown-toggle ${isActive(props.path, '/resources') || isActive(props.path, '/devices') || isActive(props.path, '/drivers') || isActive(props.path, '/northbound') || isActive(props.path, '/storage') || isActive(props.path, '/thresholds') ? 'active' : ''}`}
              onClick={() => setDropdownOpen(!dropdownOpen())}
            >
              设置 ▾
            </button>
            <Show when={dropdownOpen()}>
              <div class="dropdown-menu">
                {settingsLinks.map((l) => (
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
                    {l.label}
                  </a>
                ))}
              </div>
            </Show>
          </div>
          <a class="btn btn-danger" href="/logout" onClick={handleLogout}>退出</a>
        </div>
      </div>
    </nav>
  );
}

export default TopNav;
