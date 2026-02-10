# HuShu 智能网关（FSU）

基于 Go + SolidJS 的工业物联网网关系统，支持资源/设备/驱动管理、数据采集、阈值告警与北向对接。

## 当前版本关键说明（先看）

- 首页已切换为**拓扑视图**（不再是仪表盘）。
- “同步网关身份到北向”功能已删除（无 `/api/gateway/northbound/sync-identity`）。
- 网关设置页已移除 `ProductKey` / `DeviceKey` 字段；这两个字段在对应北向（如 Sagoo / iThings）配置中维护。
- 北向类型以代码为准：`mqtt`、`pandax`、`ithings`、`sagoo`。

---

## 1. 项目能力

- **资源管理**：串口/网络资源管理。
- **设备管理**：设备绑定资源、驱动，支持启停。
- **驱动管理**：WASM 驱动上传、重载、执行。
- **数据采集**：按设备采集周期调度，缓存最新值并归档历史。
- **阈值告警**：阈值规则、告警记录、确认流程。
- **北向对接**：内置适配器（非插件模式）统一调度发送。
- **运行时热更新**：采集/驱动/MQTT 重连等参数可在线更新并审计。

---

## 2. 技术栈

- 后端：Go（`go.mod` 当前为 `1.24.0`）
- Web：`gorilla/mux`
- 数据库：SQLite（`param.db` + `data.db`）
- 驱动：Extism + TinyGo WASM
- 前端：SolidJS + Vite
- 认证：JWT（Cookie + Bearer）

---

## 3. 架构与启动流程

程序入口：`cmd/main.go`

启动阶段（`internal/app/app.go`）主要流程：

1. 加载/生成会话密钥（`config/session_secret.key`）
2. 初始化参数库与数据库
3. 初始化 schema 与默认数据（默认管理员）
4. 启动数据同步任务（内存数据批量落盘）
5. 启动数据保留清理任务
6. 加载已启用驱动
7. 加载并启动已启用北向配置
8. 启动采集器 + 系统监控采集器
9. 启动 HTTP/HTTPS 服务（支持自动证书与手动证书）

---

## 4. 目录说明

```text
fsu/
├── cmd/main.go                  # 程序入口
├── internal/
│   ├── app/                     # 应用启动、路由、运行时调优
│   ├── auth/                    # JWT 认证
│   ├── collector/               # 采集调度、命令轮询
│   ├── database/                # DB 初始化、CRUD、同步、保留策略
│   ├── driver/                  # WASM 驱动管理与执行
│   ├── handlers/                # HTTP API handlers
│   ├── models/                  # 领域模型
│   └── northbound/              # 北向管理器与内置适配器
├── ui/frontend/                 # SolidJS 前端源码
├── ui/static/dist/              # 前端构建产物
├── config/config.yaml           # 默认配置文件
├── migrations/                  # SQL 迁移
└── README.md
```

---

## 5. 前端页面路由（当前）

前端路由定义在 `ui/frontend/src/App.jsx`：

- `/` → 拓扑视图（首页）
- `/topology` → 拓扑视图
- `/gateway` → 网关设置
- `/resources` → 资源管理
- `/devices` → 设备管理
- `/drivers` → 驱动管理
- `/northbound` → 北向配置
- `/thresholds` → 阈值管理
- `/alarms` → 报警
- `/realtime` → 实时数据
- `/login` → 登录

未命中路由默认回到拓扑页。

---

## 6. 北向类型与配置

支持类型：

- `mqtt`
- `pandax`
- `ithings`
- `sagoo`

Schema 接口：

- `GET /api/northbound/schema?type=<type>`
- 未提供 `type` 时默认 `pandax`

说明：

- Sagoo / iThings 的网关身份字段在其 `config` 中维护（例如 `productKey`、`deviceKey`）。
- 北向运行时为内置适配器模式，不依赖旧插件目录。

---

## 7. 运行时参数热更新（重点）

接口：

- `GET /api/gateway/runtime`
- `PUT /api/gateway/runtime`
- `GET /api/gateway/runtime/audits`

当前可热更新参数：

- `collector_device_sync_interval`
- `collector_command_poll_interval`
- `northbound_mqtt_reconnect_interval`
- `driver_serial_read_timeout`
- `driver_tcp_dial_timeout`
- `driver_tcp_read_timeout`
- `driver_serial_open_backoff`
- `driver_tcp_dial_backoff`
- `driver_serial_open_retries`
- `driver_tcp_dial_retries`

### 参数作用域分析

- **采集设备同步周期**：全局参数，影响采集器多久与数据库同步设备启停/配置。
- **MQTT 重连间隔**：全局参数，应用到支持 `SetReconnectInterval` 的北向适配器实例。
- **串口读超时 / TCP 超时重试**：全局驱动执行参数，作用于执行器层。

### 是否“可以不写这些参数”？

可以不传，系统会继续使用当前值/默认值；但这些参数仍有保留价值：

- 线上排障时可快速调优，不必重启或发版。
- 统一限制全局连接/重试行为，避免雪崩重连。
- 变更会写入审计日志，便于追溯。

### 与设备专属参数的关系

- 设备自身仍有专属采集参数（如 `collect_interval`、`storage_interval`）。
- 当前“运行时热更新”更多是**全局调优层**，与设备级参数互补。

---

## 8. 数据库与数据流

### 参数库（`param.db`）

持久化存储配置数据：用户、网关设置、资源、设备、驱动、北向配置、阈值、告警等。

### 数据库（`data.db`）

运行时采用内存库处理实时写入，后台批量同步至磁盘文件。

### 数据清理

- 按网关配置中的 `data_retention_days` 清理历史数据。
- 默认每天执行一次清理任务。

---

## 9. 认证与默认账户

- 登录接口：`POST /login`
- API 统一走 JWT 鉴权（Cookie `gogw_jwt` 或 `Authorization: Bearer <token>`）
- 默认账户：
  - 用户名：`admin`
  - 密码：`123456`

> 首次部署后请立即修改默认密码。

---

## 10. API 概览

所有 `/api/*` 需要登录鉴权。

### 系统

- `GET /api/status`
- `POST /api/collector/start`
- `POST /api/collector/stop`

### 网关

- `GET /api/gateway/config`
- `PUT /api/gateway/config`
- `GET /api/gateway/runtime`
- `PUT /api/gateway/runtime`
- `GET /api/gateway/runtime/audits`

### 资源

- `GET /api/resources`
- `POST /api/resources`
- `PUT /api/resources/{id}`
- `DELETE /api/resources/{id}`
- `POST /api/resources/{id}/toggle`

### 设备

- `GET /api/devices`
- `POST /api/devices`
- `PUT /api/devices/{id}`
- `DELETE /api/devices/{id}`
- `POST /api/devices/{id}/toggle`
- `POST /api/devices/{id}/execute`
- `GET /api/devices/{id}/writables`

### 驱动

- `GET /api/drivers`
- `GET /api/drivers/runtime`
- `GET /api/drivers/files`
- `POST /api/drivers`
- `PUT /api/drivers/{id}`
- `DELETE /api/drivers/{id}`
- `GET /api/drivers/{id}/runtime`
- `POST /api/drivers/{id}/reload`
- `GET /api/drivers/{id}/download`
- `POST /api/drivers/upload`

### 北向

- `GET /api/northbound`
- `GET /api/northbound/status`
- `GET /api/northbound/schema`
- `POST /api/northbound`
- `PUT /api/northbound/{id}`
- `DELETE /api/northbound/{id}`
- `POST /api/northbound/{id}/toggle`
- `POST /api/northbound/{id}/reload`

### 阈值、告警、数据、用户

- `GET/POST/PUT/DELETE /api/thresholds...`
- `GET /api/alarms`
- `POST /api/alarms/{id}/acknowledge`
- `GET /api/data`
- `GET /api/data/cache/{id}`
- `GET /api/data/history`
- `GET/POST/PUT/DELETE /api/users...`
- `PUT /api/users/password`

健康检查：`/health`、`/ready`、`/live`、`/metrics`

---

## 11. 配置说明

默认读取顺序：

1. `config/config.yaml`
2. 环境变量覆盖

常用环境变量：

- `LISTEN_ADDR`
- `PARAM_DB_PATH`
- `DATA_DB_PATH`
- `SESSION_SECRET`
- `ALLOWED_ORIGINS`
- `LOG_LEVEL` / `LOG_JSON`
- `COLLECTOR_DEVICE_SYNC_INTERVAL`
- `COLLECTOR_COMMAND_POLL_INTERVAL`
- `NORTHBOUND_MQTT_RECONNECT_INTERVAL`
- `DRIVER_SERIAL_READ_TIMEOUT`
- `DRIVER_TCP_DIAL_TIMEOUT`
- `DRIVER_TCP_READ_TIMEOUT`
- `DRIVER_SERIAL_OPEN_RETRIES`
- `DRIVER_TCP_DIAL_RETRIES`

> 配置文件中示例监听端口是 `:8088`，代码默认值是 `:8080`，最终以“加载到的配置 + 环境变量覆盖”为准。

---

## 12. 本地开发与构建

### 后端

```bash
go run ./cmd/main.go
```

或：

```bash
make run
```

### 前端

```bash
npm --prefix ui/frontend install
npm --prefix ui/frontend run build
```

开发模式：

```bash
npm --prefix ui/frontend run dev --host
```

### 常用验证

```bash
go test ./...
npm --prefix ui/frontend run test
npm --prefix ui/frontend run build
```

---

## 13. 驱动开发

驱动源码位于 `drvs/`，编译后生成 `.wasm` 供网关加载。

- 单驱动编译：进入对应子目录执行 `make`
- 批量编译：`cd drvs && make`

建议优先参考：`drvs/README.md`

---

## 14. 安全建议

- 首次部署立即修改默认管理员密码。
- 生产环境启用 HTTPS（证书或自动证书）。
- 使用强随机 `SESSION_SECRET`。
- 严格限制 `ALLOWED_ORIGINS`。

---

## 15. License

MIT
