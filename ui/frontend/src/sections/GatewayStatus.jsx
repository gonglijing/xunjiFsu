import { createSignal, createEffect, onCleanup, Show } from 'solid-js';
import { getMetrics } from '../api/metrics';
import Card from '../components/cards';

export function GatewayStatus() {
  const [metrics, setMetrics] = createSignal(null);
  const [loading, setLoading] = createSignal(true);

  const load = () => {
    getMetrics()
      .then((res) => setMetrics(res || null))
      .catch(() => {})
      .finally(() => setLoading(false));
  };

  createEffect(() => {
    load();
    const timer = setInterval(load, 8000);
    onCleanup(() => clearInterval(timer));
  });

  const m = () => metrics();

  return (
    <Card
      title="网关运行状态"
      extra={
        <button class="btn btn-ghost text-xs" type="button" onClick={load}>
          刷新
        </button>
      }
    >
      <Show
        when={!loading() && m()}
        fallback={
          <div style="display:flex; align-items:center; justify-content:center; padding:24px;">
            <div class="loading-spinner" />
          </div>
        }
      >
        <div class="grid" style="grid-template-columns: 2fr 3fr; gap:18px;">
          <div class="grid" style="grid-template-columns: 1fr; gap:12px;">
            <div class="stat-card">
              <div class="stat-card-label">运行时间</div>
              <div class="stat-card-value" style="font-size:1.5rem;">
                {m()?.uptime || '--'}
              </div>
              <div class="text-xs text-muted" style="margin-top:4px;">
                最近刷新时间：{new Date(m()?.timestamp || '').toLocaleString()}
              </div>
            </div>
            <div class="stat-card">
              <div class="stat-card-label">Go 版本</div>
              <div class="stat-card-value" style="font-size:1.25rem;">
                {m()?.go?.version || m()?.go?.Version || '--'}
              </div>
              <div class="text-xs text-muted" style="margin-top:4px;">
                Goroutines：{m()?.go?.goroutines ?? m()?.go?.Goroutines ?? '--'}
              </div>
            </div>
          </div>
          <div class="grid" style="grid-template-columns: repeat(2, minmax(0,1fr)); gap:12px;">
            <div class="stat-card">
              <div class="stat-card-label">内存占用</div>
              <div class="stat-card-value" style="font-size:1.75rem;">
                {m()?.go?.memory_alloc_mb?.toFixed
                  ? m().go.memory_alloc_mb.toFixed(1)
                  : m()?.go?.MemoryAlloc?.toFixed
                  ? m().go.MemoryAlloc.toFixed(1)
                  : '--'}
                <span style="font-size:0.9rem; margin-left:4px;">MB</span>
              </div>
              <div class="text-xs text-muted" style="margin-top:4px;">
                总分配：{' '}
                {m()?.go?.memory_total_mb ?? m()?.go?.MemoryTotal ?? '--'}
                MB
              </div>
            </div>
            <div class="stat-card">
              <div class="stat-card-label">数据库连接</div>
              <div class="stat-card-value" style="font-size:1.75rem;">
                {m()?.database?.data_db_open_conns ??
                  m()?.database?.DataDBConns ??
                  0}
              </div>
              <div class="text-xs text-muted" style="margin-top:4px;">
                Param Idle：{m()?.database?.param_db_idle_conns ??
                  m()?.database?.ParamDBIdleConns ??
                  0}
                ，Data Idle：{m()?.database?.data_db_idle_conns ??
                  m()?.database?.DataDBIdleConns ??
                  0}
              </div>
            </div>
          </div>
        </div>
      </Show>
    </Card>
  );
}
