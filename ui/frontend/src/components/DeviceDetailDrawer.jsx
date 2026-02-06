import { Show, For } from 'solid-js';
import { formatDateTime } from '../utils/time';

function DeviceDetailDrawer(props) {
  const { visible, device, cache, alarms, loading, onClose } = props;

  return (
    <Show when={visible}>
      <div
        class="modal-backdrop"
        style="position:fixed; inset:0; background:rgba(15,23,42,0.65); display:flex; justify-content:flex-end; z-index:1002;"
        onClick={(e) => { if (e.target === e.currentTarget && onClose) onClose(); }}
      >
        <div
          class="card"
          style="width:420px; max-width:90vw; height:100vh; border-radius:0; border-left:1px solid var(--border-color); display:flex; flex-direction:column;"
        >
          <div class="card-header">
            <h3 class="card-title">
              设备详情 {device() ? `- ${device().name}` : ''}
            </h3>
            <button class="btn btn-ghost" onClick={onClose} style="padding:4px 8px;">✕</button>
          </div>

          <div style="flex:1; overflow:auto; padding:12px 16px 16px;">
            <Show when={loading()}>
              <div class="text-center" style="padding:24px; color:var(--text-muted);">
                <div class="loading-spinner" style="margin:0 auto 16px;"></div>
                <div>加载中...</div>
              </div>
            </Show>

            <Show when={!loading() && device()}>
              {/* 运行概览 */}
              <div style="margin-bottom:16px;">
                <div style="font-size:12px; color:var(--text-muted); margin-bottom:4px;">运行概览</div>
                <div style="font-size:14px; line-height:1.6;">
                  <div><strong>ID：</strong>{device().id}</div>
                  <div><strong>驱动：</strong>{device().driver_name || device().driver_type}</div>
                  <div>
                    <strong>资源：</strong>
                    {device().resource_name
                      ? `${device().resource_name} (${device().resource_path || '-'})`
                      : '未绑定'}
                  </div>
                  <div>
                    <strong>周期：</strong>{device().collect_interval} ms / {device().storage_interval} s
                  </div>
                  <div style="margin-top:4px;">
                    <strong>状态：</strong>
                    <span class={`badge ${device().enabled === 1 ? 'badge-running' : 'badge-stopped'}`}>
                      {device().enabled === 1 ? '启用' : '禁用'}
                    </span>
                  </div>
                </div>
              </div>

              {/* 最新缓存 */}
              <div style="margin-bottom:16px;">
                <div style="font-size:12px; color:var(--text-muted); margin-bottom:4px;">最新缓存</div>
                <Show
                  when={cache().length}
                  fallback={<div class="text-muted text-xs">暂无缓存数据</div>}
                >
                  <div class="table-container" style="max-height:160px; overflow:auto;">
                    <table class="table">
                      <thead>
                        <tr>
                          <th>字段</th>
                          <th>值</th>
                          <th>时间</th>
                        </tr>
                      </thead>
                      <tbody>
                        <For each={cache()}>
                          {(p) => (
                            <tr>
                              <td>{p.field_name || p.FieldName}</td>
                              <td>{p.value || p.Value}</td>
                              <td>{formatDateTime(p.collected_at || p.CollectedAt)}</td>
                            </tr>
                          )}
                        </For>
                      </tbody>
                    </table>
                  </div>
                </Show>
              </div>

              {/* 最近告警 */}
              <div>
                <div style="font-size:12px; color:var(--text-muted); margin-bottom:4px;">最近告警</div>
                <Show
                  when={alarms().length}
                  fallback={<div class="text-muted text-xs">暂无告警</div>}
                >
                  <div class="table-container" style="max-height:160px; overflow:auto;">
                    <table class="table">
                      <thead>
                        <tr>
                          <th>字段</th>
                          <th>值/阈值</th>
                          <th>时间</th>
                        </tr>
                      </thead>
                      <tbody>
                        <For each={alarms()}>
                          {(a) => (
                            <tr>
                              <td>{a.field_name}</td>
                              <td>{a.actual_value} / {a.threshold_value}</td>
                              <td>{formatDateTime(a.triggered_at)}</td>
                            </tr>
                          )}
                      </tbody>
                    </table>
                  </div>
                </Show>
              </div>
            </Show>
          </div>
        </div>
      </div>
    </Show>
  );
}

export default DeviceDetailDrawer;

