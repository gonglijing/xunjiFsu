import { createSignal, createEffect, For, Show } from 'solid-js';
import Card from '../components/cards';
import { alarmsAPI, dataAPI, devicesAPI, gatewayAPI, northboundAPI, resourcesAPI } from '../api/services';
import DeviceDetailDrawer from '../components/DeviceDetailDrawer';

function Topology() {
  const [gateway, setGateway] = createSignal(null);
  const [resources, setResources] = createSignal([]);
  const [devices, setDevices] = createSignal([]);
  const [northbounds, setNorthbounds] = createSignal([]);
  const [loading, setLoading] = createSignal(true);

  const [detailVisible, setDetailVisible] = createSignal(false);
  const [detailDevice, setDetailDevice] = createSignal(null);
  const [detailCache, setDetailCache] = createSignal([]);
  const [detailAlarms, setDetailAlarms] = createSignal([]);
  const [detailLoading, setDetailLoading] = createSignal(false);

  const load = async () => {
    setLoading(true);
    try {
      const [gwRes, resRes, devRes, nbRes] = await Promise.all([
        gatewayAPI.getGatewayConfig(),
        resourcesAPI.listResources(),
        devicesAPI.listDevices(),
        northboundAPI.listNorthboundConfigs(),
      ]);
      setGateway(gwRes || null);
      setResources(resRes || []);
      setDevices(devRes || []);
      setNorthbounds(nbRes || []);
    } finally {
      setLoading(false);
    }
  };

  createEffect(load);

  const devicesByResource = () => {
    const map = {};
    for (const d of devices()) {
      const key = d.resource_id ?? 0;
      if (!map[key]) map[key] = [];
      map[key].push(d);
    }
    return map;
  };

  const devMap = () => devicesByResource();

  const openDetail = async (device) => {
    setDetailDevice(device);
    setDetailVisible(true);
    setDetailLoading(true);
    try {
      const [cacheRes, alarmsRes] = await Promise.all([
        dataAPI.getDataCacheByDevice(device.id),
        alarmsAPI.listAlarms(),
      ]);
      const cacheVal = Array.isArray(cacheRes) ? cacheRes : cacheRes?.data || [];
      const allAlarms = Array.isArray(alarmsRes) ? alarmsRes : alarmsRes?.data || [];
      setDetailCache(cacheVal);
      setDetailAlarms(allAlarms.filter((a) => String(a.device_id) === String(device.id)));
    } finally {
      setDetailLoading(false);
    }
  };

  const closeDetail = () => {
    setDetailVisible(false);
    setDetailDevice(null);
    setDetailCache([]);
    setDetailAlarms([]);
  };

  return (
    <>
      <Card title="网关拓扑视图">
        <Show when={!loading()} fallback={<div class="text-muted">加载中...</div>}>
          <div class="grid" style="grid-template-columns: 2fr 1fr 2fr; gap:32px; align-items:flex-start;">
            {/* 左：资源 + 设备 */}
            <div>
              <h4 class="text-xs text-muted" style="margin-bottom:8px;">采集侧</h4>
              <div class="grid" style="gap:12px;">
                <For each={resources()}>
                  {(res) => (
                    <div class="card" style="padding:12px;">
                      <div class="text-xs text-muted">资源</div>
                      <div><strong>{res.name}</strong> ({res.type})</div>
                      <div class="text-xs text-muted">{res.path}</div>
                      <div style="margin-top:8px; font-size:12px;">
                        <For each={devMap()[res.id] || []}>
                          {(d) => (
                            <div
                              style="padding:2px 0; cursor:pointer;"
                              onClick={() => openDetail(d)}
                            >
                              ▸ {d.name} <span class="text-muted text-xs">#{d.id}</span>
                            </div>
                          )}
                        </For>
                        <Show when={!(devMap()[res.id] || []).length}>
                          <div class="text-muted text-xs">暂无挂载设备</div>
                        </Show>
                      </div>
                    </div>
                  )}
                </For>
                {/* 无资源设备 */}
                <Show when={(devMap()[0] || []).length}>
                  <div class="card" style="padding:12px;">
                    <div class="text-xs text-muted">未绑定资源设备</div>
                    <For each={devMap()[0] || []}>
                      {(d) => (
                        <div
                          style="padding:2px 0; cursor:pointer;"
                          onClick={() => openDetail(d)}
                        >
                          ▸ {d.name} <span class="text-muted text-xs">#{d.id}</span>
                        </div>
                      )}
                    </For>
                  </div>
                </Show>
              </div>
            </div>

            {/* 中：网关 */}
            <div style="display:flex; flex-direction:column; align-items:center; gap:16px;">
              <div
                style="
                  width:140px;height:140px;border-radius:24px;
                  display:flex;flex-direction:column;align-items:center;justify-content:center;
                  background:radial-gradient(circle at 0% 0%,#22d3ee 0%,#3b82f6 40%,#020617 100%);
                  box-shadow:0 0 28px rgba(59,130,246,0.6);
                "
              >
                <div style="font-size:14px;opacity:0.8;">网关</div>
                <div style="font-weight:600;margin-top:4px;">
                  {gateway()?.name || 'HuShu 网关'}
                </div>
                <div class="text-xs text-muted" style="margin-top:4px;">
                  {gateway()?.id ? `ID: ${gateway().id}` : ''}
                </div>
              </div>
              <div class="text-xs text-muted">
                左：资源/设备 · 右：北向通道
              </div>
            </div>

            {/* 右：北向 */}
            <div>
              <h4 class="text-xs text-muted" style="margin-bottom:8px;">北向通道</h4>
              <div class="grid" style="gap:12px;">
                <For each={northbounds()}>
                  {(nb) => (
                    <div class="card" style="padding:12px;">
                      <div class="text-xs text-muted">北向</div>
                      <div><strong>{nb.name}</strong></div>
                      <div style="margin-top:4px;font-size:12px;">
                        类型：
                        <span class="badge badge-info" style="margin-left:4px;">
                          {nb.type?.toUpperCase()}
                        </span>
                      </div>
                      <div class="text-xs text-muted" style="margin-top:4px;">
                        上传间隔：{nb.upload_interval} ms
                      </div>
                      <div style="margin-top:6px;">
                        <span class={`badge ${nb.enabled === 1 ? 'badge-running' : 'badge-stopped'}`}>
                          {nb.enabled === 1 ? '启用' : '禁用'}
                        </span>
                      </div>
                    </div>
                  )}
                </For>
                <Show when={!northbounds().length}>
                  <div class="card" style="padding:12px;">
                    <div class="text-muted text-xs">暂无北向配置</div>
                  </div>
                </Show>
              </div>
            </div>
          </div>
        </Show>
      </Card>

      <DeviceDetailDrawer
        visible={detailVisible()}
        device={detailDevice}
        cache={detailCache}
        alarms={detailAlarms}
        loading={detailLoading}
        onClose={closeDetail}
      />
    </>
  );
}

export default Topology;
