import { createSignal, onMount, For, Show } from 'solid-js';
import api from '../api/services';
import Card from '../components/cards';
import { useToast } from '../components/Toast';
import { formatDateTime } from '../utils/time';
import { showErrorToast } from '../utils/errors';
import { usePageLoader } from '../utils/pageLoader';

function AlarmsPage() {
  const toast = useToast();
  const [items, setItems] = createSignal([]);
  const [selectedIds, setSelectedIds] = createSignal([]);
  const [actionBusy, setActionBusy] = createSignal(false);
  const { loading, run: runAlarmsLoad } = usePageLoader(async () => {
    const res = await api.alarms.listAlarms();
    setItems(res || []);
    setSelectedIds([]);
  }, {
    onError: (err) => showErrorToast(toast, err, '加载告警失败'),
  });

  const load = () => {
    runAlarmsLoad();
  };

  onMount(load);

  const isSelected = (id) => selectedIds().includes(id);

  const toggleSelect = (id, checked) => {
    if (checked) {
      setSelectedIds((prev) => (prev.includes(id) ? prev : [...prev, id]));
      return;
    }
    setSelectedIds((prev) => prev.filter((item) => item !== id));
  };

  const toggleSelectAll = (checked) => {
    if (!checked) {
      setSelectedIds([]);
      return;
    }
    setSelectedIds(items().map((item) => item.id));
  };

  const ack = (id) => {
    api.alarms.acknowledgeAlarm(id)
      .then(() => { toast.show('success', '已确认'); load(); })
      .catch((err) => showErrorToast(toast, err, '确认失败'));
  };

  const removeOne = async (id) => {
    if (!window.confirm('确认删除这条告警吗？')) return;
    setActionBusy(true);
    try {
      await api.alarms.deleteAlarm(id);
      toast.show('success', '删除成功');
      await load();
    } catch (err) {
      showErrorToast(toast, err, '删除失败');
    } finally {
      setActionBusy(false);
    }
  };

  const removeBatch = async () => {
    const ids = selectedIds();
    if (ids.length === 0) {
      toast.show('warning', '请先选择要删除的告警');
      return;
    }
    if (!window.confirm(`确认删除选中的 ${ids.length} 条告警吗？`)) return;

    setActionBusy(true);
    try {
      const result = await api.alarms.batchDeleteAlarms(ids);
      const deleted = result?.deleted ?? ids.length;
      toast.show('success', `已删除 ${deleted} 条`);
      await load();
    } catch (err) {
      showErrorToast(toast, err, '批量删除失败');
    } finally {
      setActionBusy(false);
    }
  };

  const clearAll = async () => {
    if (items().length === 0) {
      toast.show('info', '当前没有可清空的告警');
      return;
    }
    if (!window.confirm('确认清空全部告警日志吗？此操作不可恢复。')) return;

    setActionBusy(true);
    try {
      const result = await api.alarms.clearAlarms();
      const deleted = result?.deleted ?? 0;
      toast.show('success', `已清空 ${deleted} 条告警`);
      await load();
    } catch (err) {
      showErrorToast(toast, err, '清空失败');
    } finally {
      setActionBusy(false);
    }
  };

  const allSelected = () => items().length > 0 && selectedIds().length === items().length;

  return (
    <Card
      title="报警日志"
      extra={(
        <div style="display:flex; gap:8px; align-items:center;">
          <button class="btn btn-ghost btn-sm" onClick={load} disabled={loading() || actionBusy()}><span class="btn-ico" aria-hidden="true">↻</span>刷新</button>
          <button class="btn btn-outline-danger btn-sm" onClick={removeBatch} disabled={loading() || actionBusy() || selectedIds().length === 0}><span class="btn-ico" aria-hidden="true">−</span>批量删除</button>
          <button class="btn btn-outline-danger btn-sm" onClick={clearAll} disabled={loading() || actionBusy() || items().length === 0}><span class="btn-ico" aria-hidden="true">⌫</span>清空</button>
        </div>
      )}
    >
      <div class="table-container" style="max-height:600px; overflow:auto;">
        <table class="table">
          <thead>
            <tr>
              <th>
                <input
                  type="checkbox"
                  checked={allSelected()}
                  onChange={(e) => toggleSelectAll(e.currentTarget.checked)}
                  disabled={items().length === 0 || loading() || actionBusy()}
                />
              </th>
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
                  <td>
                    <input
                      type="checkbox"
                      checked={isSelected(a.id)}
                      onChange={(e) => toggleSelect(a.id, e.currentTarget.checked)}
                      disabled={loading() || actionBusy()}
                    />
                  </td>
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
                    <div style="display:flex; gap:8px; align-items:center;">
                      <Show when={a.acknowledged !== 1}>
                        <button class="btn btn-soft-primary btn-sm" onClick={() => ack(a.id)} disabled={actionBusy()}><span class="btn-ico" aria-hidden="true">✓</span>确认</button>
                      </Show>
                      <button class="btn btn-outline-danger btn-sm" onClick={() => removeOne(a.id)} disabled={actionBusy()}><span class="btn-ico" aria-hidden="true">✕</span>删除</button>
                    </div>
                  </td>
                </tr>
              )}
            </For>
            <For each={items().length === 0 ? [1] : []}>
              {() => (
                <tr>
                  <td colSpan={10} style="text-align:center; padding:24px; color:var(--text-muted);">
                    {loading() ? '加载中...' : '暂无告警'}
                  </td>
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
