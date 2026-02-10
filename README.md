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
│   ┌────────────────────────────────────────────────────────────────┐   │
│   │                     北向适配器 (内置)                            │   │
│   │   ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐          │   │
│   │   │  Sagoo  │  │ PandaX  │  │  MQTT   │  │  HTTP   │          │   │
│   │   └─────────┘  └─────────┘  └─────────┘  └─────────┘          │   │
│   └────────────────────────────────────────────────────────────────┘   │
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
| **北向接口** | 内置 Sagoo / PandaX / MQTT / HTTP 适配器 |

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
| 前端框架 | SolidJS | 1.8+ |
| 构建工具 | Vite | 5.x |
| 驱动运行时 | Extism | go-sdk |
| 驱动开发 | TinyGo | 0.34+ |
| 北向适配器 | 内置 | Sagoo / PandaX / MQTT / HTTP |
| 认证 | JWT | 鉴权中间件 |
| 跨平台 | CGO=0 | arm32 / arm64 / darwin / windows |

## 项目结构

```
fsu/
├── cmd/
│   └── main.go                 # 程序入口，配置加载
├── internal/
│   ├── app/
│   │   ├── app.go             # 启动逻辑、优雅关闭
│   │   ├── db.go              # 数据库初始化
│   │   ├── http.go            # HTTP 服务器构建
│   │   ├── northbound.go       # 北向调度器
│   │   └── secret.go          # JWT 密钥管理
│   ├── auth/
│   │   └── auth.go            # JWT 认证中间件
│   ├── collector/
│   │   ├── collector.go       # 采集调度器（优先队列）
│   │   ├── system_collector.go # 系统监控采集
│   │   └── threshold_cache.go  # 阈值缓存
│   ├── database/
│   │   ├── database.go         # 双数据库管理
│   │   ├── device.go          # 设备 CRUD
│   │   ├── health.go          # 健康检查
│   │   └── resource.go        # 资源 CRUD
│   ├── driver/
│   │   └── manager.go         # 驱动加载/执行/Host Functions
│   ├── handlers/
│   │   ├── auth.go            # 登录/登出
│   │   ├── data.go            # 数据查询
│   │   ├── device.go          # 设备管理
│   │   ├── driver.go          # 驱动管理
│   │   ├── handlers.go        # Handler 初始化
│   │   ├── northbound.go      # 北向配置
│   │   ├── pages.go           # 页面路由
│   │   ├── resource.go        # 资源管理
│   │   ├── response.go        # 统一响应
│   │   └── user.go            # 用户管理
│   ├── models/
│   │   └── models.go          # 数据模型定义
│   ├── northbound/
│   │   ├── adapters/          # 北向适配器
│   │   │   ├── sagoo.go      # Sagoo (原 XunJi) 适配器
│   │   │   ├── pandax.go     # PandaX 适配器
│   │   │   ├── mqtt.go       # MQTT 适配器
│   │   │   └── http.go       # HTTP 适配器
│   │   ├── manager.go        # 北向调度与熔断
│   │   └── schema/           # 北向配置 Schema
│   ├── resource/
│   │   └── manager.go        # 串口/TCP 连接管理
│   └── logger/
│       └── logger.go          # 日志封装
├── ui/
│   ├── frontend/              # SolidJS 前端
│   │   ├── src/
│   │   │   ├── api.js        # HTTP 客户端基础能力
│   │   │   ├── api/          # 领域 API 封装
│   │   │   │   └── *.js      # devices/northbound/...
│   │   │   ├── router.jsx    # 路由管理
│   │   │   ├── main.jsx      # 入口文件
│   │   │   ├── App.jsx       # 根组件
│   │   │   ├── components/   # UI 组件
│   │   │   ├── pages/        # 页面
│   │   │   └── sections/    # 页面区块
│   │   └── package.json
│   └── static/
│       └── dist/             # 构建后的静态资源
├── drvs/                     # TinyGo 驱动源码
│   ├── README.md             # 驱动开发指南
│   ├── air_conditioning/     # 空调驱动
│   ├── ups/                  # UPS 驱动
│   ├── electric_meter/       # 电表驱动
│   ├── temperature_humidity/ # 温湿度驱动
│   ├── water_leak/           # 漏水驱动
│   └── cabinet_header/       # 机柜 header 驱动
├── migrations/
│   └── *.sql                 # 数据库迁移脚本
├── Makefile                  # 编译脚本
├── config/
│   └── config.yaml           # 配置文件
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

# 运行
./fsu

# 或使用 make 运行
make run
```

### 北向配置

FSU 内置以下北向适配器：

| 类型 | 说明 | 用途 |
|------|------|------|
| **sagoo** | SagooIoT 平台 | 寻迹平台对接 |
| **pandax** | PandaX 平台 | PandaX 平台对接 |
| **mqtt** | 标准 MQTT | 通用 MQTT Broker |
| **http** | HTTP POST | 第三方 HTTP 服务 |

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

## 北向适配器

### Sagoo (原 XunJi) 适配器

用于对接 SagooIoT 平台，上报设备数据并接收平台命令。

**配置参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| productKey | string | 是 | 网关 ProductKey |
| deviceKey | string | 是 | 网关 DeviceKey |
| serverUrl | string | 是 | MQTT 地址，如 `tcp://192.168.1.100:1883` |
| username | string | 否 | MQTT 用户名 |
| password | string | 否 | MQTT 密码 |
| uploadIntervalMs | int | 否 | 上报周期（毫秒），默认 5000 |

**说明：**
- `productKey` 和 `deviceKey` 用于网关系统属性上报
- 子设备数据通过网关批量上报

### PandaX 适配器

用于对接 PandaX 平台，支持 TDengine 时序数据库。

**配置参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| serverUrl | string | 是 | TDengine REST API 地址 |
| username | string | 是 | TDengine 用户名 |
| password | string | 是 | TDengine 密码 |

### MQTT 适配器

通用 MQTT 客户端，用于对接任意 MQTT Broker。

**配置参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| broker | string | 是 | MQTT Broker 地址 |
| topic | string | 是 | 数据上报 Topic |
| clientId | string | 否 | 客户端 ID |
| username | string | 否 | 用户名 |
| password | string | 否 | 密码 |
| qos | int | 否 | QoS 等级 (0-2)，默认 0 |
| retain | bool | 否 | Retain 标记，默认 false |
| keepAlive | int | 否 | 心跳周期（秒），默认 60 |

### HTTP 适配器

HTTP POST 推送适配器。

**配置参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| url | string | 是 | HTTP 端点地址 |
| method | string | 否 | HTTP 方法，默认 POST |
| headers | object | 否 | 请求头 |
| timeout | int | 否 | 超时时间（秒），默认 30 |

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
| GET | `/` | 拓扑视图（首页） |
| GET | `/topology` | 拓扑视图 |
| GET | `/alarms` | 报警日志 |
| GET | `/realtime` | 实时数据 |
| GET | `/gateway` | 网关设置 |
| GET | `/resources` | 资源管理 |
| GET | `/devices` | 设备管理 |
| GET | `/drivers` | 驱动管理 |
| GET | `/northbound` | 北向配置 |

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

### 驱动

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/drivers` | 驱动列表 |
| GET | `/api/drivers/files` | 驱动文件列表 |
| POST | `/api/drivers` | 创建驱动 |
| PUT | `/api/drivers/{id}` | 更新驱动 |
| DELETE | `/api/drivers/{id}` | 删除驱动 |
| GET | `/api/drivers/{id}/runtime` | 获取驱动运行态 |
| POST | `/api/drivers/{id}/reload` | 重载驱动 |
| POST | `/api/drivers/upload` | 上传 WASM 文件 |

### 北向

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/northbound` | 北向配置列表 |
| GET | `/api/northbound/status` | 北向运行状态列表 |
| GET | `/api/northbound/schema?type=xunji` | 获取 Sagoo 配置 Schema |
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
| GET | `/api/data/cache/{id}` | 设备缓存 |
| GET | `/api/data/history` | 历史数据 |

### 网关配置

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/gateway/config` | 获取网关配置 |
| PUT | `/api/gateway/config` | 更新网关配置 |

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

### Host Functions

驱动可通过以下 Host Functions 与主程序交互：

| 函数 | 说明 |
|------|------|
| `serial_transceive(port, baudrate, data)` | 串口收发 |
| `tcp_transceive(host, port, data)` | TCP 收发 |
| `getDeviceConfig(key)` | 获取设备配置 |
| `setValue(key, value)` | 设置采集值 |

### 驱动目录结构

```
drvs/
├── air_conditioning/     # 空调驱动
│   ├── Makefile
│   ├── README.md
│   └── *.go
├── ups/                  # UPS 驱动
├── electric_meter/       # 电表驱动
├── temperature_humidity/ # 温湿度驱动
├── water_leak/           # 漏水驱动
└── cabinet_header/       # 机柜 header 驱动
```

### 编译驱动

```bash
# 编译所有驱动
cd drvs && make

# 编译单个驱动
cd drvs/air_conditioning && make
```

## 数据库架构

### param.db (配置数据库)

| 表名 | 说明 |
|------|------|
| gateway_config | 网关配置 |
| resources | 资源（串口/TCP） |
| devices | 设备 |
| drivers | 驱动 |
| northbound_configs | 北向配置 |
| thresholds | 阈值配置 |
| users | 用户 |

### data.db (数据数据库)

| 表名 | 说明 |
|------|------|
| data_cache | 设备最新值缓存（内存表） |
| data_points | 历史数据点（归档表） |

### 数据存储策略

- **data_cache**: 存在内存数据库中，保存每个设备字段的最新值
- **data_points**: 先写入内存，达到阈值或周期触发时增量同步到磁盘
- **历史查询**: 自动合并内存 + 磁盘数据

## 系统监控

### 监控指标

| 指标 | 说明 |
|------|------|
| CPU 使用率 | 系统 CPU 使用百分比 |
| 内存使用率 | 内存使用百分比 |
| 磁盘使用率 | 磁盘使用百分比 |
| 运行时间 | 网关运行时长 |
| 负载均值 | 1/5/15 分钟负载 |

### 监控数据

- 采集周期：1 分钟
- 数据保留：与网关数据一致
- 上报方式：北向适配器自动上报

### 拓扑视图展示

- 左侧按资源分组展示设备
- 中央展示网关节点与基础状态
- 右侧展示北向通道与启停状态
- 支持点击设备查看详情抽屉

### 日志类型

- HTTP 请求日志
- 采集数据日志
- 报警触发日志
- 北向发送日志

## 更新日志

### v1.2.0 (2026-02)

#### 架构优化
- 移除 gRPC 北向插件，改为内置适配器
- 新增 PandaX 北向适配器，支持 TDengine
- 重构北向调度器，支持热重载
- 驱动目录独立为 Git 子模块

#### 功能变更
- **移除** "同步网关身份" 功能
- `productKey`/`deviceKey` 仅用于 Sagoo 北向
- 网关系统属性自动上报

### v1.1.0 (2026-02-10)

#### 移除功能
- 移除"同步网关身份"功能（原 `POST /api/gateway/northbound/sync-identity`）
- 北向配置中的 `productKey`/`deviceKey` 现在由系统在启动时自动从网关配置填充，无需手动同步

#### 优化
- 简化北向配置管理，去除冗余操作步骤

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
