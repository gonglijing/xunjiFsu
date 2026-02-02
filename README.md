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
| **双数据库** | param.db (配置) + data.db (历史数据) |
| **北向接口** | HTTP / MQTT / XunJi 多协议适配 |

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
| 认证 | JWT | 鉴权中间件 |
| 跨平台 | CGO=0 | arm32 / arm64 / darwin / windows |

## 项目结构

```
gogw/
├── cmd/
│   └── main.go                 # 程序入口，配置加载
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
│   │   └── manager.go          # XunJi/HTTP/MQTT 适配器
│   ├── resource/
│   │   └── manager.go          # 串口/TCP 连接管理
│   └── logger/
│       └── logger.go           # 日志封装
├── web/
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
cd web/frontend && npm install && cd ../..

# 编译前端
make ui

# 编译后端（当前平台）
make build

# 运行
./gogw

# 或使用 make 运行
make run
```

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
| 数据数据库 | `data.db` |
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
| 驱动类型 | `modbus_rtu` / `modbus_tcp` |
| 通信参数 | 波特率、数据位、停止位、校验位 / IP、端口 |
| 设备地址 | Modbus 从机地址 |
| 采集周期 | 毫秒 |
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
| GET | `/realtime` | 实时数据 |
| GET | `/history` | 历史数据 |

### 资源

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/resources` | 资源列表 |
| POST | `/api/resources` | 创建资源 |
| PUT | `/api/resources/{id}` | 更新资源 |
| DELETE | `/api/resources/{id}` | 删除资源 |

### 设备

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/devices` | 设备列表 |
| POST | `/api/devices` | 创建设备 |
| PUT | `/api/devices/{id}` | 更新设备 |
| DELETE | `/api/devices/{id}` | 删除设备 |
| POST | `/api/devices/{id}/toggle` | 切换状态 |

### 驱动

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/drivers` | 驱动列表 |
| POST | `/api/drivers` | 创建驱动 |
| POST | `/api/drivers/upload` | 上传 WASM 文件 |
| GET | `/api/drivers/{id}/download` | 下载驱动文件 |

### 北向

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/northbound` | 北向配置列表 |
| POST | `/api/northbound` | 创建北向配置 |
| PUT | `/api/northbound/{id}` | 更新配置 |
| DELETE | `/api/northbound/{id}` | 删除配置 |

### 数据

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/data/points` | 最新数据点 |
| GET | `/api/data/history` | 历史数据 |

### 阈值

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/thresholds` | 阈值列表 |
| POST | `/api/thresholds` | 创建阈值 |
| DELETE | `/api/thresholds/{id}` | 删除阈值 |

## 驱动开发

详细驱动开发指南请参考 [drvs/README.md](./drvs/README.md)

### 快速示例

```go
package main

//go:export serial_transceive
func serial_transceive(wPtr unsafe.Pointer, wSize, rPtr, rCap, timeoutMs int32) int32

//go:export output
func output(ptr unsafe.Pointer, size int32)

//go:export handle
func handle() {
    cfg := getConfig()
    points := readDevice(cfg)
    outputJSON(map[string]interface{}{
        "success": true,
        "data": map[string]interface{}{"points": points},
    })
}
```

### 编译驱动

```bash
cd drvs
tinygo build -o th_modbusrtu.wasm -target=wasi -stack-size=64k th_modbusrtu.go
```

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
