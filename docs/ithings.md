# iThings 北向接口说明（FSU）

## 1. 目标与范围

本文档说明 `fsu` 中新增北向类型 **iThings**（`type: ithings`）的对接方式。

- `xunji` 北向继续保留，用于其他平台。
- `pandax` 北向用于 PandaX 平台。
- `ithings` 作为新的独立北向类型，用于对接 things/iThings 平台 MQTT。
- 仅修改 FSU 侧代码，不修改 things 平台代码。

---

## 2. iThings MQTT 协议要点（对应 things 南向）

### 2.1 关键 Topic

- 属性上行：`$thing/up/property/{productID}/{deviceName}`
- 属性下行：`$thing/down/property/{productID}/{deviceName}`
- 事件上行：`$thing/up/event/{productID}/{deviceName}`
- 事件下行：`$thing/down/event/{productID}/{deviceName}`
- 行为上行：`$thing/up/action/{productID}/{deviceName}`
- 行为下行：`$thing/down/action/{productID}/{deviceName}`

### 2.2 下行命令语义

- 属性控制下发：`method = "control"`，参数在 `params`
- 行为调用下发：`method = "action"`，携带 `actionID` 和 `params`

设备侧回执：
- 属性回执：`method = "controlReply"`（发布到上行属性 topic）
- 行为回执：`method = "actionReply"`（发布到上行行为 topic）

---

## 3. FSU iThings 适配行为

FSU 侧 `ithings` 适配器仅实现：**网关 + 子设备**。

1. **实时数据上报（packReport）**
   - 网关上行 Topic：`$thing/up/property/{gatewayProductID}/{gatewayDeviceName}`
   - Payload 使用 `method = "packReport"`
   - 子设备数据放在 `subDevices[]` 中

2. **报警上报（eventPost）**
   - 网关上行事件 Topic：`$thing/up/event/{gatewayProductID}/{gatewayDeviceName}`
   - Payload 使用 `method = "eventPost"`

3. **命令接收与回执**
   - 订阅：`$thing/down/property/+/+`、`$thing/down/action/+/+`
   - 属性命令解析为 FSU 写入命令队列
   - 行为命令解析为 FSU 动作命令队列
   - 执行结果回传 `controlReply` / `actionReply`

> `ithings` 仅支持 `gatewayMode=true`。

---

## 4. Payload 示例

### 4.1 实时上报（packReport）

Topic：`$thing/up/property/gatewayPk/gatewayDn`

```json
{
  "method": "packReport",
  "msgToken": "pack_1738999999000_1",
  "timestamp": 1738999999000,
  "properties": [],
  "events": [],
  "subDevices": [
    {
      "productID": "subPk001",
      "deviceName": "subDn001",
      "properties": [
        {
          "timestamp": 1738999999000,
          "params": {
            "temperature": 26.5,
            "humidity": 51.2
          }
        }
      ],
      "events": []
    }
  ]
}
```

### 4.2 属性下发与回执

下发 Topic：`$thing/down/property/subPk001/subDn001`

```json
{
  "method": "control",
  "msgToken": "ctrl-1001",
  "params": {
    "switch": 1
  }
}
```

回执 Topic：`$thing/up/property/subPk001/subDn001`

```json
{
  "method": "controlReply",
  "msgToken": "ctrl-1001",
  "code": 0,
  "msg": "success",
  "timestamp": 1738999999555
}
```

### 4.3 行为下发与回执

下发 Topic：`$thing/down/action/subPk001/subDn001`

```json
{
  "method": "action",
  "msgToken": "act-2001",
  "actionID": "reboot",
  "params": {
    "delay": 3
  }
}
```

回执 Topic：`$thing/up/action/subPk001/subDn001`

```json
{
  "method": "actionReply",
  "msgToken": "act-2001",
  "actionID": "reboot",
  "code": 0,
  "msg": "success",
  "timestamp": 1738999999666
}
```

---

## 5. iThings 北向配置示例

### 5.1 API 创建示例

请求：`POST /api/northbound`

```json
{
  "name": "ithings-main",
  "type": "ithings",
  "enabled": 1,
  "upload_interval": 5000,
  "server_url": "tcp://127.0.0.1",
  "port": 1883,
  "username": "ithings-user",
  "product_key": "gatewayPk",
  "device_key": "gatewayDn",
  "config": "{\"serverUrl\":\"tcp://127.0.0.1:1883\",\"username\":\"ithings-user\",\"productKey\":\"gatewayPk\",\"deviceKey\":\"gatewayDn\",\"gatewayMode\":true,\"qos\":0,\"uploadIntervalMs\":5000}"
}
```

### 5.2 推荐 config JSON

```json
{
  "serverUrl": "tcp://127.0.0.1:1883",
  "username": "ithings-user",
  "password": "",
  "clientId": "",
  "qos": 0,
  "retain": false,
  "keepAlive": 60,
  "connectTimeout": 10,
  "gatewayMode": true,
  "productKey": "gatewayPk",
  "deviceKey": "gatewayDn",
  "deviceNameMode": "deviceKey",
  "subDeviceNameMode": "deviceKey",
  "upPropertyTopicTemplate": "$thing/up/property/{productID}/{deviceName}",
  "upEventTopicTemplate": "$thing/up/event/{productID}/{deviceName}",
  "upActionTopicTemplate": "$thing/up/action/{productID}/{deviceName}",
  "downPropertyTopic": "$thing/down/property/+/+",
  "downActionTopic": "$thing/down/action/+/+",
  "alarmEventID": "alarm",
  "alarmEventType": "alert",
  "uploadIntervalMs": 5000,
  "alarmFlushIntervalMs": 2000,
  "alarmBatchSize": 20,
  "alarmQueueSize": 1000,
  "realtimeQueueSize": 1000,
  "commandQueueSize": 1000
}
```

---

## 6. 联调建议

1. 在 things 平台确认网关设备 `productID/deviceName`。
2. 在 FSU 创建 `type=ithings`，填写 `serverUrl/username/productKey/deviceKey`。
3. 启用后确认北向运行态为已注册/运行。
4. 触发 FSU 实时采集，订阅 `$thing/up/property/#` 验证 `packReport`。
5. 向 `$thing/down/property/{pk}/{dn}` 下发 `control`，验证命令执行与 `controlReply`。
6. 向 `$thing/down/action/{pk}/{dn}` 下发 `action`，验证 `actionReply`。

---

## 7. 常见问题

1. **连接失败**
   - 检查 `serverUrl` 协议与端口（如 `tcp://host:1883`）。

2. **认证失败**
   - 核对 `username/password`。

3. **实时有数据但平台不识别**
   - 确认网关 `productKey/deviceKey` 与平台设备一致。
   - 确认 payload 使用 `packReport` 且 `subDevices` 字段完整。

4. **下行命令收不到**
   - 检查是否订阅到 `$thing/down/property/+/+`、`$thing/down/action/+/+`。

5. **回执不匹配请求**
   - 确认下发中 `msgToken` 存在，FSU 会按 `msgToken` 关联回执。
