import { createSignal, createEffect, For } from 'solid-js';
import { getJSON, postJSON } from '../api';
import Card from '../components/cards';
import { useToast } from '../components/Toast';
import { formatDateTime } from '../utils/time';

function AlarmsPage() {
  const toast = useToast();
  const [items, setItems] = createSignal([]);

  const load = () => {
    getJSON('/api/alarms')
      .then((res) => setItems((res.data || res)))
      .catch(() => toast.show('error', '加载告警失败'));
  };

  createEffect(() => {
    load();
  });

  const ack = (id) => {
    postJSON(`/api/alarms/${id}/acknowledge`, {})
      .then(() => { toast.show('success', '已确认'); load(); })
      .catch(() => toast.show('error', '确认失败'));
  };

  return (
    <Card title="报警日志" extra={<button class="btn" onClick={load}>刷新</button>}>
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
              <th>操作</th>
            </tr>
          </thead>
          <tbody>
            <For each={items()}>
              {(a) => (
                <tr>
                  <td>{formatDateTime(a.triggered_at)}</td>
                  <td>{a.device_id}</td>
                  <td>{a.field_name}</td>
                  <td>{a.actual_value}</td>
                  <td>{`${a.operator} ${a.threshold_value}`}</td>
                  <td>
                    <span class={`badge ${a.severity === 'critical' ? 'badge-critical' : 'badge-running'}`}>
                      {a.severity || 'warn'}
                    </span>
                  </td>
                  <td>{a.message || '-'}</td>
                  <td>
                    {a.acknowledged === 1 ? (
                      <span class="text-muted text-xs">已确认</span>
                    ) : (
                      <span class="text-warning">未确认</span>
                    )}
                  </td>
                  <td>
                    {a.acknowledged !== 1 && (
                      <button class="btn btn-primary btn-sm" onClick={() => ack(a.id)}>确认</button>
                    )}
                  </td>
                </tr>
              )}
            </For>
            <For each={items().length === 0 ? [1] : []}>
              {() => (
                <tr>
                  <td colSpan={9} style="text-align:center; padding:24px; color:var(--text-muted);">暂无告警</td>
                </tr>
              )}
            </For>
          </tbody>
        </table>
      </div>
    </Card>
  );
}

export default AlarmsPage;
