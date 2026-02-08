import { Show } from 'solid-js';
import { usePath, navigate } from './router';
import SidebarNav from './components/SidebarNav';
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
import { StoragePage } from './pages/StoragePage';
import Topology from './pages/Topology';
import { useAuthGuard } from './utils/authGuard';

function App() {
  const [path, setNavigate] = usePath();
  useAuthGuard(path, navigate);

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
      case currentPath.startsWith('/topology'):
        return <Topology />;
      default:
        return <Dashboard />;
    }
  };

  return (
    <div class={path() === '/login' ? '' : 'shell-layout'}>
      <Show when={path() !== '/login'}>
        <aside class="shell-sidebar">
          <SidebarNav path={path()} onNav={setNavigate} />
        </aside>
      </Show>
      <main class={path() === '/login' ? 'container login-main' : 'shell-main'}>
        <div class={path() === '/login' ? '' : 'container'} style="padding-top:24px; padding-bottom:32px;">
          {render()}
        </div>
        <div id="toast-container" class="toast-container"></div>
      </main>
    </div>
  );
}

export default App;
