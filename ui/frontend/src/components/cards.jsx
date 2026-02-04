import { createSignal, createEffect, onCleanup } from 'solid-js';

function Card(props) {
  return (
    <div class="card" style={props.style || ''}>
      {(props.title || props.extra) && (
        <div class="card-header">
          {props.title && <h3 class="card-title">{props.title}</h3>}
          {props.extra && <div class="card-extra">{props.extra}</div>}
        </div>
      )}
      <div class="card-body">
        {props.children}
      </div>
    </div>
  );
}

export function SectionTabs(props) {
  return (
    <div class="section-tabs" style="display:flex; gap:4px; margin-bottom:16px;">
      {props.tabs?.map((tab, index) => (
        <button
          key={index}
          class={`btn ${props.active === index ? 'btn-primary' : ''}`}
          onClick={() => props.onChange(index)}
        >
          {tab}
        </button>
      ))}
    </div>
  );
}

export function StatusCards(props) {
  const skeleton = (label, iconClass) => (
    <div class="stat-card">
      <div class={`stat-card-icon ${iconClass}`}>⏳</div>
      <div class="stat-card-label">{label}</div>
      <div class="stat-card-value" style="opacity:0.6">--</div>
    </div>
  );

  return (
    <div class="grid grid-cols-4 mb-8">
      {props.loading ? (
        <>
          {skeleton('设备总数', 'blue')}
          {skeleton('采集使能', 'green')}
          {skeleton('北向启用', 'purple')}
          {skeleton('未确认告警', 'orange')}
        </>
      ) : (
        <>
          <div class="stat-card">
            <div class="stat-card-icon blue">📊</div>
            <div class="stat-card-label">设备总数</div>
            <div class="stat-card-value">{props.data?.devices?.total ?? 0}</div>
          </div>
          <div class="stat-card">
            <div class="stat-card-icon green">✓</div>
            <div class="stat-card-label">采集使能</div>
            <div class="stat-card-value">{props.data?.devices?.enabled ?? 0}</div>
          </div>
          <div class="stat-card">
            <div class="stat-card-icon purple">⚡</div>
            <div class="stat-card-label">北向启用</div>
            <div class="stat-card-value">{props.data?.northbound?.enabled ?? 0}</div>
          </div>
          <div class="stat-card">
            <div class="stat-card-icon orange">⚠</div>
            <div class="stat-card-label">未确认告警</div>
            <div class="stat-card-value">{props.data?.alarms?.unacked ?? 0}</div>
          </div>
        </>
      )}
    </div>
  );
}

export default Card;
