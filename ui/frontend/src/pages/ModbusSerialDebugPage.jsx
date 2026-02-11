import { createSignal, Show, For, onMount } from 'solid-js';
import api from '../api/services';
import Card from '../components/cards';
import { getErrorMessage } from '../api/errorMessages';
import { useToast } from '../components/Toast';

const defaultForm = {
  mode: 'structured',
  resource_id: '',
  serial_port: '',
  baud_rate: '9600',
  data_bits: '8',
  stop_bits: '1',
  parity: 'N',
  timeout_ms: '800',
  raw_request: '01 03 00 00 00 03 CRCA CRCB',
  expect_response_len: '256',
  slave_id: '1',
  function_code: '3',
  address: '0',
  quantity: '1',
  value: '0',
};

function toInt(value, fallback) {
  const n = Number.parseInt(String(value || '').trim(), 10);
  if (!Number.isFinite(n)) return fallback;
  return n;
}

export function ModbusSerialDebugPage() {
  const toast = useToast();
  const [form, setForm] = createSignal({ ...defaultForm });
  const [result, setResult] = createSignal(null);
  const [error, setError] = createSignal('');
  const [submitting, setSubmitting] = createSignal(false);
  const [serialResources, setSerialResources] = createSignal([]);

  const loadSerialResources = async () => {
    try {
      const items = await api.resources.listResources();
      const filtered = Array.isArray(items)
        ? items.filter((item) => String(item.type || '').toLowerCase() === 'serial')
        : [];
      setSerialResources(filtered);
    } catch {
      setSerialResources([]);
    }
  };

  onMount(() => {
    loadSerialResources();
  });

  const updateField = (key, value) => {
    setForm((prev) => ({ ...prev, [key]: value }));
  };

  const submit = async (e) => {
    e.preventDefault();
    setSubmitting(true);
    setError('');

    const current = form();
    const payload = {
      baud_rate: toInt(current.baud_rate, 9600),
      data_bits: toInt(current.data_bits, 8),
      stop_bits: toInt(current.stop_bits, 1),
      parity: String(current.parity || 'N').trim().toUpperCase() || 'N',
      timeout_ms: toInt(current.timeout_ms, 800),
    };

    const resourceIDText = String(current.resource_id || '').trim();
    if (resourceIDText) {
      payload.resource_id = toInt(resourceIDText, 0);
    }
    const serialPort = String(current.serial_port || '').trim();
    if (serialPort) {
      payload.serial_port = serialPort;
    }

    const rawMode = String(current.mode || 'structured') === 'raw';
    if (rawMode) {
      payload.raw_request = String(current.raw_request || '').trim();
      payload.expect_response_len = toInt(current.expect_response_len, 256);
    } else {
      const functionCode = toInt(current.function_code, 3);
      payload.slave_id = toInt(current.slave_id, 1);
      payload.function_code = functionCode;
      payload.address = toInt(current.address, 0);

      if (functionCode === 3) {
        payload.quantity = toInt(current.quantity, 1);
      } else {
        payload.value = toInt(current.value, 0);
      }
    }

    try {
      const data = await api.debug.modbusSerialDebug(payload);
      setResult(data);
      toast.show('success', '调试完成');
    } catch (err) {
      const message = getErrorMessage(err, '调试失败');
      setError(message);
      toast.show('error', message);
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div style="display:flex; flex-direction:column; gap:16px;">
      <Card title="串口 Modbus 调试工具">
        <form class="form" onSubmit={submit}>
          <div class="form-group">
            <label class="form-label">串口资源（可选）</label>
            <div style="display:flex; gap:8px; align-items:center;">
              <select
                class="form-select"
                value={form().resource_id}
                onChange={(e) => {
                  const selectedID = e.target.value;
                  const selected = serialResources().find((item) => String(item.id) === String(selectedID));
                  setForm((prev) => ({
                    ...prev,
                    resource_id: selectedID,
                    serial_port: selected ? String(selected.path || '') : prev.serial_port,
                  }));
                }}
              >
                <option value="">手动输入串口路径</option>
                <For each={serialResources()}>
                  {(item) => <option value={String(item.id)}>{item.name} ({item.path})</option>}
                </For>
              </select>
              <button
                type="button"
                class="btn btn-outline-primary btn-sm"
                onClick={loadSerialResources}
                disabled={submitting()}
              >
                刷新
              </button>
            </div>
          </div>

          <div class="form-group">
            <label class="form-label">串口路径（可选）</label>
            <input
              class="form-input"
              value={form().serial_port}
              onInput={(e) => updateField('serial_port', e.target.value)}
              placeholder="例如 /dev/ttyUSB0"
            />
          </div>

          <div style="display:grid; grid-template-columns:repeat(auto-fit,minmax(140px,1fr)); gap:10px;">
            <div class="form-group">
              <label class="form-label">请求模式</label>
              <select class="form-select" value={form().mode} onChange={(e) => updateField('mode', e.target.value)}>
                <option value="structured">结构化（地址/功能码）</option>
                <option value="raw">完整报文（RAW）</option>
              </select>
            </div>
            <div class="form-group">
              <label class="form-label">波特率</label>
              <input class="form-input" value={form().baud_rate} onInput={(e) => updateField('baud_rate', e.target.value)} />
            </div>
            <div class="form-group">
              <label class="form-label">数据位</label>
              <input class="form-input" value={form().data_bits} onInput={(e) => updateField('data_bits', e.target.value)} />
            </div>
            <div class="form-group">
              <label class="form-label">停止位</label>
              <select class="form-select" value={form().stop_bits} onChange={(e) => updateField('stop_bits', e.target.value)}>
                <option value="1">1</option>
                <option value="2">2</option>
              </select>
            </div>
            <div class="form-group">
              <label class="form-label">校验位</label>
              <select class="form-select" value={form().parity} onChange={(e) => updateField('parity', e.target.value)}>
                <option value="N">N</option>
                <option value="E">E</option>
                <option value="O">O</option>
              </select>
            </div>
            <div class="form-group">
              <label class="form-label">超时 (ms)</label>
              <input class="form-input" value={form().timeout_ms} onInput={(e) => updateField('timeout_ms', e.target.value)} />
            </div>
          </div>

          <Show
            when={String(form().mode || 'structured') === 'raw'}
            fallback={(
              <div style="display:grid; grid-template-columns:repeat(auto-fit,minmax(140px,1fr)); gap:10px;">
                <div class="form-group">
                  <label class="form-label">从站地址</label>
                  <input class="form-input" value={form().slave_id} onInput={(e) => updateField('slave_id', e.target.value)} />
                </div>
                <div class="form-group">
                  <label class="form-label">功能码</label>
                  <select class="form-select" value={form().function_code} onChange={(e) => updateField('function_code', e.target.value)}>
                    <option value="3">03 读保持寄存器</option>
                    <option value="6">06 写单个寄存器</option>
                  </select>
                </div>
                <div class="form-group">
                  <label class="form-label">寄存器地址</label>
                  <input class="form-input" value={form().address} onInput={(e) => updateField('address', e.target.value)} />
                </div>
                <Show
                  when={String(form().function_code) === '3'}
                  fallback={(
                    <div class="form-group">
                      <label class="form-label">写入值</label>
                      <input class="form-input" value={form().value} onInput={(e) => updateField('value', e.target.value)} />
                    </div>
                  )}
                >
                  <div class="form-group">
                    <label class="form-label">寄存器数量</label>
                    <input class="form-input" value={form().quantity} onInput={(e) => updateField('quantity', e.target.value)} />
                  </div>
                </Show>
              </div>
            )}
          >
            <div style="display:flex; flex-direction:column; gap:10px;">
              <div class="form-group">
                <label class="form-label">完整请求报文（HEX/10进制，空格分隔）</label>
                <textarea
                  class="form-input"
                  rows="3"
                  value={form().raw_request}
                  onInput={(e) => updateField('raw_request', e.target.value)}
                  placeholder="例如: 01 03 00 00 00 03 CRCA CRCB"
                  style="font-family: ui-monospace, Menlo, Monaco, monospace;"
                />
                <div style="font-size:12px; color:var(--text-muted); margin-top:4px;">
                  支持占位符：<code>CRCA</code>/<code>CRCB</code>（自动填充 CRC 低/高字节），也可用 <code>CRC</code>/<code>CRC16</code>。
                </div>
              </div>
              <div style="display:grid; grid-template-columns:repeat(auto-fit,minmax(180px,1fr)); gap:10px;">
                <div class="form-group">
                  <label class="form-label">期望响应长度（字节）</label>
                  <input
                    class="form-input"
                    value={form().expect_response_len}
                    onInput={(e) => updateField('expect_response_len', e.target.value)}
                    placeholder="默认 256"
                  />
                </div>
              </div>
            </div>
          </Show>

          <Show when={error()}>
            <div style="color:var(--accent-red); margin-top:4px;">{error()}</div>
          </Show>

          <div style="margin-top:8px; display:flex; gap:8px;">
            <button type="submit" class="btn btn-primary btn-sm" disabled={submitting()}>
              {submitting() ? '调试中...' : '开始调试'}
            </button>
            <button
              type="button"
              class="btn btn-outline-primary btn-sm"
              onClick={() => {
                setForm({ ...defaultForm });
                setError('');
              }}
              disabled={submitting()}
            >
              重置参数
            </button>
          </div>
        </form>
      </Card>

      <Show when={result()}>
        {(data) => (
          <Card title="调试结果">
            <div style="display:flex; flex-direction:column; gap:10px;">
              <div style="font-size:12px; color:var(--text-muted);">串口路径：{data().port || '-'}</div>
              <div>
                <div style="font-size:12px; color:var(--text-muted); margin-bottom:4px;">请求帧</div>
                <div style="font-family: ui-monospace, Menlo, Monaco, monospace; font-size:13px; word-break:break-all; background:rgba(15,23,42,0.65); border:1px solid var(--border-color); padding:8px; border-radius:8px;">{data().request_hex || '-'}</div>
              </div>
              <div>
                <div style="font-size:12px; color:var(--text-muted); margin-bottom:4px;">响应帧</div>
                <div style="font-family: ui-monospace, Menlo, Monaco, monospace; font-size:13px; word-break:break-all; background:rgba(15,23,42,0.65); border:1px solid var(--border-color); padding:8px; border-radius:8px;">{data().response_hex || '-'}</div>
              </div>

              <Show when={typeof data().exception_code === 'number'}>
                <div style="color:var(--accent-orange-light);">
                  设备返回异常码: {data().exception_code}
                </div>
              </Show>

              <Show when={Array.isArray(data().registers) && data().registers.length > 0}>
                <div class="table-container" style="max-height:320px; overflow:auto;">
                  <table class="table">
                    <thead>
                      <tr>
                        <th>序号</th>
                        <th>值</th>
                      </tr>
                    </thead>
                    <tbody>
                      <For each={data().registers}>
                        {(item, index) => (
                          <tr>
                            <td>{index() + 1}</td>
                            <td>{item}</td>
                          </tr>
                        )}
                      </For>
                    </tbody>
                  </table>
                </div>
              </Show>
            </div>
          </Card>
        )}
      </Show>
    </div>
  );
}

export default ModbusSerialDebugPage;
