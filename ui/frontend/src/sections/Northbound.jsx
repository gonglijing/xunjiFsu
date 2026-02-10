import { createSignal, createEffect, onMount, Show, For } from 'solid-js';
import api from '../api/services';
import { useToast } from '../components/Toast';
import Card from '../components/cards';
import CrudTable from '../components/CrudTable';
import LoadErrorHint from '../components/LoadErrorHint';
import { getErrorMessage } from '../api/errorMessages';
import { showErrorToast, withErrorToast } from '../utils/errors';
import { usePageLoader } from '../utils/pageLoader';
import { getNorthboundDefaultUploadIntervalMs } from '../utils/runtimeConfig';
import {
  NORTHBOUND_TYPE,
  getNorthboundTypeLabel,
  isIThingsType,
  isPandaXType,
  isSchemaDrivenType,
  normalizeNorthboundType,
} from '../utils/northboundType';

const DEFAULT_NORTHBOUND_UPLOAD_INTERVAL_MS = getNorthboundDefaultUploadIntervalMs();
const empty = {
  name: '',
  type: 'mqtt',
  upload_interval: DEFAULT_NORTHBOUND_UPLOAD_INTERVAL_MS,
  config: '{}',
  enabled: 1,
};

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

function normalizeConfig(raw, schemaFields, uploadIntervalFallback = DEFAULT_NORTHBOUND_UPLOAD_INTERVAL_MS) {
  const src = raw && typeof raw === 'object' ? raw : {};
  const out = { ...buildSchemaDefaults(schemaFields) };

  for (const field of schemaFields || []) {
    const value = src[field.key];
    if (value === undefined || value === null || value === '') continue;

    if (field.type === 'int') out[field.key] = toInt(value, field.default ?? 0);
    else if (field.type === 'bool') out[field.key] = toBool(value);
    else out[field.key] = `${value}`;
  }

  // 处理上传周期
  if (hasSchemaField(schemaFields, 'uploadIntervalMs') && toInt(out.uploadIntervalMs, 0) <= 0) {
    const fallback = toInt(
      src.uploadIntervalMs,
      toInt(src.reportIntervalMs, toInt(uploadIntervalFallback, DEFAULT_NORTHBOUND_UPLOAD_INTERVAL_MS)),
    );
    out.uploadIntervalMs = fallback > 0 ? fallback : DEFAULT_NORTHBOUND_UPLOAD_INTERVAL_MS;
  }

  // QOS 校验
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

function validateConfig(config, schemaFields) {
  const errors = {};

  for (const field of schemaFields || []) {
    if (!field.required) continue;
    const value = config[field.key];
    if (field.type === 'string') {
      if (!`${value ?? ''}`.trim()) errors[field.key] = `${field.label} 为必填`;
      continue;
    }
    if (field.type === 'int') {
      if (`${value ?? ''}`.trim() === '' || Number.isNaN(toInt(value, Number.NaN))) {
        errors[field.key] = `${field.label} 为必填`;
      }
      continue;
    }
    if (field.type === 'bool' && value === undefined) {
      errors[field.key] = `${field.label} 为必填`;
    }
  }

  // QOS 校验
  if (hasSchemaField(schemaFields, 'qos')) {
    const qos = toInt(config.qos, -1);
    if (qos < 0 || qos > 2) {
      errors.qos = 'QOS 必须在 0~2 之间';
    }
  }

  if (hasSchemaField(schemaFields, 'gatewayMode') && toBool(config.gatewayMode) !== true) {
    errors.gatewayMode = '网关模式仅支持 true';
  }

  // 上传周期校验
  if (hasSchemaField(schemaFields, 'uploadIntervalMs') && toInt(config.uploadIntervalMs, 0) <= 0) {
    errors.uploadIntervalMs = '上报周期必须大于 0';
  }

  // 报警相关校验
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
  const showLoadError = withErrorToast(toast, '加载北向配置失败');
  const showSaveError = withErrorToast(toast, '保存失败');
  const {
    loading,
    error: loadError,
    setError: setLoadError,
    run: runNorthboundLoad,
  } = usePageLoader(async () => {
    const [configs, status] = await Promise.all([
      api.northbound.listNorthboundConfigs(),
      api.northbound.listNorthboundStatus(),
    ]);
    setItems(configs || []);
    setRuntime(status || []);
  }, {
    onError: showLoadError,
    errorMessage: '加载北向配置失败',
  });
  const [form, setForm] = createSignal(empty);
  const [editing, setEditing] = createSignal(null);
  const [showModal, setShowModal] = createSignal(false);
  const [saving, setSaving] = createSignal(false);

  // Schema 状态
  const [schema, setSchema] = createSignal([]);
  const [schemaLoading, setSchemaLoading] = createSignal(false);
  const [schemaError, setSchemaError] = createSignal('');
  const [config, setConfig] = createSignal({});
  const [configErrors, setConfigErrors] = createSignal({});

  const runtimeByName = () => {
    const map = {};
    for (const item of runtime()) {
      if (item?.name) map[item.name] = item;
    }
    return map;
  };

  const buildPayload = () => {
    return normalizeConfig(config(), schema(), form().upload_interval);
  };

  const loadSchema = (nbType, silent = false) => {
    setSchemaLoading(true);
    return api.northbound.getNorthboundSchema(nbType)
      .then((schemaData) => {
        const fields = Array.isArray(schemaData?.fields) ? schemaData.fields : [];
        setSchema(fields);

        if (fields.length === 0) {
          setSchemaError(`${nbType.toUpperCase()} Schema 为空，请检查后端配置`);
          if (!silent) {
            toast.show('error', `${nbType.toUpperCase()} Schema 为空，请检查后端配置`);
          }
          return;
        }

        setSchemaError('');
        setConfig((prev) => normalizeConfig(prev, fields, form().upload_interval));
      })
      .catch((err) => {
        const message = getErrorMessage(err, `加载 ${nbType.toUpperCase()} Schema 失败`);
        setSchema([]);
        setSchemaError(message);
        if (!silent) {
          toast.show('error', message);
        }
      })
      .finally(() => setSchemaLoading(false));
  };

  const load = () => {
    setLoadError('');
    runNorthboundLoad();
  };

  onMount(() => {
    load();
    loadSchema(NORTHBOUND_TYPE.PANDAX, true);
  });

  const submit = (e) => {
    e.preventDefault();
    const payload = { ...form() };

    // schema 驱动类型使用动态表单
    if (isSchemaDrivenType(payload.type) && schema().length > 0) {
      const cfg = buildPayload();
      const errors = validateConfig(cfg, schema());
      setConfigErrors(errors);
      if (Object.keys(errors).length > 0) {
        toast.show('error', '请先修正配置项');
        return;
      }

      // 从 schema 配置同步上传周期
      if (hasSchemaField(schema(), 'uploadIntervalMs')) {
        payload.upload_interval = toInt(cfg.uploadIntervalMs, DEFAULT_NORTHBOUND_UPLOAD_INTERVAL_MS);
      }
      payload.config = JSON.stringify(cfg);

      if (isPandaXType(payload.type) || isIThingsType(payload.type)) {
        if (payload.server_url === undefined || !`${payload.server_url || ''}`.trim()) {
          payload.server_url = cfg.serverUrl || cfg.broker || '';
        }
        if (payload.username === undefined || !`${payload.username || ''}`.trim()) {
          payload.username = cfg.username || '';
        }
      }
    } else {
      // 无 schema，使用 JSON 编辑
      try {
        JSON.parse(payload.config || '{}');
      } catch {
        toast.show('error', '配置 JSON 格式错误');
        return;
      }
      setConfigErrors({});
    }

    setSaving(true);
    const request = editing()
      ? api.northbound.updateNorthboundConfig(editing(), payload)
      : api.northbound.createNorthboundConfig(payload);
    request.then(() => {
      toast.show('success', editing() ? '已更新' : '已创建');
      resetForm();
      load();
    })
      .catch(showSaveError)
      .finally(() => setSaving(false));
  };

  const resetForm = () => {
    setForm(empty);
    setEditing(null);
    setShowModal(false);
    setConfig(normalizeConfig({}, [], DEFAULT_NORTHBOUND_UPLOAD_INTERVAL_MS));
    setConfigErrors({});
  };

  const toggle = (id) => {
    api.northbound.toggleNorthboundConfig(id)
      .then(load)
      .catch((err) => showErrorToast(toast, err, '切换失败'));
  };

  const reload = (id) => {
    api.northbound.reloadNorthboundConfig(id)
      .then(() => {
        toast.show('success', '重载成功');
        load();
      })
      .catch((err) => showErrorToast(toast, err, '重载失败'));
  };

  const remove = (id) => {
    if (!confirm('删除该配置？')) return;
    api.northbound.deleteNorthboundConfig(id)
      .then(() => { toast.show('success', '已删除'); load(); })
      .catch((err) => showErrorToast(toast, err, '删除失败'));
  };

  const openCreate = () => {
    setForm(empty);
    setEditing(null);
    setShowModal(true);
    setConfig(normalizeConfig({}, [], DEFAULT_NORTHBOUND_UPLOAD_INTERVAL_MS));
    setConfigErrors({});
    // 确保加载了对应类型的 schema
    const currentType = form().type || NORTHBOUND_TYPE.MQTT;
    if (isSchemaDrivenType(currentType) && schema().length === 0) {
      loadSchema(currentType, true);
    }
  };

  const edit = (item) => {
    const upload = toInt(item.upload_interval, DEFAULT_NORTHBOUND_UPLOAD_INTERVAL_MS);
    const normalizedType = normalizeNorthboundType(item.type);
    const base = {
      name: item.name,
      type: normalizedType,
      upload_interval: upload,
      config: item.config,
      enabled: item.enabled,
    };

    const parsed = safeParseJSON(item.config, {});

    if (isSchemaDrivenType(normalizedType)) {
      loadSchema(normalizedType, true).then(() => {
        const cfg = normalizeConfig(parsed, schema(), upload);
        if (hasSchemaField(schema(), 'uploadIntervalMs')) {
          base.upload_interval = toInt(cfg.uploadIntervalMs, upload);
        }
        base.config = JSON.stringify(cfg);
        setConfig(cfg);
      });
    } else {
      setConfig(normalizeConfig({}, [], upload));
    }

    setEditing(item.id);
    setForm(base);
    setShowModal(true);
    setConfigErrors({});
  };

  const updateType = (nextType) => {
    nextType = normalizeNorthboundType(nextType);
    const current = form();
    const next = { ...current, type: nextType };

    if (isSchemaDrivenType(nextType)) {
      loadSchema(nextType, true).then(() => {
        const parsed = safeParseJSON(current.config, {});
        const cfg = normalizeConfig(parsed, schema(), current.upload_interval);

        if (hasSchemaField(schema(), 'uploadIntervalMs')) {
          next.upload_interval = toInt(cfg.uploadIntervalMs, current.upload_interval);
        }
        next.config = JSON.stringify(cfg);
        setConfig(cfg);
      });
    } else {
      setSchema([]);
      setSchemaError('');
      setConfig(normalizeConfig({}, [], current.upload_interval));
    }

    setForm(next);
    setConfigErrors({});
  };

  const updateField = (key, value, fieldType) => {
    setConfigErrors((prev) => ({ ...prev, [key]: undefined }));

    let nextValue = value;
    if (fieldType === 'int') nextValue = toInt(value, 0);
    if (fieldType === 'bool') nextValue = toBool(value);

    setConfig((prev) => {
      const next = { ...prev, [key]: nextValue };
      // 同步上传周期
      if (key === 'uploadIntervalMs') {
        const uploadInterval = toInt(next.uploadIntervalMs, DEFAULT_NORTHBOUND_UPLOAD_INTERVAL_MS);
        setForm((f) => ({ ...f, upload_interval: uploadInterval }));
      }
      return next;
    });
  };

  const getTypeLabel = () => {
    return getNorthboundTypeLabel(form().type);
  };

  const getSchemaTitle = () => {
    const type = normalizeNorthboundType(form().type);
    if (type === NORTHBOUND_TYPE.SAGOO) return 'Sagoo Schema 配置';
    if (type === NORTHBOUND_TYPE.PANDAX) return 'PandaX Schema 配置';
    if (type === NORTHBOUND_TYPE.ITHINGS) return 'iThings Schema 配置';
    if (type === NORTHBOUND_TYPE.MQTT) return 'MQTT Schema 配置';
    return '配置';
  };

  return (
    <div>
      <Card
        title="北向配置列表"
        extra={
          <div class="flex" style="gap:8px; align-items:center;">
            <Show when={schemaError()}>
              <span style="font-size:12px; color:var(--danger);">Schema 异常</span>
            </Show>
            <button class="btn btn-primary" onClick={openCreate}>
              新增配置
            </button>
          </div>
        }
      >
        <LoadErrorHint error={loadError()} onRetry={load} />
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
                  const connected = !!rt.connected;
                  return (
                    <div style="font-size:12px; color:var(--text-secondary); line-height:1.5;">
                      <div>{registered} / {rt.enabled ? '运行' : '停止'}</div>
                      <div>
                        北向连接: {' '}
                        <span style={`color:${connected ? 'var(--success)' : 'var(--danger)'}; font-weight:600;`}>
                          {connected ? '连接中' : '已断开'}
                        </span>
                      </div>
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
              <button class="btn btn-ghost btn-no-icon btn-only-icon" onClick={resetForm} style="padding:4px 8px;">✕</button>
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
                    <option value={NORTHBOUND_TYPE.MQTT}>MQTT</option>
                    <option value={NORTHBOUND_TYPE.PANDAX}>PandaX</option>
                    <option value={NORTHBOUND_TYPE.ITHINGS}>iThings</option>
                    <option value={NORTHBOUND_TYPE.SAGOO}>Sagoo</option>
                  </select>
                </div>
              </div>

              {/* Schema 驱动表单或 JSON 编辑 */}
              <Show
                when={schema().length > 0 && isSchemaDrivenType(form().type)}
                fallback={
                  <div class="form-group">
                    <label class="form-label">配置 (JSON)</label>
                    <textarea
                      class="form-input"
                      rows={7}
                      value={form().config}
                      onInput={(e) => setForm({ ...form(), config: e.target.value })}
                      placeholder='{ "broker": "mqtt://...", "topic": "/device/data" }'
                      style="font-family:monospace; font-size:13px;"
                    ></textarea>
                  </div>
                }
              >
                <div class="card" style="margin-top:8px; padding:12px; background:var(--surface-1); border:1px solid var(--border-color);">
                  <div style="font-weight:600; margin-bottom:8px;">{getSchemaTitle()}（后端下发）</div>

                  <Show
                    when={!schemaLoading()}
                    fallback={
                      <div class="form-hint" style="margin-bottom:8px;">
                        <div>Schema 加载中...</div>
                      </div>
                    }
                  >
                    <Show
                      when={schema().length > 0}
                      fallback={
                        <div class="form-hint" style="margin-bottom:8px; color:var(--danger);">
                          <div>{schemaError() || 'Schema 加载失败'}</div>
                          <button type="button" class="btn" style="margin-top:8px;" onClick={() => loadSchema(form().type)}>
                            重试
                          </button>
                        </div>
                      }
                    >
                      <div class="grid" style="grid-template-columns: 1fr 1fr; gap:10px 12px;">
                        <For each={schema()}>{(field) => {
                          const value = config()[field.key];
                          const error = configErrors()[field.key];
                          const isPassword = field.key.toLowerCase().includes('password');

                          return (
                            <div class="form-group" style="margin-bottom:0;">
                              <label class="form-label">
                                {field.label}
                                {field.required ? ' *' : ''}
                              </label>

                              <Show
                                when={field.type !== 'bool'}
                                fallback={
                                  <select
                                    class="form-select"
                                    value={value ? 'true' : 'false'}
                                    onChange={(e) => updateField(field.key, e.target.value, field.type)}
                                  >
                                    <option value="false">false</option>
                                    <option value="true">true</option>
                                  </select>
                                }
                              >
                                <input
                                  class="form-input"
                                  type={isPassword ? 'password' : (field.type === 'int' ? 'number' : 'text')}
                                  value={field.type === 'int' ? toInt(value, field.default ?? 0) : (value ?? '')}
                                  onInput={(e) => updateField(field.key, e.target.value, field.type)}
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
                  </Show>

                  <div class="form-group" style="margin-top:10px;">
                    <label class="form-label">生成配置预览 (JSON)</label>
                    <textarea
                      class="form-input"
                      rows={6}
                      value={JSON.stringify(buildPayload(), null, 2)}
                      readonly
                      style="font-family:monospace; font-size:12px;"
                    ></textarea>
                  </div>
                </div>
              </Show>

              <div class="flex" style={{ gap: '8px', justifyContent: 'flex-end', marginTop: '12px' }}>
                <button type="button" class="btn" onClick={resetForm} disabled={saving()}>
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
