import { createSignal, onMount, For, Show } from 'solid-js';
import api from '../api/services';
import Card from '../components/cards';
import { useToast } from '../components/Toast';
import { formatDateTime } from '../utils/time';
import { showErrorToast } from '../utils/errors';

export function LatestAlarms() {
  const toast = useToast();
  const [items, setItems] = createSignal([]);

  const load = () => {
    api.alarms.listAlarms()
      .then((res) => setItems((res || []).slice(0, 8)))
      .catch((err) => showErrorToast(toast, err, '加载告警失败'));
  };

  onMount(load);

  const ack = (id) => {
    api.alarms.acknowledgeAlarm(id)
      .then(() => { toast.show('success', '已确认'); load(); })
      .catch((err) => showErrorToast(toast, err, '确认失败'));
  };

  return (
    <Card title="最近告警" extra={<button class="btn" onClick={load}>刷新</button>}>
      <div class="table-container" style="max-height:320px; overflow:auto;">
        <table class="table">
          <thead>
            <tr>
              <th>时间</th>
              <th>设备</th>
              <th>字段</th>
              <th>值</th>
              <th>阈值</th>
              <th>级别</th>
              <th></th>
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
                  <td>
                    <Show when={a.acknowledged !== 1}>
                      <button class="btn btn-primary" onClick={() => ack(a.id)} style="padding:6px 10px;">确认</button>
                    </Show>
                    <Show when={a.acknowledged === 1}>
                      <span class="text-muted text-xs">已确认</span>
                    </Show>
                  </td>
                </tr>
              )}
            </For>
            <For each={items().length === 0 ? [1] : []}>
              {() => (
                <tr>
                  <td colSpan={7} style="text-align:center; padding:16px; color:var(--text-muted);">暂无告警</td>
                </tr>
              )}
            </For>
          </tbody>
        </table>
      </div>
    </Card>
  );
}
