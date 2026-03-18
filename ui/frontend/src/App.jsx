import { Show } from 'solid-js';
import { Dynamic } from 'solid-js/web';
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
import ModbusSerialDebugPage from './pages/ModbusSerialDebugPage';
import ModbusTCPDebugPage from './pages/ModbusTCPDebugPage';
import { DebugToolsPage } from './pages/DebugToolsPage';
import { Devices } from './sections/Devices';
import { Drivers } from './sections/Drivers';
import { Northbound } from './sections/Northbound';
import { Thresholds } from './sections/Thresholds';

const routeComponentByKey = {
  login: Login,
  home: Topology,
  alarms: AlarmsPage,
  realtime: Realtime,
  gateway: GatewayPage,
  resources: Resources,
  devices: Devices,
  drivers: Drivers,
  northbound: Northbound,
  thresholds: Thresholds,
  'debug-modbus-serial': ModbusSerialDebugPage,
  'debug-modbus-tcp': ModbusTCPDebugPage,
  'debug-tools': DebugToolsPage,
};

function App() {
  const [path, setNavigate] = usePath();
  useAuthGuard(path, navigate);
  const currentRoute = () => resolveRoute(path());
  const isLogin = () => isLoginRoute(path());
  const currentPage = () => routeComponentByKey[currentRoute().key] ?? Topology;

  return (
    <div class={isLogin() ? '' : 'shell-layout'}>
      <Show when={!isLogin()}>
        <aside class="shell-sidebar">
          <SidebarNav path={path()} onNav={setNavigate} />
        </aside>
      </Show>
      <main class={isLogin() ? 'container login-main' : 'shell-main'}>
        <div class={isLogin() ? '' : 'container'} style="padding-top:24px; padding-bottom:32px;">
          <Dynamic component={currentPage()} onSuccess={() => setNavigate('/')} />
        </div>
        <div id="toast-container" class="toast-container"></div>
      </main>
    </div>
  );
}

export default App;
