import { createSignal, onMount, Show } from 'solid-js';
import api from '../api/services';
import Card from '../components/cards';
import { useToast } from '../components/Toast';
import { getErrorMessage } from '../api/errorMessages';
import { showErrorToast } from '../utils/errors';
import { usePageLoader } from '../utils/pageLoader';

function GatewayPage() {
  const toast = useToast();
  const [form, setForm] = createSignal({
    product_key: '',
    device_key: '',
    gateway_name: 'HuShu智能网关',
  });
  const [runtimeForm, setRuntimeForm] = createSignal({
    collector_device_sync_interval: '10s',
    collector_command_poll_interval: '500ms',
    northbound_mqtt_reconnect_interval: '5s',
    driver_serial_read_timeout: '200ms',
    driver_tcp_dial_timeout: '5s',
    driver_tcp_read_timeout: '500ms',
    driver_serial_open_backoff: '200ms',
    driver_tcp_dial_backoff: '200ms',
    driver_serial_open_retries: 0,
    driver_tcp_dial_retries: 0,
  });
  const [runtimeAudits, setRuntimeAudits] = createSignal([]);
  const { loading, run: runGatewayLoad } = usePageLoader(async () => {
    const [data, runtime, audits] = await Promise.all([
      api.gateway.getGatewayConfig(),
      api.gateway.getGatewayRuntimeConfig(),
      api.gateway.getGatewayRuntimeAudits(20),
    ]);
    setForm({
      product_key: data.product_key || '',
      device_key: data.device_key || '',
      gateway_name: data.gateway_name || 'HuShu智能网关',
    });
    setRuntimeForm({
      collector_device_sync_interval: runtime.collector_device_sync_interval || '10s',
      collector_command_poll_interval: runtime.collector_command_poll_interval || '500ms',
      northbound_mqtt_reconnect_interval: runtime.northbound_mqtt_reconnect_interval || '5s',
      driver_serial_read_timeout: runtime.driver_serial_read_timeout || '200ms',
      driver_tcp_dial_timeout: runtime.driver_tcp_dial_timeout || '5s',
      driver_tcp_read_timeout: runtime.driver_tcp_read_timeout || '500ms',
      driver_serial_open_backoff: runtime.driver_serial_open_backoff || '200ms',
      driver_tcp_dial_backoff: runtime.driver_tcp_dial_backoff || '200ms',
      driver_serial_open_retries: runtime.driver_serial_open_retries ?? 0,
      driver_tcp_dial_retries: runtime.driver_tcp_dial_retries ?? 0,
    });
    setRuntimeAudits(Array.isArray(audits) ? audits : []);
  }, {
    onError: (err) => showErrorToast(toast, err, '加载网关配置失败'),
  });
  const [saving, setSaving] = createSignal(false);
  const [runtimeSaving, setRuntimeSaving] = createSignal(false);
  const [syncing, setSyncing] = createSignal(false);
  const [err, setErr] = createSignal('');

  const load = () => {
    runGatewayLoad();
  };

  onMount(load);

  const submit = (e) => {
    e.preventDefault();
    setSaving(true);
    setErr('');

    api.gateway.updateGatewayConfig(form())
      .then(() => {
        toast.show('success', '网关配置已保存');
      })
      .catch((er) => {
        const msg = getErrorMessage(er, '保存失败');
        setErr(msg);
        toast.show('error', msg);
      })
      .finally(() => setSaving(false));
  };

  const syncNorthboundIdentity = () => {
    setSyncing(true);
    api.gateway.syncGatewayIdentityToNorthbound()
      .then((data) => {
        const updated = data.updated?.length || 0;
        const failed = data.failed ? Object.keys(data.failed).length : 0;
        toast.show('success', `同步完成：更新 ${updated} 个，失败 ${failed} 个`);
      })
      .catch((er) => {
        showErrorToast(toast, er, '同步失败');
      })
      .finally(() => setSyncing(false));
  };

  const submitRuntime = (e) => {
    e.preventDefault();
    setRuntimeSaving(true);
    api.gateway.updateGatewayRuntimeConfig({
      ...runtimeForm(),
      driver_serial_open_retries: Number(runtimeForm().driver_serial_open_retries || 0),
      driver_tcp_dial_retries: Number(runtimeForm().driver_tcp_dial_retries || 0),
    })
      .then((data) => {
        setRuntimeForm({
          collector_device_sync_interval: data.collector_device_sync_interval || runtimeForm().collector_device_sync_interval,
          collector_command_poll_interval: data.collector_command_poll_interval || runtimeForm().collector_command_poll_interval,
          northbound_mqtt_reconnect_interval: data.northbound_mqtt_reconnect_interval || runtimeForm().northbound_mqtt_reconnect_interval,
          driver_serial_read_timeout: data.driver_serial_read_timeout || runtimeForm().driver_serial_read_timeout,
          driver_tcp_dial_timeout: data.driver_tcp_dial_timeout || runtimeForm().driver_tcp_dial_timeout,
          driver_tcp_read_timeout: data.driver_tcp_read_timeout || runtimeForm().driver_tcp_read_timeout,
          driver_serial_open_backoff: data.driver_serial_open_backoff || runtimeForm().driver_serial_open_backoff,
          driver_tcp_dial_backoff: data.driver_tcp_dial_backoff || runtimeForm().driver_tcp_dial_backoff,
          driver_serial_open_retries: data.driver_serial_open_retries ?? runtimeForm().driver_serial_open_retries,
          driver_tcp_dial_retries: data.driver_tcp_dial_retries ?? runtimeForm().driver_tcp_dial_retries,
        });
        toast.show('success', '运行时参数已热更新');
        return api.gateway.getGatewayRuntimeAudits(20);
      })
      .then((audits) => {
        setRuntimeAudits(Array.isArray(audits) ? audits : []);
      })
      .catch((er) => {
        showErrorToast(toast, er, '更新运行时参数失败');
      })
      .finally(() => setRuntimeSaving(false));
  };

  return (
    <div>
      <Card title="网关设置">
        <Show when={loading()}>
          <div style="text-align:center; padding:40px; color:var(--text-muted);">加载中...</div>
        </Show>

        <Show when={!loading()}>
          <form class="form" onSubmit={submit} style="max-width:480px;">
            <div class="form-group">
              <label class="form-label">网关名称</label>
              <input
                class="form-input"
                value={form().gateway_name}
                onInput={(e) => setForm({ ...form(), gateway_name: e.target.value })}
                placeholder="网关名称"
              />
            </div>

            <div class="form-group">
              <label class="form-label">ProductKey <span style="color:var(--text-muted); font-weight:normal;">(产品标识)</span></label>
              <input
                class="form-input"
                value={form().product_key}
                onInput={(e) => setForm({ ...form(), product_key: e.target.value })}
                placeholder="请输入产品密钥"
              />
              <div class="form-hint">用于北向平台身份认证</div>
            </div>

            <div class="form-group">
              <label class="form-label">DeviceKey <span style="color:var(--text-muted); font-weight:normal;">(设备标识)</span></label>
              <input
                class="form-input"
                value={form().device_key}
                onInput={(e) => setForm({ ...form(), device_key: e.target.value })}
                placeholder="请输入设备密钥"
              />
              <div class="form-hint">用于北向平台设备认证</div>
            </div>

            <Show when={err()}>
              <div style="color:var(--accent-red); padding:4px 0;">{err()}</div>
            </Show>

            <div class="flex" style={{ gap: '8px', marginTop: '16px' }}>
              <button type="submit" class="btn btn-primary" disabled={saving()}>
                {saving() ? '保存中...' : '保存配置'}
              </button>
              <button type="button" class="btn" onClick={syncNorthboundIdentity} disabled={syncing() || saving()}>
                {syncing() ? '同步中...' : '同步到北向'}
              </button>
            </div>
          </form>
        </Show>
      </Card>

      <Card title="说明" style="margin-top:16px;">
        <div style="color:var(--text-secondary); font-size:0.875rem; line-height:1.8;">
          <p><strong>ProductKey</strong>：网关产品的唯一标识符，用于区分不同型号的网关产品。</p>
          <p><strong>DeviceKey</strong>：网关设备的唯一标识符，用于在同一产品下区分不同设备。</p>
          <p>这两个密钥将在数据上报到北向平台时使用，用于设备认证和数据路由。</p>
        </div>
      </Card>

      <Card title="运行时参数热更新" style="margin-top:16px;">
        <form class="form" onSubmit={submitRuntime} style="max-width:640px;">
          <div class="form-group">
            <label class="form-label">采集设备同步周期</label>
            <input class="form-input" value={runtimeForm().collector_device_sync_interval}
              onInput={(e) => setRuntimeForm({ ...runtimeForm(), collector_device_sync_interval: e.target.value })}
              placeholder="例如 10s" />
          </div>
          <div class="form-group">
            <label class="form-label">采集命令轮询周期</label>
            <input class="form-input" value={runtimeForm().collector_command_poll_interval}
              onInput={(e) => setRuntimeForm({ ...runtimeForm(), collector_command_poll_interval: e.target.value })}
              placeholder="例如 500ms" />
          </div>
          <div class="form-group">
            <label class="form-label">MQTT 重连间隔</label>
            <input class="form-input" value={runtimeForm().northbound_mqtt_reconnect_interval}
              onInput={(e) => setRuntimeForm({ ...runtimeForm(), northbound_mqtt_reconnect_interval: e.target.value })}
              placeholder="例如 5s" />
          </div>
          <div class="form-group">
            <label class="form-label">串口读超时</label>
            <input class="form-input" value={runtimeForm().driver_serial_read_timeout}
              onInput={(e) => setRuntimeForm({ ...runtimeForm(), driver_serial_read_timeout: e.target.value })}
              placeholder="例如 200ms" />
          </div>
          <div class="form-group">
            <label class="form-label">TCP 建连超时</label>
            <input class="form-input" value={runtimeForm().driver_tcp_dial_timeout}
              onInput={(e) => setRuntimeForm({ ...runtimeForm(), driver_tcp_dial_timeout: e.target.value })}
              placeholder="例如 5s" />
          </div>
          <div class="form-group">
            <label class="form-label">TCP 读超时</label>
            <input class="form-input" value={runtimeForm().driver_tcp_read_timeout}
              onInput={(e) => setRuntimeForm({ ...runtimeForm(), driver_tcp_read_timeout: e.target.value })}
              placeholder="例如 500ms" />
          </div>
          <div class="form-group">
            <label class="form-label">串口打开退避</label>
            <input class="form-input" value={runtimeForm().driver_serial_open_backoff}
              onInput={(e) => setRuntimeForm({ ...runtimeForm(), driver_serial_open_backoff: e.target.value })}
              placeholder="例如 200ms" />
          </div>
          <div class="form-group">
            <label class="form-label">TCP 建连退避</label>
            <input class="form-input" value={runtimeForm().driver_tcp_dial_backoff}
              onInput={(e) => setRuntimeForm({ ...runtimeForm(), driver_tcp_dial_backoff: e.target.value })}
              placeholder="例如 200ms" />
          </div>
          <div class="form-group">
            <label class="form-label">串口打开重试次数</label>
            <input class="form-input" type="number" min="0" value={runtimeForm().driver_serial_open_retries}
              onInput={(e) => setRuntimeForm({ ...runtimeForm(), driver_serial_open_retries: e.target.value })}
              placeholder="例如 0" />
          </div>
          <div class="form-group">
            <label class="form-label">TCP 建连重试次数</label>
            <input class="form-input" type="number" min="0" value={runtimeForm().driver_tcp_dial_retries}
              onInput={(e) => setRuntimeForm({ ...runtimeForm(), driver_tcp_dial_retries: e.target.value })}
              placeholder="例如 0" />
          </div>
          <button type="submit" class="btn btn-primary" disabled={runtimeSaving()}>
            {runtimeSaving() ? '更新中...' : '热更新参数'}
          </button>
        </form>
      </Card>

      <Card title="运行参数审计日志" style="margin-top:16px;">
        <Show
          when={runtimeAudits().length > 0}
          fallback={<div style="color:var(--text-muted);">暂无审计记录</div>}
        >
          <div style="display:grid; gap:8px;">
            {runtimeAudits().map((item) => (
              <div style="border:1px solid var(--border-color); border-radius:8px; padding:10px;">
                <div style="font-size:12px; color:var(--text-muted); margin-bottom:4px;">
                  #{item.id} · {item.created_at} · {item.operator_username || 'unknown'} · {item.source_ip || '-'}
                </div>
                <pre style="margin:0; white-space:pre-wrap; word-break:break-word; font-size:12px; color:var(--text-secondary);">{item.changes}</pre>
              </div>
            ))}
          </div>
        </Show>
      </Card>
    </div>
  );
}

export default GatewayPage;
