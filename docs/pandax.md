# FSU ↔ PandaX（MQTT）网关/子设备遥测与同步说明

## 1. 目标与范围

本文档说明 FSU 与 PandaX 在 **仅使用 MQTT** 的前提下，如何实现以下能力：

- 网关模式上报子设备遥测；
- 手动触发“同步设备”，将 FSU 子设备信息与遥测物模型同步到 PandaX；
- PandaX 自动创建子设备与补齐遥测物模型字段。

> 范围限制：仅对应网关与子设备遥测，不涉及 API 对接、命令编排、资产管理等。

---

## 2. 总体设计

### 2.1 触发策略

- **不在启动时触发同步**；
- 仅在用户手动点击 FSU 北向配置中的 **“同步设备”** 按钮时触发。

### 2.2 关键通道

- 实时遥测：`v1/gateway/telemetry`
- 同步注册：`v1/gateway/register/telemetry`

### 2.3 角色分工

- FSU：负责组织子设备清单 + 最新遥测字段并发布到注册 Topic；
- PandaX：消费注册 Topic，在网关已认证前提下执行子设备归属校验/自动创建，并自动创建 telemetry 物模型字段。

---

## 3. FSU 侧实现

### 3.1 按钮与接口

- 前端按钮：PandaX 北向行显示 **同步设备**；
- 后端接口：`POST /api/northbound/{id}/sync-devices`

说明：

- PandaX 类型点击“同步设备”调用上述接口；
- 其他北向仍保留“重载”。

### 3.2 适配器行为

PandaX 适配器新增 `SyncDevices()`：

1. 读取设备清单（`devices`）；
2. 读取每个设备最新遥测（`GetAllDevicesLatestData`）；
3. 组装注册报文并发布到 `gatewayRegisterTopic`（默认 `v1/gateway/register/telemetry`）。

同步前增加字段完整性预检查：

- 以 `productKey` 维度检查本次同步数据；
- 若某个产品下所有子设备都没有遥测字段，则 FSU 直接拒绝本次同步并返回错误；
- 目的是避免 PandaX 因“无模型且无字段”而跳过该产品子设备创建。

驱动 productKey 透传：

- FSU 驱动执行结果支持返回 `productKey`（兼容 `product_key`）；
- 采集链路会优先使用驱动返回的 `productKey`，并回写到设备 `product_key` 字段；
- 按“一个驱动固定一个 productKey”处理：同一驱动首次识别后会缓存，后续若回包不一致则以缓存值为准；
- 后续“同步设备”将直接使用回写后的产品标识参与同步。
- 若设备 `product_key` 为空或与驱动固定值不一致，`SyncDevices` 会按 `driver_id -> 驱动 wasm version()` 提取 `productKey`，并在同步前回写修正。

### 3.3 配置项

FSU 对 PandaX 配置分两层：

- **新增表单 schema（`GET /api/northbound/schema?type=pandax`）**：仅暴露 4 个创建必填项：`serverUrl`、`username`、`password`、`qos`；
- **运行时完整配置（`config` JSON）**：支持高级参数，如 `gatewayTelemetryTopic`、`gatewayRegisterTopic`、`subDeviceTokenMode`、`rpcRequestTopic`、`rpcResponseTopic`、队列/批量参数等。

当前关键规则：

- `gatewayMode` 必须为 `true`（适配器会拒绝 `false`）；
- `gatewayRegisterTopic` 默认 `v1/gateway/register/telemetry`，兼容别名 `registerTopic`；
- `rpcRequestTopic` 默认 `v1/devices/me/rpc/request`，自动订阅基路径与 `.../+`；
- `uploadIntervalMs` 默认 5000ms，`alarmFlushIntervalMs` 默认 2000ms，`alarmBatchSize` 默认 20，队列默认 1000。

### 3.4 注册报文格式（FSU -> PandaX）

当前推荐格式：

```json
{
  "ts": 1739433600000,
  "subDevices": [
    {
      "productKey": "productA",
      "deviceName": "pump01",
      "ts": 1739433600000,
      "values": {
        "temperature": 23.5,
        "running": true
      }
    }
  ]
}
```

说明：

- 子设备标识由 `productKey + deviceName`（缺省回退 `deviceKey`）在 PandaX 侧推导；
- 统一使用 `values` 承载遥测字段；
- 若设备暂无遥测，`values` 可为空对象，但会影响物模型自动补齐效果。

---

## 4. PandaX 侧实现

### 4.1 MQTT 识别

`v1/gateway/register/telemetry` 已加入网关消息路由。

### 4.2 注册处理逻辑

PandaX 收到注册消息后：

1. 解析 `subDevices`（也兼容旧 map 格式）；
2. 读取子设备 `productKey/productId`，先检查该产品是否已存在 telemetry 物模型；
3. 若不存在：先按上报字段创建 telemetry 物模型；
4. 在网关认证上下文内校验子设备归属（`parent_id=当前网关` 且 `deviceType=gatewayS`）；
   - 子设备不存在则自动创建并绑定到当前网关；
5. 若该产品模型已存在，则跳过模型创建，仅执行子设备创建/绑定。

补充：

- 若该产品尚无模型且本次同步未携带字段，PandaX 将跳过该子设备创建（避免出现“无模型先创建设备”）。

### 4.3 兼容策略

- 支持 `subDevices` 数组模式；
- 支持旧式 `{ "token": {"values": {...}} }` map 模式；
- 自动忽略 `__system__` 等系统项。

### 4.4 安全边界（当前实现）

- 网关 MQTT 连接必须先通过 `username=gatewayToken` 认证；
- 子设备不再做独立认证（不再依赖 `SubAuth`）；
- 子设备上行仅在“归属于当前网关”时被接收（`parent_id` 绑定校验）；
- 不满足归属关系的子设备数据将被拒绝并记录日志。

### 4.5 实时上行处理约束

- `v1/gateway/telemetry` 上行时，PandaX 仅接收“已绑定当前网关”的子设备数据；
- 对于未绑定、类型错误（非 `gatewayS`）或产品不一致的子设备，实时数据会被拒绝；
- 子设备首次创建建议通过 FSU **同步设备**（`v1/gateway/register/telemetry`）完成，避免实时上报被丢弃。

---

## 5. 联调步骤（建议）

### 5.1 前置校验

- PandaX 中存在目标产品，且 FSU 设备 `product_key` 与 PandaX `productId` 对齐；
- FSU PandaX 北向已启用并可连通；
- FSU 子设备已产生至少一次遥测（用于补齐字段）。

### 5.2 执行

1. 在 FSU 页面点击 PandaX 行的 **同步设备**；
2. 观察 FSU 日志出现：`SyncDevices: 已发布同步消息`；
3. 观察 PandaX 日志无注册解析错误。

### 5.3 结果验证

- 子设备：`GET /device/list?deviceType=gatewayS&pageNum=1&pageSize=50`
- 物模型：`GET /device/template/list?pid=<productId>&classify=telemetry&pageNum=1&pageSize=200`
- 遥测状态：`GET /device/{id}/status?classify=telemetry`

---

## 6. 常见问题

### Q1：为什么同步后设备有了，但模型字段不全？

原因通常是该设备最近遥测字段不足或为空。同步逻辑按最新遥测字段补齐模型。

### Q2：为什么子设备会被拒绝？

优先检查：

- 当前 MQTT 连接对应网关是否认证通过；
- 子设备是否绑定到当前网关（`parent_id` 一致）；
- 子设备类型是否为 `gatewayS`，且 `productId` 与上报一致。

### Q3：可以自动定时同步吗？

当前设计为手动触发，避免启动抖动和无效写入。如需定时任务，可后续在 FSU 北向层增加调度开关。

---

## 7. 变更要点（摘要）

- FSU：新增 `sync-devices` 接口与 PandaX `SyncDevices()`，仅手动触发；
- FSU：PandaX 行按钮文案由“重载”改为“同步设备”；
- PandaX：新增 `v1/gateway/register/telemetry` 消费分支，按“网关认证 + 子设备归属校验”自动建子设备/绑定关系 + 自动建 telemetry 物模型。


---

## 8. 端到端演练（含 MQTT 报文校验）

### 8.1 演练目标

验证以下链路在一次“手动同步设备”中全部成立：

1. FSU 点击 **同步设备** 后才触发注册同步（启动时不触发）；
2. FSU 发送的 `subDevices[*].productKey` 使用驱动固定 `productKey`（必要时自动回写设备表）；
3. PandaX 对每个子设备执行“先模型后设备 + 归属校验”的处理：
   - 有同 `productKey` 遥测模型：仅创建设备/绑定；
   - 无同 `productKey` 遥测模型：先创建模型，再创建设备/绑定；
   - 不属于当前网关的子设备：拒绝处理并记录日志。

### 8.2 前置准备

- FSU：
  - PandaX 北向已启用且连接正常；
  - 子设备已绑定驱动（`driver_id` 有效）；
  - 子设备至少有一条最新遥测（用于模型字段提取）。
- PandaX：
  - 网关 MQTT 接入正常；
  - 允许网关 Topic `v1/gateway/register/telemetry`。

### 8.3 执行步骤

1. 在 FSU 前端进入北向配置，找到 PandaX 行；
2. 点击 **同步设备**；
3. 观察 FSU 日志：出现 `SyncDevices: 已发布同步消息`；
4. 观察 PandaX 日志：无注册解析错误、无模型创建异常。

### 8.4 MQTT 抓包与报文校验

可在 PandaX MQTT Broker 上临时订阅（示例）：

```bash
mosquitto_sub -h <broker_host> -p <broker_port> -u <username> -P <password> \
  -t 'v1/gateway/register/telemetry' -v
```

收到报文后重点校验：

- Topic 必须是 `v1/gateway/register/telemetry`；
- 根字段必须包含：`ts`、`subDevices`；
- `subDevices` 必须是数组，且每项包含：`productKey`、`deviceName`、`ts`、`values`；
- `subDevices[*].productKey` 应与该设备驱动固定 `productKey` 一致；
- `values` 不能为空对象（至少一个可用字段），否则 PandaX 可能跳过模型创建。

建议同时对照 FSU 数据库 `devices.product_key`：同步后应已被修正为驱动固定值。

### 8.5 PandaX 结果验收

按 `productKey + deviceKey` 维度验收：

1. 子设备存在：`GET /device/list?deviceType=gatewayS&pageNum=1&pageSize=200`
2. 遥测模型存在：`GET /device/template/list?pid=<productId>&classify=telemetry&pageNum=1&pageSize=200`
3. 字段已补齐：模型字段集合包含本次 `subDevices[*].values` 的 key。

### 8.6 失败场景定位

- 现象：点击同步后无注册报文
  - 排查：北向是否启用、MQTT 是否连接、`gatewayRegisterTopic` 是否被改错。
- 现象：子设备创建了但模型不全
  - 排查：`values` 字段是否缺失或值为空、该设备最近遥测是否不足。
- 现象：`productKey` 不正确
  - 排查：驱动 `version()` 是否返回固定 `productKey`；设备是否绑定了正确 `driver_id`；FSU 日志是否有 productKey 回写失败。
- 现象：PandaX 跳过创建
  - 排查：该 `productKey` 下是否“全部子设备无字段”，会触发 FSU 预检查拒绝。
- 现象：部分子设备实时数据被丢弃
  - 排查：子设备是否已绑定当前网关（`parent_id`），设备类型是否为 `gatewayS`。

---

## 9. 现场联调记录模板

> 用法：每次联调复制本节内容，填写一次记录，便于问题追踪和回归比对。

### 9.1 基本信息

- 联调日期：`YYYY-MM-DD`
- 联调时间段：`HH:mm - HH:mm`
- 环境：`测试 / 预发 / 生产`
- FSU 版本：`<git commit / tag>`
- PandaX 版本：`<git commit / tag>`
- 操作人：`<姓名>`
- 记录人：`<姓名>`

### 9.2 联调对象

- 网关标识：`<gateway token / device key>`
- 同步 Topic：`v1/gateway/register/telemetry`
- 遥测 Topic：`v1/gateway/telemetry`
- 子设备清单（productKey + deviceKey）：
  - `pk_1 / dk_1`
  - `pk_2 / dk_2`

### 9.3 执行步骤检查

- [ ] FSU PandaX 北向已启用且 MQTT 已连接
- [ ] 子设备存在最新遥测（至少 1 条）
- [ ] 在 FSU 页面点击 **同步设备**
- [ ] FSU 日志出现 `SyncDevices: 已发布同步消息`
- [ ] Broker 抓到 `v1/gateway/register/telemetry` 报文
- [ ] PandaX 侧无注册解析报错
- [ ] PandaX 侧子设备创建/绑定成功（无需子设备独立认证）
- [ ] PandaX 侧遥测物模型创建/补齐成功

### 9.4 MQTT 报文校验记录

- 抓包命令：

```bash
mosquitto_sub -h <broker_host> -p <broker_port> -u <username> -P <password> \
  -t 'v1/gateway/register/telemetry' -v
```

- 最近一次报文摘要：
  - `ts`：`<value>`
  - `subDevices.count`：`<value>`
  - `subDevices[0].productKey`：`<value>`
  - `subDevices[0].values.keys`：`<k1,k2,...>`

- 关键校验结果：
  - [ ] Topic 正确
  - [ ] 报文结构完整（ts/subDevices）
  - [ ] 子设备 `productKey` 与驱动固定值一致
  - [ ] `values` 非空且字段符合预期

### 9.5 PandaX 结果核验

- 子设备查询：`GET /device/list?deviceType=gatewayS&pageNum=1&pageSize=200`
  - 结果：`通过 / 不通过`
- 模型查询：`GET /device/template/list?pid=<productId>&classify=telemetry&pageNum=1&pageSize=200`
  - 结果：`通过 / 不通过`
- 字段匹配（模型字段 vs FSU 上报字段）：
  - 结果：`通过 / 不通过`
  - 缺失字段：`<无 / 字段列表>`

### 9.6 问题与处理记录

| 序号 | 现象 | 初步定位 | 处理动作 | 结论 |
|---|---|---|---|---|
| 1 |  |  |  |  |
| 2 |  |  |  |  |

### 9.7 验收结论

- 本次联调结论：`通过 / 有条件通过 / 不通过`
- 遗留问题：
  - `问题1`：`责任方`，`计划完成日期`
  - `问题2`：`责任方`，`计划完成日期`
- 下次联调计划时间：`YYYY-MM-DD HH:mm`

### 9.8 附件

- 日志文件：`<路径>`
- 抓包截图：`<路径>`
- 关键请求/响应截图：`<路径>`

---

## 10. FSU 代码解析（PandaX 北向）

### 10.1 生命周期与入口

- 适配器实现位于 `internal/northbound/adapters/pandax.go`，类型名固定为 `pandax`；
- 创建/更新北向配置后，运行时通过 `registerNorthboundAdapter -> adapter.Initialize(config)` 装载；
- `Start()` 仅启动循环，不会自动触发设备同步；
- 手动同步入口：`POST /api/northbound/{id}/sync-devices`（仅 PandaX 适配器支持）。

### 10.2 配置解析与默认值

`parsePandaXConfig` 支持多种别名输入：

- `serverUrl` 兼容 `broker` / `server_url`；
- `username` 兼容 `token` / `deviceToken`；
- `gatewayRegisterTopic` 兼容 `registerTopic`。

归一化后关键约束：

- `serverUrl`、`username` 必填；
- `gatewayMode=true` 强制要求；
- `qos` 仅允许 0~2；
- 默认 topic：
  - 实时：`v1/gateway/telemetry`
  - 同步：`v1/gateway/register/telemetry`
  - 报警：`v1/devices/event/alarm`
  - RPC 请求/响应：`v1/devices/me/rpc/request` / `v1/devices/me/rpc/response`。

### 10.3 上行数据链路（实时+系统属性）

- 采集器会把设备实时数据调用 `Send()` 入 PandaX 适配器队列；
- `runLoop()` 按 `uploadIntervalMs` 周期从数据库读取最新遥测并触发 `flushRealtime()`；
- `buildBatchRealtimePublish` 输出格式：
  - `{subToken: {"ts": <ms>, "values": {...}}}`；
- `subToken` 由 `subDeviceTokenMode` 决定（`deviceName` / `deviceKey` / `product_deviceKey` / `product_deviceName`）；
- 字段值会自动尝试转换为 `bool/int/float`，转换失败保持字符串。

补充：若数据库没有系统属性最新值，适配器会通过 `SystemStatsProvider` 追加 `__system__` 数据上报。

### 10.4 报警上报链路

- `SendAlarm()` 入报警队列；
- `runLoop()` 按 `alarmFlushIntervalMs` 周期批量发送；
- 报警 payload 包含 `product_key/device_key/field_name/actual_value/threshold/severity/message/ts`；
- 发送失败会把本批次数据回插队列头部（保序重试）。

### 10.5 设备同步链路（手动触发）

- `SyncDevices()` 读取设备列表 + 最新遥测，构造 `subDevices[]` 并发布到 `gatewayRegisterTopic`；
- `subDevices[i]` 包含 `productKey/deviceName/ts/values`；
- 同步前会执行按产品维度预检查：若某产品下全部子设备均无字段，则整次同步失败并返回错误；
- `productKey` 优先取驱动固定值：
  - `driver_id -> 驱动 wasm version() -> productKey`；
  - 若设备表 `product_key` 为空或不一致，会在同步前回写修正。

### 10.6 下行命令闭环

- 适配器订阅 `rpcRequestTopic` 及其 `.../+` 变体；
- 支持从 payload 和 topic 两处提取 `requestId`；
- `buildPandaXCommands` 支持多种参数结构：
  - `params.properties`
  - `sub_device/subDevice`
  - `sub_devices/subDevices`
  - `fieldName + value`
  - 兜底泛化字段；
- 命令进入 `commandQueue` 后，采集器每 500ms 轮询执行；
- 执行结果通过 `ReportCommandResult()` 回传到 `rpcResponseTopic`。

### 10.7 队列与失败策略

- 实时/报警/命令队列均有容量上限（默认 1000）；
- 队列满时丢弃最旧数据，优先保证新数据进入；
- MQTT 断连后会触发指数退避重连（上限 5 分钟）。

