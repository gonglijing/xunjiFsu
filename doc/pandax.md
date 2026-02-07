# PandaX 北向接口说明（FSU）

## 1. 目标与范围

本文档用于说明 `fsu` 中新增的北向类型 **PandaX**（`type: pandax`）的对接方式。

- `xunji` 北向继续保留，用于其他平台。
- `pandax` 作为新的独立北向类型，用于对接 PandaX 平台 MQTT。
- 不修改 PandaX 平台代码，仅通过 FSU 侧适配实现联通。

---

## 2. PandaX MQTT 协议要点

### 2.1 认证

- 使用 MQTT `Username` 作为设备 token（必填）。
- `Password` 可为空（取决于平台策略）。

### 2.2 关键 Topic

- 设备直连上报
  - `v1/devices/me/telemetry`
  - `v1/devices/me/attributes`
  - `v1/devices/me/row`
- 网关子设备上报
  - `v1/gateway/telemetry`
  - `v1/gateway/attributes`
- 事件上报
  - `v1/devices/event/{identifier}`
- RPC
  - 请求：`v1/devices/me/rpc/request`（以及带 requestId 的子路径）
  - 响应：`v1/devices/me/rpc/response`

---

## 3. FSU PandaX 适配行为

FSU 中 `pandax` 适配器的默认行为如下：

1. **实时数据上报**
   - 默认 `gatewayMode=true`：发送到 `v1/gateway/telemetry`
   - payload 结构：

```json
{
  "{subDeviceToken}": {
    "ts": 1738999999000,
    "values": {
      "temperature": 26.5,
      "humidity": 51.2
    }
  }
}
```

2. **直连模式上报**（`gatewayMode=false`）
   - 发送到 `v1/devices/me/telemetry`
   - payload 结构：

```json
{
  "ts": 1738999999000,
  "values": {
    "temperature": 26.5,
    "humidity": 51.2
  }
}
```

3. **报警上报**
   - 默认 Topic：`v1/devices/event/alarm`
   - 可通过 `alarmTopic` 或 `eventTopicPrefix + alarmIdentifier` 调整。

4. **命令下发接收**
   - 订阅：`v1/devices/me/rpc/request` 和 `v1/devices/me/rpc/request/+`
   - 从请求中解析写入命令，进入 FSU 北向命令队列。

5. **命令执行结果回传**
   - 发布到：`v1/devices/me/rpc/response`

---

## 4. 创建 PandaX 北向配置

### 4.1 API 创建示例

请求：`POST /api/northbound`

```json
{
  "name": "pandax-main",
  "type": "pandax",
  "enabled": 1,
  "upload_interval": 5000,
  "server_url": "tcp://127.0.0.1",
  "port": 1883,
  "username": "your-device-token",
  "config": "{\"serverUrl\":\"tcp://127.0.0.1:1883\",\"username\":\"your-device-token\",\"gatewayMode\":true,\"qos\":0,\"connectTimeout\":10,\"uploadIntervalMs\":5000}"
}
```

> 建议：`server_url/username` 与 `config` 中保持一致，便于排错。

### 4.2 推荐 config JSON（可直接用于页面 Schema 表单）

```json
{
  "serverUrl": "tcp://127.0.0.1:1883",
  "username": "your-device-token",
  "password": "",
  "clientId": "",
  "qos": 0,
  "retain": false,
  "keepAlive": 60,
  "connectTimeout": 10,
  "gatewayMode": true,
  "subDeviceTokenMode": "deviceName",
  "gatewayTelemetryTopic": "v1/gateway/telemetry",
  "gatewayAttributesTopic": "v1/gateway/attributes",
  "telemetryTopic": "v1/devices/me/telemetry",
  "attributesTopic": "v1/devices/me/attributes",
  "eventTopicPrefix": "v1/devices/event",
  "alarmIdentifier": "alarm",
  "rpcRequestTopic": "v1/devices/me/rpc/request",
  "rpcResponseTopic": "v1/devices/me/rpc/response",
  "uploadIntervalMs": 5000,
  "alarmFlushIntervalMs": 2000,
  "alarmBatchSize": 20,
  "alarmQueueSize": 1000,
  "realtimeQueueSize": 1000,
  "commandQueueSize": 1000,
  "productKey": "",
  "deviceKey": ""
}
```

---

## 5. RPC 联调示例

### 5.1 平台下发（FSU 订阅）

Topic：`v1/devices/me/rpc/request/1001`

```json
{
  "requestId": "1001",
  "method": "write",
  "params": {
    "productKey": "pk001",
    "deviceKey": "dk001",
    "properties": {
      "switch": 1,
      "targetTemp": 24
    }
  }
}
```

### 5.2 FSU 回执（FSU 发布）

Topic：`v1/devices/me/rpc/response`

```json
{
  "requestId": "1001",
  "method": "write",
  "params": {
    "success": true,
    "code": 200,
    "message": "success",
    "productKey": "pk001",
    "deviceKey": "dk001",
    "fieldName": "switch",
    "value": 1
  }
}
```

---

## 6. 联调步骤（建议）

1. 在 PandaX 平台准备好设备 token。
2. 在 FSU 新建北向配置：`type=pandax`，填写 `serverUrl` 与 `username(token)`。
3. 启用配置后，在 FSU 页面确认北向运行态为已注册/运行。
4. 让 FSU 产生实时数据，平台订阅 `v1/gateway/telemetry` 验证上行。
5. 在平台发布 RPC 到 `v1/devices/me/rpc/request/{id}`，检查 FSU 命令队列和回执。
6. 触发报警，验证 `v1/devices/event/alarm` 是否收到事件。

---

## 7. 常见问题

1. **连接不上**
   - 先检查 `serverUrl` 是否带协议（如 `tcp://`）和端口是否可达。

2. **认证失败**
   - 优先确认 `username` 是否为正确 device token。

3. **有数据无命令**
   - 检查 `rpcRequestTopic` 是否为 `v1/devices/me/rpc/request`。
   - 检查平台是否发布到 `request` 或 `request/{id}`。

4. **命令入队但写入失败**
   - 检查请求中的 `productKey/deviceKey/fieldName/value` 是否符合设备点位映射。

5. **子设备 token 不符合平台期望**
   - 调整 `subDeviceTokenMode`：`deviceName` / `deviceKey` / `product_deviceKey` / `product_deviceName`。

