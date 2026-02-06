import { createSignal, createEffect, Show, For } from 'solid-js';
import { del, getJSON, post, postJSON, putJSON, unwrapData } from '../api';
import { useToast } from '../components/Toast';
import Card from '../components/cards';
import CrudTable from '../components/CrudTable';

const empty = { name: '', type: 'http', upload_interval: 5000, config: '{}', enabled: 1 };

function toInt(value, fallback = 0) {
  const n = Number.parseInt(`${value ?? ''}`, 10);
  return Number.isFinite(n) ? n : fallback;
}

function toBool(value) {
  if (typeof value === 'boolean') return value;
  if (typeof value === 'string') {
    const v = value.trim().toLowerCase();
    return v === 'true' || v === '1' || v === 'yes';
  }
  return !!value;
}

function buildSchemaDefaults(schemaFields) {
  const defaults = {};
  for (const field of schemaFields || []) {
    defaults[field.key] = field.default;
  }
  return defaults;
}

function hasSchemaField(schemaFields, key) {
  return (schemaFields || []).some((field) => field.key === key);
}

function normalizeXunJiConfig(raw, schemaFields, uploadIntervalFallback = 5000) {
  const src = raw && typeof raw === 'object' ? raw : {};
  const out = { ...buildSchemaDefaults(schemaFields) };

  for (const field of schemaFields || []) {
    const value = src[field.key];
    if (value === undefined || value === null || value === '') continue;

    if (field.type === 'int') out[field.key] = toInt(value, field.default ?? 0);
    else if (field.type === 'bool') out[field.key] = toBool(value);
    else out[field.key] = `${value}`;
  }

  if (hasSchemaField(schemaFields, 'uploadIntervalMs') && toInt(out.uploadIntervalMs, 0) <= 0) {
    const fallback = toInt(src.uploadIntervalMs, toInt(src.reportIntervalMs, toInt(uploadIntervalFallback, 5000)));
    out.uploadIntervalMs = fallback > 0 ? fallback : 5000;
  }

  if (hasSchemaField(schemaFields, 'qos')) {
    out.qos = toInt(out.qos, 0);
    if (out.qos < 0) out.qos = 0;
    if (out.qos > 2) out.qos = 2;
  }

  return out;
}

function safeParseJSON(value, fallback = {}) {
  try {
    return JSON.parse(value || '{}');
  } catch {
    return fallback;
  }
}

function validateXunJiConfig(config, schemaFields) {
  const errors = {};

  for (const field of schemaFields || []) {
    if (!field.required) continue;
    const value = config[field.key];
    if (field.type === 'string' && !`${value ?? ''}`.trim()) {
      errors[field.key] = `${field.label} 为必填`;
    }
  }

  if (hasSchemaField(schemaFields, 'qos')) {
    const qos = toInt(config.qos, -1);
    if (qos < 0 || qos > 2) {
      errors.qos = 'QOS 必须在 0~2 之间';
    }
  }

  if (hasSchemaField(schemaFields, 'uploadIntervalMs') && toInt(config.uploadIntervalMs, 0) <= 0) {
    errors.uploadIntervalMs = '上传周期必须大于 0';
  }

  if (hasSchemaField(schemaFields, 'alarmFlushIntervalMs') && toInt(config.alarmFlushIntervalMs, 0) <= 0) {
    errors.alarmFlushIntervalMs = '报警刷新周期必须大于 0';
  }

  if (hasSchemaField(schemaFields, 'alarmBatchSize') && toInt(config.alarmBatchSize, 0) <= 0) {
    errors.alarmBatchSize = '报警批量条数必须大于 0';
  }

  if (hasSchemaField(schemaFields, 'alarmQueueSize') && toInt(config.alarmQueueSize, 0) <= 0) {
    errors.alarmQueueSize = '报警队列长度必须大于 0';
  }

  if (hasSchemaField(schemaFields, 'realtimeQueueSize') && toInt(config.realtimeQueueSize, 0) <= 0) {
    errors.realtimeQueueSize = '实时队列长度必须大于 0';
  }

  return errors;
}

export function Northbound() {
  const toast = useToast();
  const [items, setItems] = createSignal([]);
  const [runtime, setRuntime] = createSignal([]);
  const [loading, setLoading] = createSignal(true);
  const [form, setForm] = createSignal(empty);
  const [editing, setEditing] = createSignal(null);
  const [showModal, setShowModal] = createSignal(false);
  const [saving, setSaving] = createSignal(false);
  const [syncing, setSyncing] = createSignal(false);
  const [xunjiSchema, setXunjiSchema] = createSignal([]);
  const [xunjiSchemaLoading, setXunjiSchemaLoading] = createSignal(false);
  const [xunjiSchemaError, setXunjiSchemaError] = createSignal('');
  const [xunjiConfig, setXunjiConfig] = createSignal({});
  const [xunjiErrors, setXunjiErrors] = createSignal({});

  const runtimeByName = () => {
    const map = {};
    for (const item of runtime()) {
      if (item?.name) map[item.name] = item;
    }
    return map;
  };

  const buildXunJiPayload = () => {
    return normalizeXunJiConfig(xunjiConfig(), xunjiSchema(), form().upload_interval);
  };

  const loadXunJiSchema = (silent = false) => {
    setXunjiSchemaLoading(true);
    return getJSON('/api/northbound/schema?type=xunji')
      .then((schemaResult) => {
        const schemaData = unwrapData(schemaResult, {});
        const fields = Array.isArray(schemaData?.fields) ? schemaData.fields : [];
        setXunjiSchema(fields);

        if (fields.length === 0) {
          setXunjiSchemaError('XUNJI Schema 为空，请检查后端配置');
          if (!silent) {
            toast.show('error', 'XUNJI Schema 为空，请检查后端配置');
          }
          return;
        }

        setXunjiSchemaError('');
        setXunjiConfig((prev) => normalizeXunJiConfig(prev, fields, form().upload_interval));
      })
      .catch((err) => {
        const message = err?.message || '加载 XUNJI Schema 失败';
        setXunjiSchema([]);
        setXunjiSchemaError(message);
        if (!silent) {
          toast.show('error', message);
        }
      })
      .finally(() => setXunjiSchemaLoading(false));
  };

  const load = () => {
    setLoading(true);
    Promise.all([
      getJSON('/api/northbound'),
      getJSON('/api/northbound/status'),
    ])
      .then(([configs, status]) => {
        setItems(unwrapData(configs, []));
        setRuntime(unwrapData(status, []));
      })
      .catch(() => toast.show('error', '加载北向配置失败'))
      .finally(() => setLoading(false));
  };

  createEffect(() => {
    load();
    loadXunJiSchema(true);
  });

  const submit = (e) => {
    e.preventDefault();

    const payload = { ...form() };

    if (payload.type === 'xunji') {
      const schemaFields = xunjiSchema();
      if (schemaFields.length === 0) {
        loadXunJiSchema();
        toast.show('error', 'XUNJI Schema 尚未加载完成，请稍后重试');
        return;
      }

      const cfg = buildXunJiPayload();
      const errors = validateXunJiConfig(cfg, schemaFields);
      setXunjiErrors(errors);
      if (Object.keys(errors).length > 0) {
        toast.show('error', '请先修正 XUNJI 配置项');
        return;
      }

      payload.upload_interval = toInt(cfg.uploadIntervalMs, 5000);
      payload.config = JSON.stringify(cfg);
    } else {
      try {
        JSON.parse(payload.config || '{}');
      } catch {
        toast.show('error', '配置 JSON 格式错误');
        return;
      }
      setXunjiErrors({});
    }

    setSaving(true);
    const api = editing() ? putJSON(`/api/northbound/${editing()}`, payload) : postJSON('/api/northbound', payload);
    api.then(() => {
      toast.show('success', editing() ? '已更新' : '已创建');
      setForm(empty);
      setEditing(null);
      setShowModal(false);
      setXunjiConfig(normalizeXunJiConfig({}, xunjiSchema(), 5000));
      setXunjiErrors({});
      load();
    })
      .catch((err) => toast.show('error', err?.message || '操作失败'))
      .finally(() => setSaving(false));
  };

  const toggle = (id) => {
    postJSON(`/api/northbound/${id}/toggle`, {})
      .then(load)
      .catch(() => toast.show('error', '切换失败'));
  };

  const reload = (id) => {
    post(`/api/northbound/${id}/reload`)
      .then(() => {
        toast.show('success', '重载成功');
        load();
      })
      .catch(() => toast.show('error', '重载失败'));
  };

  const syncGatewayIdentity = () => {
    setSyncing(true);
    post('/api/gateway/northbound/sync-identity')
      .then((res) => {
        const data = unwrapData(res, {});
        const updated = data.updated?.length || 0;
        const failed = data.failed ? Object.keys(data.failed).length : 0;
        toast.show('success', `同步完成：更新 ${updated} 个，失败 ${failed} 个`);
        load();
      })
      .catch((err) => toast.show('error', err?.message || '同步失败'))
      .finally(() => setSyncing(false));
  };

  const remove = (id) => {
    if (!confirm('删除该配置？')) return;
    del(`/api/northbound/${id}`)
      .then(() => { toast.show('success', '已删除'); load(); })
      .catch(() => toast.show('error', '删除失败'));
  };

  const openCreate = () => {
    setForm(empty);
    setEditing(null);
    setShowModal(true);
    if (xunjiSchema().length === 0) {
      loadXunJiSchema(true);
    }
    setXunjiConfig(normalizeXunJiConfig({}, xunjiSchema(), 5000));
    setXunjiErrors({});
  };

  const edit = (item) => {
    const upload = toInt(item.upload_interval, 5000);
    const base = { name: item.name, type: item.type, upload_interval: upload, config: item.config, enabled: item.enabled };

    if (item.type === 'xunji') {
      if (xunjiSchema().length === 0) {
        loadXunJiSchema(true);
      }
      const parsed = safeParseJSON(item.config, {});
      const cfg = normalizeXunJiConfig(parsed, xunjiSchema(), upload);
      base.upload_interval = toInt(cfg.uploadIntervalMs, upload);
      base.config = JSON.stringify(cfg);
      setXunjiConfig(cfg);
    } else {
      setXunjiConfig(normalizeXunJiConfig({}, xunjiSchema(), upload));
    }

    setEditing(item.id);
    setForm(base);
    setShowModal(true);
    setXunjiErrors({});
  };

  const updateType = (nextType) => {
    const current = form();
    const next = { ...current, type: nextType };

    if (nextType === 'xunji') {
      if (xunjiSchema().length === 0) {
        loadXunJiSchema(true);
      }
      const parsed = safeParseJSON(current.config, {});
      const cfg = normalizeXunJiConfig(parsed, xunjiSchema(), current.upload_interval);
      next.upload_interval = toInt(cfg.uploadIntervalMs, current.upload_interval);
      next.config = JSON.stringify(cfg);
      setXunjiConfig(cfg);
    }

    setForm(next);
    setXunjiErrors({});
  };

  const updateXunJiField = (key, value, fieldType) => {
    setXunjiErrors((prev) => ({ ...prev, [key]: undefined }));

    let nextValue = value;
    if (fieldType === 'int') nextValue = toInt(value, 0);
    if (fieldType === 'bool') nextValue = toBool(value);

    setXunjiConfig((prev) => {
      const next = { ...prev, [key]: nextValue };
      if (key === 'uploadIntervalMs') {
        const uploadInterval = toInt(next.uploadIntervalMs, 5000);
        setForm((f) => ({ ...f, upload_interval: uploadInterval }));
      }
      return next;
    });
  };

  return (
    <div>
      <Card
        title="北向配置列表"
        extra={
          <div class="flex" style="gap:8px; align-items:center;">
            <Show when={xunjiSchemaError()}>
              <span style="font-size:12px; color:var(--danger);">XUNJI Schema 异常</span>
            </Show>
            <button class="btn" onClick={syncGatewayIdentity} disabled={syncing()}>
              {syncing() ? '同步中...' : '同步网关身份'}
            </button>
            <button class="btn btn-primary" onClick={openCreate}>
              新增配置
            </button>
          </div>
        }
      >
        {loading() ? (
          <div class="text-center" style="padding:48px; color:var(--text-muted);">
            <div class="loading-spinner" style="margin:0 auto 16px;"></div>
            <div>加载中...</div>
          </div>
        ) : (
          <CrudTable
            style="max-height:520px; overflow:auto;"
            loading={loading()}
            items={items()}
            emptyText="暂无配置"
            columns={[
              { key: 'id', title: 'ID' },
              { key: 'name', title: '名称' },
              {
                key: 'type',
                title: '类型',
                render: (n) => (
                  <span class="badge badge-info">{n.type.toUpperCase()}</span>
                ),
              },
              { key: 'upload_interval', title: '上传间隔(ms)' },
              {
                key: 'enabled',
                title: '状态',
                render: (n) => (
                  <span class={`badge ${n.enabled === 1 ? 'badge-running' : 'badge-stopped'}`}>
                    {n.enabled === 1 ? '启用' : '禁用'}
                  </span>
                ),
              },
              {
                key: 'runtime',
                title: '运行态',
                render: (n) => {
                  const rt = runtimeByName()[n.name] || n.runtime || {};
                  const registered = rt.registered ? '已注册' : '未注册';
                  const breaker = rt.breaker_state || 'closed';
                  return (
                    <div style="font-size:12px; color:var(--text-secondary); line-height:1.5;">
                      <div>{registered} / {rt.enabled ? '运行' : '停止'}</div>
                      <div>熔断: {breaker}</div>
                    </div>
                  );
                },
              },
            ]}
            renderActions={(n) => (
              <div class="flex" style="gap:8px;">
                <button class="btn" onClick={() => edit(n)}>编辑</button>
                <button class="btn" onClick={() => toggle(n.id)}>{n.enabled === 1 ? '禁用' : '启用'}</button>
                <button class="btn" onClick={() => reload(n.id)}>重载</button>
                <button class="btn btn-danger" onClick={() => remove(n.id)}>删除</button>
              </div>
            )}
          />
        )}
      </Card>

      <Show when={showModal()}>
        <div class="modal-backdrop" style="position:fixed; inset:0; background:rgba(0,0,0,0.45); display:flex; align-items:center; justify-content:center; z-index:1000; overflow:auto; padding:16px;">
          <div class="card" style="width:780px; max-width:94vw;">
            <div class="card-header">
              <h3 class="card-title">{editing() ? '编辑北向配置' : '新增北向配置'}</h3>
              <button class="btn btn-ghost" onClick={() => { setShowModal(false); setEditing(null); setForm(empty); setXunjiErrors({}); }} style="padding:4px 8px;">✕</button>
            </div>
            <form class="form" onSubmit={submit} style="padding:12px 16px 16px;">
              <div class="grid" style="grid-template-columns: 1fr 1fr; gap:12px;">
                <div class="form-group">
                  <label class="form-label">名称</label>
                  <input
                    class="form-input"
                    value={form().name}
                    onInput={(e) => setForm({ ...form(), name: e.target.value })}
                    placeholder="配置名称"
                    required
                  />
                </div>
                <div class="form-group">
                  <label class="form-label">类型</label>
                  <select
                    class="form-select"
                    value={form().type}
                    onChange={(e) => updateType(e.target.value)}
                  >
                    <option value="http">HTTP</option>
                    <option value="mqtt">MQTT</option>
                    <option value="xunji">寻迹</option>
                  </select>
                </div>
              </div>

              <Show
                when={form().type !== 'xunji'}
                fallback={(
                  <div class="form-group">
                    <label class="form-label">上传间隔 (ms)</label>
                    <input class="form-input" type="number" value={form().upload_interval} disabled />
                    <div class="form-hint">XUNJI 模式下由 Schema 字段 `uploadIntervalMs` 驱动，并同步到北向上传间隔。</div>
                  </div>
                )}
              >
                <div class="form-group">
                  <label class="form-label">上传间隔 (ms)</label>
                  <input
                    class="form-input"
                    type="number"
                    value={form().upload_interval}
                    onInput={(e) => setForm({ ...form(), upload_interval: +e.target.value })}
                    required
                  />
                </div>
              </Show>

              <Show
                when={form().type === 'xunji'}
                fallback={(
                  <div class="form-group">
                    <label class="form-label">配置 (JSON)</label>
                    <textarea
                      class="form-input"
                      rows={7}
                      value={form().config}
                      onInput={(e) => setForm({ ...form(), config: e.target.value })}
                      placeholder='{ "url": "http://...", "method": "POST" }'
                      style="font-family:monospace; font-size:13px;"
                    ></textarea>
                  </div>
                )}
              >
                <div class="card" style="margin-top:8px; padding:12px; background:var(--surface-1); border:1px solid var(--border-color);">
                  <div style="font-weight:600; margin-bottom:8px;">XUNJI Schema 配置（后端下发）</div>

                  <Show
                    when={xunjiSchema().length > 0}
                    fallback={(
                      <div class="form-hint" style="margin-bottom:8px; color:var(--danger);">
                        <div>{xunjiSchemaLoading() ? 'XUNJI Schema 加载中...' : (xunjiSchemaError() || 'XUNJI Schema 尚未加载')}</div>
                        <Show when={!xunjiSchemaLoading()}>
                          <button type="button" class="btn" style="margin-top:8px;" onClick={() => loadXunJiSchema()}>
                            重试加载 Schema
                          </button>
                        </Show>
                      </div>
                    )}
                  >
                    <div class="grid" style="grid-template-columns: 1fr 1fr; gap:10px 12px;">
                      <For each={xunjiSchema()}>{(field) => {
                        const value = xunjiConfig()[field.key];
                        const error = xunjiErrors()[field.key];
                        const isPassword = field.key.toLowerCase().includes('password');

                        return (
                          <div class="form-group" style="margin-bottom:0;">
                            <label class="form-label">
                              {field.label}
                              {field.required ? ' *' : ''}
                            </label>

                            <Show
                              when={field.type !== 'bool'}
                              fallback={(
                                <select
                                  class="form-select"
                                  value={value ? 'true' : 'false'}
                                  onChange={(e) => updateXunJiField(field.key, e.target.value, field.type)}
                                >
                                  <option value="false">false</option>
                                  <option value="true">true</option>
                                </select>
                              )}
                            >
                              <input
                                class="form-input"
                                type={isPassword ? 'password' : (field.type === 'int' ? 'number' : 'text')}
                                value={field.type === 'int' ? toInt(value, field.default ?? 0) : (value ?? '')}
                                onInput={(e) => updateXunJiField(field.key, e.target.value, field.type)}
                                placeholder={field.default !== '' ? `${field.default}` : ''}
                                required={field.required}
                              />
                            </Show>

                            <div class="form-hint">{field.description}</div>
                            <Show when={error}>
                              <div style="color:var(--danger); font-size:12px; margin-top:4px;">{error}</div>
                            </Show>
                          </div>
                        );
                      }}</For>
                    </div>
                  </Show>

                  <div class="form-group" style="margin-top:10px;">
                    <label class="form-label">生成配置预览 (JSON)</label>
                    <textarea
                      class="form-input"
                      rows={6}
                      value={JSON.stringify(buildXunJiPayload(), null, 2)}
                      readonly
                      style="font-family:monospace; font-size:12px;"
                    ></textarea>
                  </div>
                </div>
              </Show>

              <div class="flex" style={{ gap: '8px', justifyContent: 'flex-end', marginTop: '12px' }}>
                <button
                  type="button"
                  class="btn"
                  onClick={() => { setShowModal(false); setEditing(null); setForm(empty); setXunjiErrors({}); }}
                  disabled={saving()}
                >
                  取消
                </button>
                <button type="submit" class="btn btn-primary" disabled={saving()}>
                  {saving() ? '保存中...' : (editing() ? '保存' : '创建')}
                </button>
              </div>
            </form>
          </div>
        </div>
      </Show>
    </div>
  );
}
