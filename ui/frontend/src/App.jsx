import { Show } from 'solid-js';
import { usePath, navigate } from './router';
import SidebarNav from './components/SidebarNav';
import { useAuthGuard } from './utils/authGuard';
import { isLoginRoute, resolveRoute } from './route-config';
import Login from './pages/Login';
import Topology from './pages/Topology';
import AlarmsPage from './pages/AlarmsPage';
import Realtime from './pages/Realtime';
import GatewayPage from './pages/GatewayPage';
import Resources from './pages/Resources';
import { DevicesPage } from './pages/DevicesPage';
import { DriversPage } from './pages/DriversPage';
import { NorthboundPage } from './pages/NorthboundPage';
import { ThresholdsPage } from './pages/ThresholdsPage';
import ModbusSerialDebugPage from './pages/ModbusSerialDebugPage';
import ModbusTCPDebugPage from './pages/ModbusTCPDebugPage';
import { DebugToolsPage } from './pages/DebugToolsPage';

const routeComponentByKey = {
  login: Login,
  home: Topology,
  alarms: AlarmsPage,
  realtime: Realtime,
  gateway: GatewayPage,
  resources: Resources,
  devices: DevicesPage,
  drivers: DriversPage,
  northbound: NorthboundPage,
  thresholds: ThresholdsPage,
  'debug-modbus-serial': ModbusSerialDebugPage,
  'debug-modbus-tcp': ModbusTCPDebugPage,
  'debug-tools': DebugToolsPage,
};

function App() {
  const [path, setNavigate] = usePath();
  useAuthGuard(path, navigate);
  const currentRoute = () => resolveRoute(path());
  const isLogin = () => isLoginRoute(path());
  const CurrentPage = () => routeComponentByKey[currentRoute().key] ?? Topology;

  return (
    <div class={isLogin() ? '' : 'shell-layout'}>
      <Show when={!isLogin()}>
        <aside class="shell-sidebar">
          <SidebarNav path={path()} onNav={setNavigate} />
        </aside>
      </Show>
      <main class={isLogin() ? 'container login-main' : 'shell-main'}>
        <div class={isLogin() ? '' : 'container'} style="padding-top:24px; padding-bottom:32px;">
          <CurrentPage onSuccess={() => setNavigate('/')} />
        </div>
        <div id="toast-container" class="toast-container"></div>
      </main>
    </div>
  );
}

export default App;
