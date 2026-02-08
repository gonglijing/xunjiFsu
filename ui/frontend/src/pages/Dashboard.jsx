import { createSignal, onMount, onCleanup } from 'solid-js';
import api from '../api/services';
import StatusCards from '../components/cards';
import { RealtimeMini } from '../sections/RealtimeMini';
import { LatestAlarms } from '../sections/LatestAlarms';
import { QuickActions } from '../sections/QuickActions';
import { GatewayStatus } from '../sections/GatewayStatus';
import { usePageLoader } from '../utils/pageLoader';
import { getDashboardStatusPollIntervalMs } from '../utils/runtimeConfig';

const DASHBOARD_STATUS_POLL_INTERVAL_MS = getDashboardStatusPollIntervalMs();

function Dashboard() {
  const [status, setStatus] = createSignal(null);
  const { loading, run: runStatusLoad } = usePageLoader(async () => {
    const res = await api.status.getStatus();
    setStatus(res);
  });

  const loadStatus = () => {
    runStatusLoad();
  };

  onMount(() => {
    loadStatus();
    const timer = setInterval(loadStatus, DASHBOARD_STATUS_POLL_INTERVAL_MS);
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
