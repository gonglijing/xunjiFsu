import { createSignal, createEffect, onCleanup, Show, For } from 'solid-js';
import api from '../api/services';
import StatusCards from '../components/cards';
import { RealtimeMini } from '../sections/RealtimeMini';
import { LatestAlarms } from '../sections/LatestAlarms';
import { QuickActions } from '../sections/QuickActions';
import { GatewayStatus } from '../sections/GatewayStatus';

function Dashboard() {
  const [status, setStatus] = createSignal(null);
  const [loading, setLoading] = createSignal(true);

  const loadStatus = () => {
    api.status.getStatus()
      .then((res) => setStatus(res))
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
