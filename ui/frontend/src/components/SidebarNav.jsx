import { isActive } from '../router';
import { mainLinks, settingsLinks, debugLinks } from './TopNav';

function SidebarNav(props) {
  return (
    <div class="sidebar-nav">
      <div class="sidebar-brand">
        <div class="sidebar-brand-mark">GW</div>
        <div class="sidebar-brand-text">
          <div class="sidebar-title">HuShu 网关</div>
          <div class="sidebar-subtitle">Industrial IoT</div>
        </div>
      </div>

      <div class="sidebar-section">
        <div class="sidebar-section-label">监控总览</div>
        {mainLinks.map((l) => (
          <button
            type="button"
            class={`sidebar-link ${isActive(props.path, l.to) ? 'active' : ''}`}
            onClick={() => props.onNav(l.to)}
          >
            <span class="sidebar-link-icon">{l.icon}</span>
            <span>{l.label}</span>
          </button>
        ))}
      </div>

      <div class="sidebar-section">
        <div class="sidebar-section-label">网关配置</div>
        {settingsLinks.map((l) => (
          <button
            type="button"
            class={`sidebar-link ${isActive(props.path, l.to) ? 'active' : ''}`}
            onClick={() => props.onNav(l.to)}
          >
            <span class="sidebar-link-icon">{l.icon}</span>
            <span>{l.label}</span>
          </button>
        ))}
      </div>

      <div class="sidebar-section">
        <div class="sidebar-section-label">调试工具</div>
        {debugLinks.map((l) => (
          <button
            type="button"
            class={`sidebar-link ${isActive(props.path, l.to) ? 'active' : ''}`}
            onClick={() => props.onNav(l.to)}
          >
            <span class="sidebar-link-icon">{l.icon}</span>
            <span>{l.label}</span>
          </button>
        ))}
      </div>

      <div class="sidebar-footer">
        <button
          type="button"
          class="sidebar-link danger"
          onClick={() => {
            localStorage.removeItem('gogw_jwt');
            props.onNav('/login');
          }}
        >
          <span class="sidebar-link-icon">⏻</span>
          <span>退出登录</span>
        </button>
      </div>
    </div>
  );
}

export default SidebarNav;
