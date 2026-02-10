import { createSignal, onMount, onCleanup, Show } from 'solid-js';
import api from '../api/services';
import Card from '../components/cards';
import { usePageLoader } from '../utils/pageLoader';
import { getGatewayMetricsPollIntervalMs } from '../utils/runtimeConfig';
import { getNorthboundTypeLabel, normalizeNorthboundType, NORTHBOUND_TYPE_ORDER } from '../utils/northboundType';

const GATEWAY_METRICS_POLL_INTERVAL_MS = getGatewayMetricsPollIntervalMs();

export function GatewayStatus() {
  const [metrics, setMetrics] = createSignal(null);
  const [northboundStatus, setNorthboundStatus] = createSignal([]);
  const { loading, run: runMetricsLoad } = usePageLoader(async () => {
    const [res, status] = await Promise.all([
      api.metrics.getMetrics(),
      api.northbound.listNorthboundStatus().catch(() => []),
    ]);
    setMetrics(res || null);
    setNorthboundStatus(Array.isArray(status) ? status : []);
  });

  const load = () => {
    runMetricsLoad();
  };

  onMount(() => {
    load();
    const timer = setInterval(load, GATEWAY_METRICS_POLL_INTERVAL_MS);
    onCleanup(() => clearInterval(timer));
  });

  const m = () => metrics();
  const northboundRuntime = () => {
    const buckets = Object.fromEntries(
      NORTHBOUND_TYPE_ORDER.map((type) => [type, {
        type,
        label: getNorthboundTypeLabel(type),
        total: 0,
        connected: 0,
        disconnected: 0,
        disconnectedNames: [],
      }]),
    );

    for (const item of northboundStatus() || []) {
      const itemType = normalizeNorthboundType(item?.type);
      if (!item?.enabled || !buckets[itemType]) continue;

      buckets[itemType].total += 1;
      if (item?.connected) {
        buckets[itemType].connected += 1;
      } else {
        buckets[itemType].disconnected += 1;
        const name = `${item?.name || ''}`.trim();
        if (name) buckets[itemType].disconnectedNames.push(name);
      }
    }

    return NORTHBOUND_TYPE_ORDER.map((type) => buckets[type]);
  };

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
            <div class="stat-card">
              <div class="stat-card-label">北向连接状态</div>
              <div class="stat-card-value" style="font-size:1rem; line-height:1.6; display:flex; flex-direction:column; gap:2px;">
                {northboundRuntime().map((item) => (
                  <div style="display:flex; align-items:center; justify-content:space-between; gap:8px;">
                    <span>{item.label}</span>
                    <span style={`font-weight:600; color:${item.disconnected > 0 ? 'var(--danger)' : 'var(--success)'};`}>
                      {item.total > 0 ? `${item.connected}/${item.total}` : '--'}
                    </span>
                  </div>
                ))}
              </div>
              <div class="text-xs text-muted" style="margin-top:4px;">
                {northboundRuntime().some((item) => item.disconnected > 0)
                  ? northboundRuntime()
                    .filter((item) => item.disconnected > 0)
                    .map((item) => `${item.label} 断开 ${item.disconnected} 个`)
                    .join('，')
                  : '已启用北向均连接正常'}
              </div>
            </div>
          </div>
        </div>
      </Show>
    </Card>
  );
}
