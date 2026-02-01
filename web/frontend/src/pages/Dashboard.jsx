import { StatusCards } from '../components/StatusCards';
import { RealtimeMini } from '../sections/RealtimeMini';
import { LatestAlarms } from '../sections/LatestAlarms';
import { QuickActions } from '../sections/QuickActions';
import { useEffect, useState } from 'preact/hooks';
import { getJSON } from '../api';

export function Dashboard() {
  const [status, setStatus] = useState(null);
  const loadStatus = () => getJSON('/api/status').then((res)=>setStatus(res.data || res)).catch(()=>{});

  useEffect(() => { loadStatus(); }, []);

  return (
    <div>
      <StatusCards />
      <div class="grid" style={{ gridTemplateColumns: '2fr 1fr', gap: '24px' }}>
        <div class="grid" style={{ gridTemplateColumns: '1fr', gap: '24px' }}>
          <RealtimeMini />
          <LatestAlarms />
        </div>
        <div class="grid" style={{ gridTemplateColumns: '1fr', gap: '24px' }}>
          <QuickActions collectorRunning={status?.collector_running} onRefresh={loadStatus} />
        </div>
      </div>
    </div>
  );
}
