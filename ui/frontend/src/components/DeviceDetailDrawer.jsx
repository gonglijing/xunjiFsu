import { Show, For } from 'solid-js';
import { formatDateTime } from '../utils/time';

function DeviceDetailDrawer(props) {
  const { visible, device, cache, alarms, loading, onClose, onWrite } = props;

  return (
    <Show when={visible}>
      <div
        class="modal-backdrop"
        classList={{ 'device-drawer': true }}
        onClick={(e) => { if (e.target === e.currentTarget && onClose) onClose(); }}
      >
        <div class="card device-drawer-panel">
          <div class="card-header">
            <h3 class="card-title">
              设备详情 {device() ? `- ${device().name}` : ''}
            </h3>
            <div class="toolbar-actions">
              <Show when={device()}>
                <button class="btn btn-soft-primary btn-sm" onClick={() => onWrite && onWrite(device())}>
                  单点写入
                </button>
              </Show>
              <button class="btn btn-ghost btn-no-icon btn-only-icon btn-close-lite" onClick={onClose}>✕</button>
            </div>
          </div>

          <div class="device-drawer-body">
            <Show when={loading()}>
              <div class="loading-state" style="padding:24px;">
                <div class="loading-spinner"></div>
                <div>加载中...</div>
              </div>
            </Show>

            <Show when={!loading() && device()}>
              <div class="device-drawer-section">
                <div class="device-drawer-label">运行概览</div>
                <div class="device-drawer-summary">
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
                  <div class="device-drawer-summary-status">
                    <strong>状态：</strong>
                    <span class={`badge ${device().enabled === 1 ? 'badge-running' : 'badge-stopped'}`}>
                      {device().enabled === 1 ? '启用' : '禁用'}
                    </span>
                  </div>
                </div>
              </div>

              <div class="device-drawer-section">
                <div class="device-drawer-label">最新缓存</div>
                <Show
                  when={cache().length}
                  fallback={<div class="text-muted text-xs">暂无缓存数据</div>}
                >
                  <div class="table-container table-scroll-xs">
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
                <div class="device-drawer-label">最近告警</div>
                <Show
                  when={alarms().length}
                  fallback={<div class="text-muted text-xs">暂无告警</div>}
                >
                  <div class="table-container table-scroll-xs">
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
                        </For>
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
