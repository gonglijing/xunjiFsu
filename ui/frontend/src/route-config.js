export const routeTable = [
  { key: 'login', path: '/login', layout: 'login' },
  { key: 'home', path: '/', layout: 'shell', navGroup: 'main', label: '拓扑视图', icon: '⛓' },
  { key: 'alarms', path: '/alarms', layout: 'shell', navGroup: 'main', label: '报警', icon: '⚠' },
  { key: 'realtime', path: '/realtime', layout: 'shell', navGroup: 'main', label: '实时', icon: '◈' },
  { key: 'debug-modbus-serial', path: '/debug/modbus-serial', layout: 'shell', navGroup: 'debug', label: '串口 Modbus', icon: '🧪' },
  { key: 'debug-modbus-tcp', path: '/debug/modbus-tcp', layout: 'shell', navGroup: 'debug', label: 'TCP Modbus', icon: '🌐' },
  { key: 'debug-tools', path: '/debug/tools', layout: 'shell' },
  { key: 'gateway', path: '/gateway', layout: 'shell', navGroup: 'settings', label: '网关设置', icon: '⚙' },
  { key: 'resources', path: '/resources', layout: 'shell', navGroup: 'settings', label: '资源', icon: '◐' },
  { key: 'devices', path: '/devices', layout: 'shell', navGroup: 'settings', label: '设备', icon: '◑' },
  { key: 'drivers', path: '/drivers', layout: 'shell', navGroup: 'settings', label: '驱动', icon: '▣' },
  { key: 'northbound', path: '/northbound', layout: 'shell', navGroup: 'settings', label: '北向', icon: '◉' },
  { key: 'thresholds', path: '/thresholds', layout: 'shell', navGroup: 'settings', label: '阈值', icon: '◐' },
];

export const fallbackRoute = routeTable.find((route) => route.key === 'home');
export const mainLinks = routeTable.filter((route) => route.navGroup === 'main');
export const settingsLinks = routeTable.filter((route) => route.navGroup === 'settings');
export const debugLinks = routeTable.filter((route) => route.navGroup === 'debug');

export function resolveRoute(pathname) {
  return routeTable.find((route) => route.path === pathname) ?? fallbackRoute;
}

export function isLoginRoute(pathname) {
  return resolveRoute(pathname).layout === 'login';
}
