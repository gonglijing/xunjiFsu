import { createSignal, createEffect, onCleanup, Show, For } from 'solid-js';
import { getJSON, unwrapData } from '../api';
import StatusCards, { SectionTabs } from '../components/cards';
import { RealtimeMini } from '../sections/RealtimeMini';
import { LatestAlarms } from '../sections/LatestAlarms';
import { QuickActions } from '../sections/QuickActions';

function Dashboard() {
  const [status, setStatus] = createSignal(null);
  const [loading, setLoading] = createSignal(true);

  const loadStatus = () => {
    getJSON('/api/status')
      .then((res) => setStatus(unwrapData(res, null)))
      .catch(() => {})
      .finally(() => setLoading(false));
  };

  createEffect(() => {
    loadStatus();
    const timer = setInterval(loadStatus, 5000);
    onCleanup(() => clearInterval(timer));
  });

  return (
    <div>
      <StatusCards data={status()} loading={loading()} />
      <div class="grid" style="grid-template-columns: 2fr 1fr; gap:24px;">
        <div class="grid" style="grid-template-columns: 1fr; gap:24px;">
          <RealtimeMini />
          <LatestAlarms />
        </div>
        <div class="grid" style="grid-template-columns: 1fr; gap:24px;">
          <QuickActions collectorRunning={status()?.collector_running} onRefresh={loadStatus} />
        </div>
      </div>
    </div>
  );
}

export default Dashboard;
