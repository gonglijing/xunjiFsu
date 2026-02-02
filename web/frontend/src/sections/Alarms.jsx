import { createSignal, createEffect, For } from 'solid-js';
import { getJSON } from '../api';
import { useToast } from '../components/Toast';
import Card from '../components/cards';

export function Alarms() {
  const toast = useToast();
  const [items, setItems] = createSignal([]);
  const [loading, setLoading] = createSignal(true);

  const load = () => {
    setLoading(true);
    getJSON('/api/alarms')
      .then((res) => setItems(res.data || res))
      .catch(() => toast.show('error', '加载告警失败'))
      .finally(() => setLoading(false));
  };

  createEffect(() => {
    load();
  });

  return (
    <Card title="报警日志" extra={<button class="btn" onClick={load}>刷新</button>}>
      {loading() ? (
        <div class="text-center" style="padding:48px; color:var(--text-muted);">
          <div class="loading-spinner" style="margin:0 auto 16px;"></div>
          <div>加载中...</div>
        </div>
      ) : (
        <div class="table-container" style="max-height:600px; overflow:auto;">
          <table class="table">
            <thead>
              <tr>
                <th>时间</th>
                <th>设备ID</th>
                <th>字段</th>
                <th>实际值</th>
                <th>阈值条件</th>
                <th>级别</th>
                <th>消息</th>
                <th>状态</th>
              </tr>
            </thead>
            <tbody>
              <For each={items()}>
                {(a) => (
                  <tr>
                    <td>{a.triggered_at?.slice(0, 19) || a.triggered_at}</td>
                    <td>{a.device_id}</td>
                    <td>{a.field_name}</td>
                    <td>{a.actual_value}</td>
                    <td>{`${a.operator} ${a.threshold_value}`}</td>
                    <td>
                      <span class={`badge ${a.severity === 'critical' ? 'badge-critical' : 'badge-running'}`}>
                        {a.severity || 'warning'}
                      </span>
                    </td>
                    <td>{a.message || '-'}</td>
                    <td>
                      {a.acknowledged === 1 ? (
                        <span class="text-muted">已确认</span>
                      ) : (
                        <span class="text-warning">未确认</span>
                      )}
                    </td>
                  </tr>
                )}
              </For>
              <For each={items().length === 0 ? [1] : []}>
                {() => (
                  <tr>
                    <td colSpan={8} style="text-align:center; padding:24px; color:var(--text-muted);">暂无告警</td>
                  </tr>
                )}
              </For>
            </tbody>
          </table>
        </div>
      )}
    </Card>
  );
}
