import { createSignal, Show, For, onMount } from 'solid-js';
import api from '../api/services';
import Card from '../components/cards';
import { getErrorMessage } from '../api/errorMessages';
import { useToast } from '../components/Toast';

const defaultForm = {
  resource_id: '',
  endpoint: '',
  timeout_ms: '2000',
  slave_id: '1',
  function_code: '3',
  address: '0',
  quantity: '1',
  value: '0',
  transaction_id: '',
};

function toInt(value, fallback) {
  const n = Number.parseInt(String(value || '').trim(), 10);
  if (!Number.isFinite(n)) return fallback;
  return n;
}

export function ModbusTCPDebugPage() {
  const toast = useToast();
  const [form, setForm] = createSignal({ ...defaultForm });
  const [result, setResult] = createSignal(null);
  const [error, setError] = createSignal('');
  const [submitting, setSubmitting] = createSignal(false);
  const [netResources, setNetResources] = createSignal([]);

  const loadNetResources = async () => {
    try {
      const items = await api.resources.listResources();
      const filtered = Array.isArray(items)
        ? items.filter((item) => String(item.type || '').toLowerCase() === 'net')
        : [];
      setNetResources(filtered);
    } catch {
      setNetResources([]);
    }
  };

  onMount(() => {
    loadNetResources();
  });

  const updateField = (key, value) => {
    setForm((prev) => ({ ...prev, [key]: value }));
  };

  const submit = async (e) => {
    e.preventDefault();
    setSubmitting(true);
    setError('');

    const current = form();
    const functionCode = toInt(current.function_code, 3);
    const payload = {
      timeout_ms: toInt(current.timeout_ms, 2000),
      slave_id: toInt(current.slave_id, 1),
      function_code: functionCode,
      address: toInt(current.address, 0),
    };

    const resourceIDText = String(current.resource_id || '').trim();
    if (resourceIDText) {
      payload.resource_id = toInt(resourceIDText, 0);
    }

    const endpoint = String(current.endpoint || '').trim();
    if (endpoint) {
      payload.endpoint = endpoint;
    }

    const transactionID = String(current.transaction_id || '').trim();
    if (transactionID) {
      payload.transaction_id = toInt(transactionID, 0);
    }

    if (functionCode === 3) {
      payload.quantity = toInt(current.quantity, 1);
    } else {
      payload.value = toInt(current.value, 0);
    }

    try {
      const data = await api.debug.modbusTCPDebug(payload);
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
      <Card title="Modbus TCP 调试工具">
        <form class="form" onSubmit={submit}>
          <div class="form-group">
            <label class="form-label">TCP 资源（可选）</label>
            <div style="display:flex; gap:8px; align-items:center;">
              <select
                class="form-select"
                value={form().resource_id}
                onChange={(e) => {
                  const selectedID = e.target.value;
                  const selected = netResources().find((item) => String(item.id) === String(selectedID));
                  setForm((prev) => ({
                    ...prev,
                    resource_id: selectedID,
                    endpoint: selected ? String(selected.path || '') : prev.endpoint,
                  }));
                }}
              >
                <option value="">手动输入 endpoint</option>
                <For each={netResources()}>
                  {(item) => <option value={String(item.id)}>{item.name} ({item.path})</option>}
                </For>
              </select>
              <button
                type="button"
                class="btn btn-outline-primary btn-sm"
                onClick={loadNetResources}
                disabled={submitting()}
              >
                刷新
              </button>
            </div>
          </div>

          <div class="form-group">
            <label class="form-label">Endpoint（可选）</label>
            <input
              class="form-input"
              value={form().endpoint}
              onInput={(e) => updateField('endpoint', e.target.value)}
              placeholder="例如 192.168.1.100:502"
            />
          </div>

          <div style="display:grid; grid-template-columns:repeat(auto-fit,minmax(140px,1fr)); gap:10px;">
            <div class="form-group">
              <label class="form-label">超时 (ms)</label>
              <input class="form-input" value={form().timeout_ms} onInput={(e) => updateField('timeout_ms', e.target.value)} />
            </div>
            <div class="form-group">
              <label class="form-label">从站地址</label>
              <input class="form-input" value={form().slave_id} onInput={(e) => updateField('slave_id', e.target.value)} />
            </div>
            <div class="form-group">
              <label class="form-label">事务 ID（可选）</label>
              <input class="form-input" value={form().transaction_id} onInput={(e) => updateField('transaction_id', e.target.value)} />
            </div>
          </div>

          <div style="display:grid; grid-template-columns:repeat(auto-fit,minmax(140px,1fr)); gap:10px;">
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
              <div style="font-size:12px; color:var(--text-muted);">目标地址：{data().endpoint || '-'}</div>
              <div>
                <div style="font-size:12px; color:var(--text-muted); margin-bottom:4px;">请求帧</div>
                <div style="font-family: ui-monospace, Menlo, Monaco, monospace; font-size:13px; word-break:break-all; background:rgba(15,23,42,0.65); border:1px solid var(--border-color); padding:8px; border-radius:8px;">{data().request_hex || '-'}</div>
              </div>
              <div>
                <div style="font-size:12px; color:var(--text-muted); margin-bottom:4px;">响应帧</div>
                <div style="font-family: ui-monospace, Menlo, Monaco, monospace; font-size:13px; word-break:break-all; background:rgba(15,23,42,0.65); border:1px solid var(--border-color); padding:8px; border-radius:8px;">{data().response_hex || '-'}</div>
              </div>

              <Show when={typeof data().exception_code === 'number'}>
                <div style="color:var(--accent-orange-light);">设备返回异常码: {data().exception_code}</div>
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

export default ModbusTCPDebugPage;
