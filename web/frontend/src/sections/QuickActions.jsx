import { useState } from 'preact/hooks';
import { postJSON } from '../api';
import { Card } from '../components/cards';
import { useToast } from '../components/Toast';

export function QuickActions({ collectorRunning, onRefresh }) {
  const toast = useToast();
  const [busy, setBusy] = useState(false);

  const toggleCollector = () => {
    setBusy(true);
    const api = collectorRunning ? '/api/collector/stop' : '/api/collector/start';
    postJSON(api, {})
      .then(() => {
        toast('success', collectorRunning ? '已停止采集' : '已启动采集');
        onRefresh?.();
      })
      .catch(() => toast('error', '操作失败'))
      .finally(() => setBusy(false));
  };

  return (
    <Card title="快捷操作">
      <div class="flex" style="gap:12px; flex-wrap:wrap;">
        <button class={`btn ${collectorRunning ? 'btn-danger' : 'btn-success'}`} onClick={toggleCollector} disabled={busy}>
          {collectorRunning ? '停止采集器' : '启动采集器'}
        </button>
      </div>
    </Card>
  );
}
