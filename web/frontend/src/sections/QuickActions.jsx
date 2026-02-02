import { createSignal } from 'solid-js';
import { postJSON } from '../api';
import Card from '../components/cards';
import { useToast } from '../components/Toast';

export function QuickActions(props) {
  const toast = useToast();
  const [busy, setBusy] = createSignal(false);

  const toggleCollector = () => {
    setBusy(true);
    const api = props.collectorRunning ? '/api/collector/stop' : '/api/collector/start';
    postJSON(api, {})
      .then(() => {
        toast.show('success', props.collectorRunning ? '已停止采集' : '已启动采集');
        props.onRefresh?.();
      })
      .catch(() => toast.show('error', '操作失败'))
      .finally(() => setBusy(false));
  };

  return (
    <Card title="快捷操作">
      <div class="flex" style="gap:12px; flex-wrap:wrap;">
        <button 
          class={`btn ${props.collectorRunning ? 'btn-danger' : 'btn-success'}`} 
          onClick={toggleCollector} 
          disabled={busy()}
        >
          {props.collectorRunning ? '停止采集器' : '启动采集器'}
        </button>
      </div>
    </Card>
  );
}
