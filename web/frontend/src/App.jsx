import { TopNav } from './components/TopNav';
import { usePath } from './router';
import { Dashboard } from './pages/Dashboard';
import { Resources } from './pages/Resources';
import { DevicesPage } from './pages/DevicesPage';
import { DriversPage } from './pages/DriversPage';
import { NorthboundPage } from './pages/NorthboundPage';
import { ThresholdsPage } from './pages/ThresholdsPage';
import { AlarmsPage } from './pages/AlarmsPage';
import { Realtime } from './pages/Realtime';
import { History } from './pages/History';
import { StoragePage } from './pages/StoragePage';
import { useEffect, useState } from 'preact/hooks';
import { getJSON } from './api';

export function App() {
  const [path, navigate] = usePath();
  const [authed, setAuthed] = useState(true);

  // 简单鉴权探测
  useEffect(() => {
    getJSON('/api/status').then(() => setAuthed(true)).catch(() => {
      setAuthed(false);
      window.location.href = '/login';
    });
  }, []);

  const render = () => {
    switch (true) {
      case path === '/':
        return <Dashboard />;
      case path.startsWith('/resources'):
        return <Resources />;
      case path.startsWith('/devices'):
        return <DevicesPage />;
      case path.startsWith('/drivers'):
        return <DriversPage />;
      case path.startsWith('/northbound'):
        return <NorthboundPage />;
      case path.startsWith('/storage'):
        return <StoragePage />;
      case path.startsWith('/thresholds'):
        return <ThresholdsPage />;
      case path.startsWith('/alarms'):
        return <AlarmsPage />;
      case path.startsWith('/realtime'):
        return <Realtime />;
      case path.startsWith('/history'):
        return <History />;
      default:
        return <Dashboard />;
    }
  };

  return (
    <div>
      {path !== '/login' && <TopNav path={path} onNav={navigate} />}
      <main class="container" style="padding-top:32px; padding-bottom:32px;">{render()}</main>
      <div id="toast-container" class="toast-container"></div>
    </div>
  );
}
