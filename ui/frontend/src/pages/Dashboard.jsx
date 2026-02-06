import { createSignal, createEffect, onCleanup, Show, For } from 'solid-js';
import { getJSON, unwrapData } from '../api';
import StatusCards from '../components/cards';
import { RealtimeMini } from '../sections/RealtimeMini';
import { LatestAlarms } from '../sections/LatestAlarms';
import { QuickActions } from '../sections/QuickActions';
import { GatewayStatus } from '../sections/GatewayStatus';

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
    <div class="dashboard-root">
      <StatusCards data={status()} loading={loading()} />
      <div class="grid" style="grid-template-columns: 3fr 2fr; gap:24px;">
        <div class="grid" style="grid-template-columns: 1fr; gap:24px;">
          <GatewayStatus />
          <RealtimeMini />
        </div>
        <div class="grid" style="grid-template-columns: 1fr; gap:24px;">
          <LatestAlarms />
          <QuickActions collectorRunning={status()?.collector_running} onRefresh={loadStatus} />
        </div>
      </div>
    </div>
  );
}

export default Dashboard;
