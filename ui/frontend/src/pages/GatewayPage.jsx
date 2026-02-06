import { createSignal, createEffect, onCleanup, Show } from 'solid-js';
import { getJSON, post, putJSON, unwrapData } from '../api';
import Card from '../components/cards';
import { useToast } from '../components/Toast';

function GatewayPage() {
  const toast = useToast();
  const [form, setForm] = createSignal({
    product_key: '',
    device_key: '',
    gateway_name: 'HuShu智能网关',
  });
  const [loading, setLoading] = createSignal(true);
  const [saving, setSaving] = createSignal(false);
  const [syncing, setSyncing] = createSignal(false);
  const [err, setErr] = createSignal('');

  const load = () => {
    setLoading(true);
    getJSON('/api/gateway/config')
      .then((res) => {
        const data = unwrapData(res, {});
        setForm({
          product_key: data.product_key || '',
          device_key: data.device_key || '',
          gateway_name: data.gateway_name || 'HuShu智能网关',
        });
      })
      .catch(() => toast.show('error', '加载网关配置失败'))
      .finally(() => setLoading(false));
  };

  createEffect(() => {
    load();
  });

  const submit = (e) => {
    e.preventDefault();
    setSaving(true);
    setErr('');

    putJSON('/api/gateway/config', form())
      .then(() => {
        toast.show('success', '网关配置已保存');
      })
      .catch((er) => {
        setErr(er.message || '保存失败');
        toast.show('error', '保存失败');
      })
      .finally(() => setSaving(false));
  };

  const syncNorthboundIdentity = () => {
    setSyncing(true);
    post('/api/gateway/northbound/sync-identity')
      .then((res) => {
        const data = unwrapData(res, {});
        const updated = data.updated?.length || 0;
        const failed = data.failed ? Object.keys(data.failed).length : 0;
        toast.show('success', `同步完成：更新 ${updated} 个，失败 ${failed} 个`);
      })
      .catch((er) => {
        toast.show('error', er.message || '同步失败');
      })
      .finally(() => setSyncing(false));
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
    </div>
  );
}

export default GatewayPage;
