# XunJi 北向接口说明（FSU）

## 1. 目标与范围

本文档用于说明 `fsu` 中 `xunji` 北向（`type: xunji`）与 `xunji` 平台南向 MQTT 接口的对接方式。

本次语义约定如下：

- **FSU 本身 = 网关设备**（gateway）
- **FSU 里的采集设备 = 子设备**（sub-device）
- 仅修改 FSU 侧适配逻辑，不修改 `xunji` 平台代码

---

## 2. 与 xunji 南向接口的对齐点

### 2.1 上行（FSU -> xunji）

FSU 使用网关批量上报 topic：

- `/sys/{gatewayProductKey}/{gatewayDeviceKey}/thing/event/property/pack/post`

报文方法固定为：

- `method = "thing.event.property.pack.post"`

子设备数据放在 `params.subDevices[]`：

- `subDevices[i].identity.productKey/deviceKey`：子设备身份
- `subDevices[i].properties`：子设备属性
- `subDevices[i].events`：子设备事件（报警也走这里）

### 2.2 下行（xunji -> FSU）

xunji 平台下发到网关 topic（由网关转子设备）：

- 属性设置：`/sys/{gatewayPK}/{gatewayDK}/thing/service/property/set`
- 服务调用：`/sys/{gatewayPK}/{gatewayDK}/thing/service/{service}`
- 配置下发：`/sys/{gatewayPK}/{gatewayDK}/thing/config/push`

子设备目标通过 `identity` 指定（与 xunji 平台下行路由一致）：

```json
{
  "id": "req-1001",
  "version": "1.0.0",
  "params": {
    "switch": 1,
    "targetTemp": 24
  },
  "identity": {
    "productKey": "subPk001",
    "deviceKey": "subDk001"
  }
}
```

> 兼容说明：FSU 也兼容 `params.subDevice` / `params.subDevices` 等包装格式。

---

## 3. FSU `xunji` 适配行为

1. **实时上报**
   - 每个采集设备作为一个子设备节点写入 `subDevices[]`
   - 网关身份来自北向配置的 `productKey/deviceKey`

2. **报警上报**
   - 与实时同 topic（`property/pack/post`）
   - 报警事件写入对应子设备的 `events.alarm`

3. **下行命令解析与执行**
   - FSU 从下行请求解析写入点位，进入命令队列执行
   - 子设备身份选择优先级：
     - 根 `identity`
     - `params` 内 identity（`identity/subDevice/subDevices`）
     - 网关 topic identity（兜底）

4. **下行回包**
   - 回复 topic：`/sys/{gatewayPK}/{gatewayDK}/thing/service/property/set_reply`
   - 回复使用网关身份（即 FSU 身份）

---

## 4. Payload 示例

### 4.1 实时上报（网关批量）

Topic：`/sys/gatewayPk/gatewayDk/thing/event/property/pack/post`

```json
{
  "id": "msg_1739000000000_1",
  "version": "1.0",
  "sys": { "ack": 0 },
  "method": "thing.event.property.pack.post",
  "params": {
    "properties": {},
    "events": {},
    "subDevices": [
      {
        "identity": {
          "productKey": "subPk001",
          "deviceKey": "subDk001"
        },
        "properties": {
          "temperature": 26.5,
          "humidity": 51.2
        },
        "events": {}
      }
    ]
  }
}
```

### 4.2 报警上报

```json
{
  "id": "alarm_1739000000000_1",
  "version": "1.0",
  "sys": { "ack": 0 },
  "method": "thing.event.property.pack.post",
  "params": {
    "properties": {},
    "events": {},
    "subDevices": [
      {
        "identity": {
          "productKey": "subPk001",
          "deviceKey": "subDk001"
        },
        "properties": {},
        "events": {
          "alarm": {
            "value": {
              "field_name": "temperature",
              "actual_value": 95.1,
              "threshold": 80,
              "operator": ">",
              "message": "温度超限"
            },
            "time": 1739000000000
          }
        }
      }
    ]
  }
}
```

### 4.3 属性下发（平台 -> FSU）

Topic：`/sys/gatewayPk/gatewayDk/thing/service/property/set`

```json
{
  "id": "req-1001",
  "version": "1.0.0",
  "params": {
    "switch": 1
  },
  "identity": {
    "productKey": "subPk001",
    "deviceKey": "subDk001"
  }
}
```

---

## 5. XunJi 北向配置建议

推荐最小配置：

```json
{
  "productKey": "gatewayPk",
  "deviceKey": "gatewayDk",
  "serverUrl": "tcp://127.0.0.1:1883",
  "username": "",
  "password": "",
  "qos": 0,
  "retain": false,
  "keepAlive": 60,
  "connectTimeout": 10,
  "uploadIntervalMs": 5000
}
```

说明：

- `productKey/deviceKey` 填网关身份（即 FSU 自身）
- 设备侧（子设备）身份来自设备表里的 `product_key/device_key`
- 若设备未配置身份，会回退网关身份（不建议）

---

## 6. 联调步骤（建议）

1. 在 xunji 平台创建网关设备，拿到 `gatewayPK/gatewayDK`。
2. 在 FSU 创建 `type=xunji` 北向配置，填入网关 `productKey/deviceKey`。
3. 在 FSU 设备列表给每个采集设备配置子设备 `product_key/device_key`。
4. 触发采集，平台订阅 `/sys/+/+/thing/event/property/pack/post` 验证 `subDevices` 上行。
5. 平台下发 `/sys/{gatewayPK}/{gatewayDK}/thing/service/property/set`，携带根 `identity` 指向子设备。
6. 观察 FSU 命令执行与 `set_reply` 回包。

### 6.1 最小联调清单（10分钟版）

1. **确认网关身份（1分钟）**
   - FSU：`/api/gateway/config` 中已配置 `product_key/device_key`。
   - xunji：已存在对应网关设备。

2. **确认子设备身份（2分钟）**
   - FSU 设备列表中，每个待联调设备都配置了 `product_key/device_key`。

3. **确认北向配置（1分钟）**
   - `type=xunji` 配置启用，`productKey/deviceKey` 使用网关身份。
   - `serverUrl` 指向可达 MQTT Broker。

4. **验证上行（2分钟）**
   - 在 Broker 订阅：`/sys/+/+/thing/event/property/pack/post`。
   - 触发一次采集，确认 payload 中出现 `params.subDevices[]`。

5. **验证下行（2分钟）**
   - 发布到：`/sys/{gatewayPK}/{gatewayDK}/thing/service/property/set`。
   - 请求体包含根 `identity`（子设备）+ `params`（写入点位）。

6. **验证回包（2分钟）**
   - 订阅：`/sys/{gatewayPK}/{gatewayDK}/thing/service/property/set_reply`。
   - 检查回包 `id/code/message/data` 是否与请求一致。

> 通过标准：
> - 上行包：能看到子设备 `identity` 和实时点位。
> - 下行包：能命中目标子设备并完成写入。
> - 回包：稳定返回到网关 `set_reply` topic。

### 6.2 `mosquitto` 命令模板（可直接复制）

先准备环境变量（按你的环境替换）：

```bash
export MQTT_HOST="127.0.0.1"
export MQTT_PORT="1883"
export MQTT_USER=""
export MQTT_PASS=""

export GATEWAY_PK="gatewayPk"
export GATEWAY_DK="gatewayDk"
export SUB_PK="subPk001"
export SUB_DK="subDk001"
```

1) 订阅上行批量数据（观察 `subDevices`）：

```bash
mosquitto_sub -h "$MQTT_HOST" -p "$MQTT_PORT" \
  -u "$MQTT_USER" -P "$MQTT_PASS" \
  -t "/sys/+/+/thing/event/property/pack/post" -v
```

2) 订阅属性下发回包：

```bash
mosquitto_sub -h "$MQTT_HOST" -p "$MQTT_PORT" \
  -u "$MQTT_USER" -P "$MQTT_PASS" \
  -t "/sys/${GATEWAY_PK}/${GATEWAY_DK}/thing/service/property/set_reply" -v
```

3) 向网关下发“写子设备属性”命令：

```bash
mosquitto_pub -h "$MQTT_HOST" -p "$MQTT_PORT" \
  -u "$MQTT_USER" -P "$MQTT_PASS" \
  -t "/sys/${GATEWAY_PK}/${GATEWAY_DK}/thing/service/property/set" \
  -m '{
    "id":"req-1001",
    "version":"1.0.0",
    "params":{"switch":1},
    "identity":{"productKey":"'"${SUB_PK}"'","deviceKey":"'"${SUB_DK}"'"}
  }'
```

4) （可选）下发服务调用命令：

```bash
mosquitto_pub -h "$MQTT_HOST" -p "$MQTT_PORT" \
  -u "$MQTT_USER" -P "$MQTT_PASS" \
  -t "/sys/${GATEWAY_PK}/${GATEWAY_DK}/thing/service/reboot" \
  -m '{
    "id":"svc-2001",
    "version":"1.0.0",
    "params":{"delay":3},
    "identity":{"productKey":"'"${SUB_PK}"'","deviceKey":"'"${SUB_DK}"'"}
  }'
```

---

## 7. 常见问题

1. **平台能收到上行，但下行写不到目标设备**
   - 检查下行请求是否携带根 `identity.productKey/deviceKey`。
   - 检查 FSU 设备 `product_key/device_key` 是否与 identity 一致。

2. **命令回包 topic 不正确**
   - 回包必须走网关 topic：`/sys/{gatewayPK}/{gatewayDK}/thing/service/property/set_reply`。

3. **子设备上行被识别成网关自身**
   - 说明设备未正确配置子设备身份，回退到了网关身份。

4. **MQTT 连接失败**
   - 检查 `serverUrl` 是否包含协议（如 `tcp://`）和可达端口。
