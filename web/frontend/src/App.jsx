import { createSignal, createEffect, Show } from 'solid-js';
import { usePath, navigate } from './router';
import { getJSON } from './api';
import TopNav from './components/TopNav';
import Dashboard from './pages/Dashboard';
import Login from './pages/Login';
import GatewayPage from './pages/GatewayPage';
import Resources from './pages/Resources';
import { DevicesPage } from './pages/DevicesPage';
import { DriversPage } from './pages/DriversPage';
import { NorthboundPage } from './pages/NorthboundPage';
import { ThresholdsPage } from './pages/ThresholdsPage';
import AlarmsPage from './pages/AlarmsPage';
import Realtime from './pages/Realtime';
import History from './pages/History';
import { StoragePage } from './pages/StoragePage';

function App() {
  const [path, setNavigate, query] = usePath();
  const [authed, setAuthed] = createSignal(true);

  // 鉴权探测
  createEffect(() => {
    const currentPath = path();
    // 登录页不做探测，避免循环跳转
    if (currentPath === '/login') return;
    
    getJSON('/api/status')
      .then(() => setAuthed(true))
      .catch(() => {
        setAuthed(false);
        navigate('/login');
      });
  });

  // 路由渲染
  const render = () => {
    const currentPath = path();
    
    switch (true) {
      case currentPath === '/login':
        return <Login onSuccess={() => setNavigate('/')} />;
      case currentPath === '/':
        return <Dashboard />;
      case currentPath.startsWith('/gateway'):
        return <GatewayPage />;
      case currentPath.startsWith('/resources'):
        return <Resources />;
      case currentPath.startsWith('/devices'):
        return <DevicesPage />;
      case currentPath.startsWith('/drivers'):
        return <DriversPage />;
      case currentPath.startsWith('/northbound'):
        return <NorthboundPage />;
      case currentPath.startsWith('/storage'):
        return <StoragePage />;
      case currentPath.startsWith('/thresholds'):
        return <ThresholdsPage />;
      case currentPath.startsWith('/alarms'):
        return <AlarmsPage />;
      case currentPath.startsWith('/realtime'):
        return <Realtime />;
      case currentPath.startsWith('/history'):
        return <History />;
      default:
        return <Dashboard />;
    }
  };

  return (
    <div>
      <Show when={path() !== '/login'}>
        <TopNav path={path()} onNav={setNavigate} />
      </Show>
      <main class="container" style="padding-top:32px; padding-bottom:32px;">
        {render()}
      </main>
      <div id="toast-container" class="toast-container"></div>
    </div>
  );
}

export default App;
