# HuShu 智能网关

一个基于 Go + SolidJS 开发的工业物联网网关管理系统，支持串口/网口资源配置、设备驱动管理、数据采集、阈值报警和北向接口对接。

```
┌─────────────────────────────────────────────────────────────────────────┐
│                           HuShu 智能网关系统                             │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│   ┌────────────────────────────────────────────────────────────────┐    │
│   │                        Web 管理界面                              │    │
│   │   SolidJS + Vite + CSS3 (深色主题、霓虹发光、平滑动画)           │    │
│   └────────────────────────────────────────────────────────────────┘    │
│                                    │                                    │
│                                    ▼                                    │
│   ┌────────────────────────────────────────────────────────────────┐    │
│   │                      Go HTTP Server                             │    │
│   │   Gorilla Mux + JWT 认证 + 双 SQLite 数据库                      │    │
│   └────────────────────────────────────────────────────────────────┘    │
│                                    │                                    │
│         ┌──────────────────────────┼──────────────────────────┐        │
│         ▼                          ▼                          ▼        │
│   ┌───────────┐           ┌───────────────┐           ┌───────────┐   │
│   │  采集器    │           │  北向调度器   │           │  阈值检查  │   │
│   │ Collector │           │  Scheduler    │           │  Checker  │   │
│   └─────┬─────┘           └───────┬───────┘           └─────┬─────┘   │
│         │                         │                          │         │
│         ▼                         ▼                          ▼         │
│   ┌────────────────────────────────────────────────────────────────┐   │
│   │                  DriverManager (驱动管理器)                      │   │
│   │         ┌─────────────────────────────────────────────────┐     │   │
│   │         │              Extism WASM Runtime                │     │   │
│   │         │  serial_transceive  │  tcp_transceive  │ ...   │     │   │
│   │         └─────────────────────────────────────────────────┘     │   │
│   └────────────────────────────────────────────────────────────────┘   │
│                                    │                                    │
│         ┌──────────────────────────┼──────────────────────────┐        │
│         ▼                          ▼                          ▼        │
│   ┌───────────┐           ┌───────────────┐           ┌───────────┐   │
│   │  串口资源  │           │   TCP 连接    │           │  其他资源  │   │
│   └───────────┘           └───────────────┘           └───────────┘   │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

## 功能特性

### 核心功能

| 功能 | 说明 |
|------|------|
| **资源配置** | 串口 (Serial)、TCP/UDP 网络资源管理 |
| **设备管理** | 一个资源可对接多个设备，独立通信参数 |
| **驱动管理** | Extism + TinyGo WASM 插件式驱动 |
| **数据采集** | 优先队列调度、自定义采集周期 |
| **阈值报警** | 多条件判断、严重程度分级、自动触发 |
| **双数据库** | param.db (配置) + data.db (历史数据，内存缓冲 + 磁盘归档) |
| **北向接口** | HashiCorp go-plugin + HTTP / MQTT / XunJi 插件 |

### Web 管理界面

- **SolidJS 前端**: 信号式状态管理、响应式渲染
- **现代化 UI**: 深色主题、渐变色彩、发光效果、平滑动画
- **响应式设计**: 适配桌面和移动设备
- **弹出窗口**: 新增/编辑设备/配置采用模态框
- **下拉导航**: 设置菜单整合资源/设备/驱动等

## 技术栈

| 层级 | 技术 | 版本/说明 |
|------|------|-----------|
| 后端 | Go | 1.21+ |
| 数据库 | SQLite | pure Go (glebarez/go-sqlite) |
| Web 框架 | Gorilla Mux | 路由中间件 |
| 前端框架 | SolidJS | 1.8.15 |
| 构建工具 | Vite | 5.2.9 |
| 驱动运行时 | Extism | go-sdk |
| 驱动开发 | TinyGo | 0.34+ |
| 北向插件 | HashiCorp go-plugin | net/rpc |
| 认证 | JWT | 鉴权中间件 |
| 跨平台 | CGO=0 | arm32 / arm64 / darwin / windows |

## 项目结构

```
gogw/
├── cmd/
│   └── main.go                 # 程序入口，配置加载
├── plugin_north/               # 北向插件源码与产物
│   ├── src/
│   │   ├── northbound-http/    # HTTP 插件入口
│   │   ├── northbound-mqtt/    # MQTT 插件入口
│   │   └── northbound-xunji/   # XunJi 插件入口
│   ├── northbound-http         # 插件二进制 (make northbound-plugins)
│   ├── northbound-mqtt
│   └── northbound-xunji
├── internal/
│   ├── app/
│   │   ├── app.go              # 启动逻辑、优雅关闭
│   │   ├── db.go               # 数据库初始化
│   │   ├── http.go             # HTTP 服务器构建
│   │   ├── northbound.go       # 北向调度器
│   │   └── secret.go           # JWT 密钥管理
│   ├── auth/
│   │   └── auth.go             # JWT 认证中间件
│   ├── collector/
│   │   ├── collector.go        # 采集调度器（优先队列）
│   │   └── threshold_cache.go  # 阈值缓存
│   ├── database/
│   │   ├── database.go         # 双数据库管理
│   │   ├── device.go           # 设备 CRUD
│   │   ├── health.go           # 健康检查
│   │   └── resource.go         # 资源 CRUD
│   ├── driver/
│   │   └── manager.go          # 驱动加载/执行/Host Functions
│   ├── handlers/
│   │   ├── auth.go             # 登录/登出
│   │   ├── data.go             # 数据查询
│   │   ├── device.go           # 设备管理
│   │   ├── driver.go           # 驱动管理
│   │   ├── handlers.go         # Handler 初始化
│   │   ├── northbound.go       # 北向配置
│   │   ├── pages.go            # 页面路由
│   │   ├── resource.go         # 资源管理
│   │   ├── response.go         # 统一响应
│   │   └── user.go             # 用户管理
│   ├── models/
│   │   └── models.go           # 数据模型定义
│   ├── northbound/
│   │   ├── manager.go          # 北向调度与熔断
│   │   └── plugin.go           # go-plugin 适配与 RPC
│   ├── resource/
│   │   └── manager.go          # 串口/TCP 连接管理
│   └── logger/
│       └── logger.go           # 日志封装
├── ui/
│   ├── frontend/               # SolidJS 前端
│   │   ├── src/
│   │   │   ├── api.js          # HTTP 客户端 + hooks
│   │   │   ├── router.jsx      # 路由管理
│   │   │   ├── main.jsx        # 入口文件
│   │   │   ├── App.jsx         # 根组件
│   │   │   ├── components/     # UI 组件
│   │   │   │   ├── TopNav.jsx  # 导航栏
│   │   │   │   ├── cards.jsx   # 卡片组件
│   │   │   │   └── Toast.jsx   # 通知组件
│   │   │   ├── pages/          # 页面
│   │   │   │   ├── Dashboard.jsx
│   │   │   │   ├── Login.jsx
│   │   │   │   ├── DevicesPage.jsx
│   │   │   │   └── ...
│   │   │   └── sections/       # 页面区块
│   │   │       ├── Devices.jsx
│   │   │       ├── Northbound.jsx
│   │   │       └── ...
│   │   ├── package.json
│   │   └── vite.config.js
│   └── static/
│       ├── dist/main.js        # 构建后的 JS
│       └── style.css           # 样式文件
├── drvs/                       # TinyGo 驱动源码
│   ├── README.md               # 驱动开发指南
│   ├── th_modbusrtu.go         # Modbus RTU 驱动
│   ├── th_modbustcp.go         # Modbus TCP 驱动
│   └── modbus/
│       ├── rtu.go              # Modbus RTU 协议包
│       └── README.md
├── migrations/
│   ├── 001_init.sql
│   ├── 002_param_schema.sql
│   ├── 003_data_schema.sql
│   └── 004_indexes.sql
├── Makefile                    # 编译脚本
├── config/
│   └── config.yaml             # 配置文件
└── README.md
```

## 快速开始

### 环境要求

- **Go 1.21+**
- **Node.js 18+** (前端开发)
- **TinyGo 0.34+** (驱动开发)

### 安装运行

```bash
# 克隆项目
git clone https://github.com/gonglijing/xunjiFsu.git
cd xunjiFsu

# 安装前端依赖
cd ui/frontend && npm install && cd ../..

# 编译前端
make ui

# 编译后端（当前平台）
make build

# 编译北向插件
make northbound-plugins

# 运行
./gogw

# 或使用 make 运行
make run
```

### 北向插件配置

默认插件目录：`plugin_north/`，可在 `config/config.yaml` 中设置：

```yaml
northbound:
  plugins_dir: "plugin_north"
```

如果需要为单个北向配置指定插件路径，可在配置 JSON 中设置 `plugin_path`。

#### 配置示例

HTTP:

```json
{
  "url": "http://127.0.0.1:8080/ingest",
  "headers": { "Authorization": "Bearer xxx" },
  "timeout": 30
}
```

MQTT:

```json
{
  "broker": "tcp://127.0.0.1:1883",
  "topic": "gogw/data",
  "alarm_topic": "gogw/alarm",
  "client_id": "gogw-mqtt-1",
  "username": "",
  "password": "",
  "qos": 0,
  "retain": false,
  "keep_alive": 60,
  "connect_timeout": 10
}
```

XunJi:

```json
{
  "productKey": "pk",
  "deviceKey": "dk",
  "serverUrl": "tcp://127.0.0.1:1883",
  "username": "",
  "password": "",
  "topic": "xunji/pk/dk",
  "alarmTopic": "xunji/pk/dk/alarm",
  "uploadIntervalMs": 5000,
  "grpcAddress": "127.0.0.1:32001",
  "alarmFlushIntervalMs": 2000,
  "alarmBatchSize": 20,
  "alarmQueueSize": 1000,
  "realtimeQueueSize": 1000,
  "qos": 0,
  "retain": false,
  "keepAlive": 60,
  "connectTimeout": 10
}
```

说明：

- 主程序与 XUNJI 插件使用 gRPC 通信。
- 主程序将实时数据和报警数据主动推送到插件 gRPC 服务。
- `grpcAddress` 可显式指定；若为空，插件与主程序会按 `productKey + deviceKey` 计算同一默认地址。
- `uploadIntervalMs` 为插件侧 MQTT 上报周期，报警批量参数由插件侧管理。
- XUNJI 配置参数采用类似 Terraform SDK Schema 的定义方式（必填/可选/默认值/类型校验）。

### 开发模式

```bash
# 后端热重载
make run

# 前端开发服务器（热重载）
make ui-dev
```

### 跨平台编译

```bash
# 查看帮助
make help

# 编译所有平台
make deploy

# 指定平台
make deploy-arm64    # Linux ARM64
make deploy-arm32    # Linux ARM32
make deploy-darwin   # macOS x86_64
make deploy-darwin-arm64  # macOS Apple Silicon
make deploy-windows  # Windows
```

### 默认配置

| 配置项 | 值 |
|--------|-----|
| 配置数据库 | `param.db` |
| 数据数据库 | `data.db`（内存缓冲 + 磁盘归档） |
| 监听地址 | `:8080` |
| 默认用户 | `admin` |
| 默认密码 | `123456` |

## 配置说明

### 资源配置

| 类型 | 类型字段 | 配置示例 |
|------|----------|----------|
| 串口 | `serial` | `/dev/ttyUSB0`, `COM1` |
| TCP | `tcp` | `192.168.1.100:502` |
| UDP | `udp` | `192.168.1.100:502` |

### 设备配置

| 参数 | 说明 |
|------|------|
| 名称 | 设备唯一标识 |
| 资源 | 关联的资源 |
| 驱动类型 | `modbus_rtu` / `modbus_tcp`（旧）或 `modbus_rtu_wasm` / `modbus_tcp_wasm` / `modbus_rtu_excel` / `modbus_tcp_excel` |
| 通信参数 | 波特率、数据位、停止位、校验位 / IP、端口 |
| 设备地址 | Modbus 从机地址 |
| 采集周期 | 毫秒 |
| 存储周期 | 秒（默认 300） |
| 使能 | 启用/禁用采集 |

### 北向配置

```json
{
    "type": "http",
    "config": {
        "url": "http://server/api/upload",
        "method": "POST"
    },
    "upload_interval": 5000
}
```

### 阈值配置

| 参数 | 说明 |
|------|------|
| 字段 | 数据字段名 |
| 条件 | `>`, `<`, `>=`, `<=`, `==`, `!=` |
| 阈值 | 数值 |
| 严重程度 | `info`, `warning`, `error`, `critical` |

## API 接口

> 所有 `/api` 路径需要 JWT 鉴权（`Authorization: Bearer <token>`）。

### 认证

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/login` | 登录页面 |
| POST | `/login` | 登录提交 |
| GET | `/logout` | 登出 |

### 页面

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/` | 仪表盘 |
| GET | `/alarms` | 报警日志 |
| GET | `/realtime` | 实时数据 |
| GET | `/gateway` | 网关设置 |
| GET | `/resources` | 资源管理 |
| GET | `/devices` | 设备管理 |
| GET | `/drivers` | 驱动管理 |
| GET | `/northbound` | 北向配置 |
| GET | `/storage` | 存储策略 |
| GET | `/thresholds` | 阈值配置 |

### 健康检查

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/health` | 健康检查 |
| GET | `/ready` | 就绪检查 |
| GET | `/live` | 存活检查 |
| GET | `/metrics` | 指标 |

### 系统

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/status` | 系统状态 |
| POST | `/api/collector/start` | 启动采集器 |
| POST | `/api/collector/stop` | 停止采集器 |

### 资源

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/resources` | 资源列表 |
| POST | `/api/resources` | 创建资源 |
| PUT | `/api/resources/{id}` | 更新资源 |
| DELETE | `/api/resources/{id}` | 删除资源 |
| POST | `/api/resources/{id}/toggle` | 启用/禁用 |

### 设备

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/devices` | 设备列表 |
| POST | `/api/devices` | 创建设备 |
| PUT | `/api/devices/{id}` | 更新设备 |
| DELETE | `/api/devices/{id}` | 删除设备 |
| POST | `/api/devices/{id}/toggle` | 切换状态 |
| POST | `/api/devices/{id}/execute` | 执行驱动函数（读/写） |
| GET | `/api/devices/{id}/writables` | 获取可写字段元数据 |

### 驱动

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/drivers` | 驱动列表 |
| GET | `/api/drivers/files` | 驱动文件列表 |
| POST | `/api/drivers` | 创建驱动 |
| PUT | `/api/drivers/{id}` | 更新驱动 |
| DELETE | `/api/drivers/{id}` | 删除驱动 |
| GET | `/api/drivers/{id}/runtime` | 获取驱动运行态 |
| GET | `/api/drivers/runtime` | 获取所有驱动运行态 |
| POST | `/api/drivers/{id}/reload` | 重载驱动 |
| POST | `/api/drivers/upload` | 上传 WASM 文件 |
| GET | `/api/drivers/{id}/download` | 下载驱动文件 |

### 北向

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/northbound` | 北向配置列表 |
| GET | `/api/northbound/status` | 北向运行状态列表 |
| GET | `/api/northbound/schema?type=xunji` | 获取 XUNJI 配置 Schema（前端动态表单） |
| POST | `/api/northbound` | 创建北向配置 |
| PUT | `/api/northbound/{id}` | 更新配置 |
| DELETE | `/api/northbound/{id}` | 删除配置 |
| POST | `/api/northbound/{id}/toggle` | 启用/禁用 |
| POST | `/api/northbound/{id}/reload` | 重载单个北向运行时 |

### 报警

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/alarms` | 报警日志 |
| POST | `/api/alarms/{id}/acknowledge` | 确认报警 |

### 数据

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/data` | 缓存概览 |
| GET | `/api/data/cache/{id}` | 设备缓存 |
| GET | `/api/data/history` | 历史数据 |

### 数据存储说明

- `data_cache` 存在 `data.db` 的**内存库**中，保存每个设备字段的最新值。
- `data_points` 先写入内存数据库，达到阈值或周期触发时**增量同步**到磁盘 `data.db`。
- 同步到磁盘后，内存中已同步的数据会被清理，减少内存占用和磁盘频繁写入。
- 历史查询会**合并内存 + 磁盘**数据，确保读取完整历史。

### 缓存流程图

```
采集器采集数据
        │
        ├─> 更新 data_cache（内存，最新值）
        │
        └─> 是否到达 storage_interval？
                │
                ├─ 否 → 仅更新缓存
                │
                └─ 是 → 写入 data_points（内存历史）
                         │
                         ├─ 达到 SyncInterval / SyncBatchTrigger
                         │      └─ 增量同步到磁盘 data.db
                         │             └─ 清理内存中已同步数据
                         │
                         └─ 历史查询：合并内存 + 磁盘返回
```

### 存储

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/storage` | 存储策略列表 |
| POST | `/api/storage` | 创建存储策略 |
| PUT | `/api/storage/{id}` | 更新存储策略 |
| DELETE | `/api/storage/{id}` | 删除存储策略 |
| POST | `/api/storage/run` | 按策略清理 |
| POST | `/api/storage/cleanup` | 手动清理 |

### 阈值

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/thresholds` | 阈值列表 |
| POST | `/api/thresholds` | 创建阈值 |
| PUT | `/api/thresholds/{id}` | 更新阈值 |
| DELETE | `/api/thresholds/{id}` | 删除阈值 |

### 用户

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/users` | 用户列表 |
| POST | `/api/users` | 创建用户 |
| PUT | `/api/users/{id}` | 更新用户 |
| DELETE | `/api/users/{id}` | 删除用户 |
| PUT | `/api/users/password` | 修改密码 |

### 网关配置

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/gateway/config` | 获取网关配置 |
| PUT | `/api/gateway/config` | 更新网关配置 |
| POST | `/api/gateway/northbound/sync-identity` | 将网关 ProductKey/DeviceKey 同步到 xunji 北向配置 |

## 驱动开发

网关使用 **Extism + TinyGo** 框架实现插件式设备驱动，所有驱动以 WASM 形式运行。

### 驱动架构

```
┌─────────────────────────────────────────────────────────────────┐
│                        TinyGo 驱动 (WASM)                          │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│   ┌─────────────────────────────────────────────────────────┐   │
│   │  【用户修改】点表定义                                      │   │
│   │  const (REG_TEMPERATURE = 0x0000, ...)                   │   │
│   └─────────────────────────────────────────────────────────┘   │
│                                                                  │
│   ┌─────────────────────────────────────────────────────────┐   │
│   │  【用户修改】点表配置                                      │   │
│   │  var pointConfig = []PointConfig{...}                    │   │
│   └─────────────────────────────────────────────────────────┘   │
│                                                                  │
│   ┌─────────────────────────────────────────────────────────┐   │
│   │  【用户修改】读取/写入逻辑                                  │   │
│   │  func readAllPoints() {...}                             │   │
│   │  func doWrite() {...}                                   │   │
│   └─────────────────────────────────────────────────────────┘   │
│                                                                  │
│   ┌─────────────────────────────────────────────────────────┐   │
│   │  【固定不变】Modbus 通信函数                               │   │
│   │  serial_transceive / tcp_transceive                    │   │
│   │  buildReadFrame / parseReadResponse / crc16            │   │
│   └─────────────────────────────────────────────────────────┘   │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Extism WASM Runtime                           │
│           (serial_transceive / tcp_transceive)                  │
└─────────────────────────────────────────────────────────────────┘
```

### 代码结构规范

每个驱动文件分为以下清晰的部分：

| 区域 | 标记 | 说明 | 是否需要修改 |
|------|------|------|------------|
| Host 函数声明 | `【固定不变】` | 串口/网络通信接口 | 否 |
| 配置结构 | `【固定不变】` | DriverConfig | 否 |
| 点表定义 | `【用户修改】` | 寄存器地址常量 | **是** |
| 点表配置 | `【用户修改】` | pointConfig 数组 | **是** |
| 驱动入口 | `【固定不变】` | handle(), describe() | 否 |
| 读取逻辑 | `【用户修改】` | readAllPoints() | **是** |
| 写入逻辑 | `【用户修改】` | doWrite() | **是** |
| 通信函数 | `【固定不变】` | Modbus 收发、帧构建、CRC | 否 |
| 工具函数 | `【固定不变】` | 配置获取、格式化输出 | 否 |

### 快速开始

```bash
# 进入驱动目录
cd drvs

# 编译所有驱动
make

# 编译单个驱动
tinygo build -o th_modbusrtu.wasm -target=wasip1 -buildmode=c-shared ./th_modbusrtu.go
tinygo build -o th_modbustcp.wasm -target=wasip1 -buildmode=c-shared ./th_modbustcp.go
tinygo build -o ups_kstar.wasm -target=wasip1 -buildmode=c-shared ./ups_kstar.go
```

### 内置驱动

| 驱动文件 | 协议 | 说明 | 文档 |
|---------|------|------|------|
| `th_modbusrtu.go` | Modbus RTU | 温湿度传感器 RTU 驱动 | [README_th_modbusrtu.md](./drvs/README_th_modbusrtu.md) |
| `th_modbustcp.go` | Modbus TCP | 温湿度传感器 TCP 驱动 | [README_th_modbustcp.md](./drvs/README_th_modbustcp.md) |
| `temperature_humidity.go` | Modbus RTU | 温湿度传感器通用驱动 | [README_temhumidity.md](./drvs/README_temhumidity.md) |
| `ups_kstar.go` | Modbus TCP | 科士达 UPS 驱动 | [README_ups_kstar.md](./drvs/README_ups_kstar.md) |

### 编写新驱动

参考以下模板创建新驱动：

```go
// =============================================================================
// [设备名称] - [协议] 驱动
// =============================================================================
//
// 设备点表:
//   - [字段1]: FC=03, 地址=0x0000, 长度=1, 缩放=0.1, 1位小数
//   - [字段2]: FC=03, 地址=0x0001, 长度=1, 缩放=1, 0位小数
//
// Host 提供: serial_transceive / tcp_transceive
//
// =============================================================================
package main

import (
	"encoding/json"
	"strconv"
	"strings"

	pdk "github.com/extism/go-pdk"
)

// =============================================================================
// 【固定不变】Host 函数声明
// =============================================================================
//go:wasmimport extism:host/user serial_transceive  // 或 tcp_transceive
func serial_transceive(wPtr uint64, wSize uint64, rPtr uint64, rCap uint64, timeoutMs uint64) uint64

// =============================================================================
// 【固定不变】配置结构
// =============================================================================
type DriverConfig struct {
	DeviceAddress int    `json:"device_address"`
	FuncName      string `json:"func_name"`
	FieldName     string `json:"field_name"`
	Value         string `json:"value"`
}

// =============================================================================
// 【用户修改】点表定义
// =============================================================================
const (
	REG_POINT1 = 0x0000 // 寄存器1
	REG_POINT2 = 0x0001 // 寄存器2

	FUNC_CODE_READ  = 0x03
	FUNC_CODE_WRITE = 0x06
)

// =============================================================================
// 【用户修改】点表配置
// =============================================================================
var pointConfig = []PointConfig{
	{Field: "point1", Address: REG_POINT1, Length: 1, Scale: 0.1, Decimals: 1, RW: "R", Unit: "unit", Label: "字段1"},
	{Field: "point2", Address: REG_POINT2, Length: 1, Scale: 1, Decimals: 0, RW: "R", Unit: "unit", Label: "字段2"},
}

type PointConfig struct {
	Field    string
	Address  uint16
	Length   uint16
	Scale    float64
	Decimals int
	RW       string
	Unit     string
	Label    string
}

// =============================================================================
// 【固定不变】驱动入口
// =============================================================================
//go:wasmexport handle
func handle() int32 {
	cfg := getConfig()
	points := readAllPoints(cfg.DeviceAddress)
	outputJSON(map[string]interface{}{"success": true, "points": points})
	return 0
}

// =============================================================================
// 【固定不变】描述可写字段
// =============================================================================
//go:wasmexport describe
func describe() int32 {
	outputJSON(map[string]interface{}{"success": true, "data": map[string]string{}})
	return 0
}

// =============================================================================
// 【用户修改】读取所有测点
// =============================================================================
func readAllPoints(devAddr int) []map[string]interface{} {
	// 根据 pointConfig 批量读取寄存器并转换为实际值
	...
}

// =============================================================================
// 【固定不变】Modbus 通信函数 (参考现有驱动)
// =============================================================================
func serialTransceive(...) {...}
func buildReadFrame(...) {...}
func parseReadResponse(...) {...}
func crc16(...) {...}

// =============================================================================
// 【固定不变】工具函数
// =============================================================================
func getConfig() DriverConfig {...}
func formatFloat(val float64, decimals int) string {...}
func outputJSON(v interface{}) {...}
```

详细开发指南请参考 [drvs/README.md](./drvs/README.md)

## 数据库架构

### param.db (配置数据库)

| 表名 | 说明 |
|------|------|
| users | 用户表 |
| resources | 资源表 |
| devices | 设备表 |
| drivers | 驱动表 |
| northbound_configs | 北向配置表 |
| thresholds | 阈值配置表 |
| alarm_logs | 报警日志表 |
| storage_configs | 存储配置表 |

### data.db (历史数据数据库)

| 表名 | 说明 |
|------|------|
| data_points | 采集数据点 |

**同步策略**: 内存模式 + 每 5 分钟持久化到磁盘

## 资源访问控制

- 同一资源（串口/TCP）同一时间只允许一个设备访问
- 使用互斥锁防止并发读取导致数据乱码
- 支持超时等待机制

## 系统监控

### 状态指标

- 采集器运行状态
- 设备总数 / 已启用数量
- 北向配置总数 / 已启用数量
- 报警总数 / 待处理数量

### 日志类型

- HTTP 请求日志
- 采集数据日志
- 报警触发日志
- 北向发送日志

## 许可证

MIT License

## 作者

[gonglijing](https://github.com/gonglijing)

## 致谢

- [Go](https://golang.org/) - 程序语言
- [Gorilla Mux](https://github.com/gorilla/mux) - Web 路由
- [SolidJS](https://www.solidjs.com/) - 前端框架
- [Vite](https://vitejs.dev/) - 构建工具
- [Extism](https://extism.org/) - WASM 插件框架
- [TinyGo](https://tinygo.org/) - Go 编译器
- [SQLite](https://www.sqlite.org/) - 数据库
