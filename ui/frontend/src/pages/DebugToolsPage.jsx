import Card from '../components/cards';
import { navigate } from '../router';

const tools = [
  {
    title: '串口 Modbus 调试工具',
    desc: '用于快速验证 RTU 通讯参数、寄存器读写与响应帧。',
    path: '/debug/modbus-serial',
    button: '打开串口工具',
  },
  {
    title: 'Modbus TCP 调试工具',
    desc: '用于快速验证 TCP 连接、寄存器读写与响应数据。',
    path: '/debug/modbus-tcp',
    button: '打开 TCP 工具',
  },
];

export function DebugToolsPage() {
  return (
    <div style="display:flex; flex-direction:column; gap:12px;">
      {tools.map((tool) => (
        <Card title="调试工具">
          <div style="display:flex; align-items:center; justify-content:space-between; gap:12px; flex-wrap:wrap;">
            <div>
              <div style="font-size:14px; color:var(--text-primary);">{tool.title}</div>
              <div style="font-size:12px; color:var(--text-muted); margin-top:4px;">{tool.desc}</div>
            </div>
            <button
              type="button"
              class="btn btn-primary btn-sm"
              onClick={() => navigate(tool.path)}
            >
              {tool.button}
            </button>
          </div>
        </Card>
      ))}
    </div>
  );
}

export default DebugToolsPage;
