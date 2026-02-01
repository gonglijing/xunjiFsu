# 循迹智联网关

一个基于Go语言开发的工业网关管理系统，支持串口/网口资源配置、设备驱动管理、数据采集、阈值报警和北向接口对接。

## 功能特性

### 核心功能

- **资源配置管理**: 支持串口(Serial)、TCP/UDP网络资源配置
- **设备管理**: 一个资源可对接多个设备，设备级通信参数配置
- **驱动管理**: 基于Extism + TinyGo的WASM驱动支持，支持上传/下载
- **数据采集**: 支持独立设置采集周期、上传周期和存储周期
- **阈值报警**: 采集数据与阈值对比，触发报警时调用北向接口
- **双数据库架构**: param.db(配置) + data.db(历史数据，内存模式+定期持久化)

### 北向接口

- 支持多种北向接口类型（XunJi、HTTP、MQTT）
- 每个北向接口可单独设置上报周期
- XunJi接口支持批量属性上报和事件上报
- ProductKey/DeviceKey 可配置

### Web管理界面

- **HTMX + Go Template**: 现代化的服务端渲染前端架构
- **零外部依赖**: HTMX CDN，可完全离线运行
- **科技感UI**: 深色主题、霓虹发光效果、平滑动画
- **响应式设计**: 适配不同屏幕尺寸
- **动态交互**: 无刷新页面更新，局部内容自动刷新
- 实时状态监控和历史数据查询
- 支持登录认证和密码管理
- 支持使能/禁用设备采集

## 技术栈

- **后端**: Go 1.21+
- **数据库**: SQLite (纯Go实现，modernc.org/sqlite，无cgo依赖)
- **Web框架**: Gorilla Mux
- **前端**: HTMX + Go html/template + CSS3
- **驱动**: Extism + TinyGo
- **认证**: gorilla/sessions + bcrypt
- **跨平台**: 支持 arm32、arm64、darwin、windows

### 前端技术优势

| 特性 | 说明 |
|------|------|
| 服务端渲染 | Go template 直接渲染 HTML 片段 |
| HTMX 交互 | 无需编写 JavaScript 实现动态交互 |
| 轻量级 | 首屏加载快，HTMX 压缩后仅 ~14KB |
| 离线支持 | 可替换为本地 HTMX 文件实现完全离线 |

## 安装与运行

### 环境要求

- Go 1.21 或更高版本
- 无其他依赖（纯Go实现）

### 编译运行

```bash
# 克隆项目
git clone https://github.com/gonglijing/xunjiFsu.git
cd xunjiFsu

# 下载依赖
go mod download

# 编译（当前平台）
go build -o gogw ./cmd/main.go

# 或使用Makefile编译所有平台
make help
make build

# 运行
./gogw

# 指定参数
./gogw --db=param.db --addr=:8080
```

### Docker 部署（可选）

```bash
docker build -t xunji-gateway .
docker run -d -p 8080:8080 -v $(pwd)/data:/app/data xunji-gateway
```

### 默认配置

- 配置数据库: `param.db`
- 数据数据库: `data.db`
- 监听地址: `:8080`
- 默认用户: `admin`
- 默认密码: `123456`

## 配置说明

### 资源配置

支持三种类型的资源：

1. **串口资源 (Serial)**
   - 名称: 唯一标识
   - 类型: serial
   - 串口: 如 /dev/ttyUSB0、COM1

2. **TCP连接 (TCP)**
   - 名称: 唯一标识
   - 类型: tcp
   - 配置: IP:端口，如 192.168.1.100:502

3. **UDP连接 (UDP)**
   - 名称: 唯一标识
   - 类型: udp
   - 配置: IP:端口

### 设备配置

设备关联资源、通信参数和驱动：

| 参数 | 说明 |
|------|------|
| 名称 | 设备唯一标识 |
| 资源 | 关联的资源 |
| 驱动 | 关联的驱动文件 (.wasm) |
| 通信参数 | 波特率、数据位、停止位、校验位、IP、端口、协议 |
| 采集周期 | 毫秒 |
| 上传周期 | 毫秒 |
| 使能 | 启用/禁用采集 |

### 北向配置

XunJi配置示例:

```json
{
    "productKey": "your-product-key",
    "deviceKey": "your-device-key",
    "serverUrl": "mqtt://broker.example.com:1883",
    "username": "mqtt-username",
    "password": "mqtt-password"
}
```

### 存储配置

存储周期配置（存储时长控制data.db历史数据保留时间）：

| 参数 | 说明 |
|------|------|
| ProductKey | 产品标识 |
| DeviceKey | 设备标识 |
| 存储天数 | 历史数据保留天数 |
| 使能 | 启用/禁用 |

### 阈值配置

阈值规则：

| 参数 | 说明 |
|------|------|
| 字段 | 数据字段名 |
| 条件 | >, <, >=, <=, ==, != |
| 阈值 | 数值 |
| 严重程度 | info, warning, error, critical |

## API接口

### 认证接口

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/login` | 登录页面 |
| POST | `/login` | 登录提交 |
| GET | `/logout` | 登出 |

### 页面接口

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/` | 仪表盘 |
| GET | `/realtime` | 实时数据 |
| GET | `/history` | 历史数据 |

### 状态接口

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/status` | 系统状态 (HTMX 片段) |

### 资源接口

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/resources` | 获取资源列表 (支持 HTMX) |
| POST | `/api/resources` | 创建资源 |
| PUT | `/api/resources/{id}` | 更新资源 |
| DELETE | `/api/resources/{id}` | 删除资源 |
| POST | `/api/resources/{id}/open` | 打开资源 |
| POST | `/api/resources/{id}/close` | 关闭资源 |

### 设备接口

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/devices` | 获取设备列表 (支持 HTMX) |
| POST | `/api/devices` | 创建设备 |
| PUT | `/api/devices/{id}` | 更新设备 |
| DELETE | `/api/devices/{id}` | 删除设备 |
| POST | `/api/devices/{id}/toggle` | 切换使能状态 |

### 驱动接口

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/drivers` | 获取驱动列表 (支持 HTMX) |
| POST | `/api/drivers` | 创建驱动 |
| PUT | `/api/drivers/{id}` | 更新驱动 |
| DELETE | `/api/drivers/{id}` | 删除驱动 |
| POST | `/api/drivers/upload` | 上传驱动文件 (.wasm) |
| GET | `/api/drivers/{id}/download` | 下载驱动文件 |
| GET | `/api/drivers/files` | 列出驱动文件 |

### 北向配置接口

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/northbound` | 获取北向配置 (支持 HTMX) |
| POST | `/api/northbound` | 创建北向配置 |
| PUT | `/api/northbound/{id}` | 更新北向配置 |
| DELETE | `/api/northbound/{id}` | 删除北向配置 |
| POST | `/api/northbound/{id}/toggle` | 切换使能状态 |

### 阈值接口

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/thresholds` | 获取阈值列表 (支持 HTMX) |
| POST | `/api/thresholds` | 创建阈值 |
| PUT | `/api/thresholds/{id}` | 更新阈值 |
| DELETE | `/api/thresholds/{id}` | 删除阈值 |

### 报警接口

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/alarms` | 获取报警日志 (支持 HTMX) |
| POST | `/api/alarms/{id}/acknowledge` | 确认报警 |

### 数据接口

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/data` | 获取数据缓存 |
| GET | `/api/data/cache/{id}` | 获取设备数据缓存 |
| GET | `/api/data/points/{id}` | 获取历史数据点 |
| GET | `/api/data/points` | 获取最新数据点 (支持 HTMX) |
| GET | `/api/data/history` | 获取历史数据 (支持 HTMX) |

### 存储配置接口

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/storage` | 获取存储配置 |
| POST | `/api/storage` | 创建存储配置 |
| PUT | `/api/storage/{id}` | 更新存储配置 |
| DELETE | `/api/storage/{id}` | 删除存储配置 |
| POST | `/api/storage/cleanup` | 清理过期数据 |

### 用户接口

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/users` | 获取用户列表 |
| POST | `/api/users` | 创建用户 |
| PUT | `/api/users/{id}` | 更新用户 |
| DELETE | `/api/users/{id}` | 删除用户 |
| PUT | `/api/users/password` | 修改密码 |

### 采集控制接口

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/collector/start` | 启动采集器 |
| POST | `/api/collector/stop` | 停止采集器 |

## 项目结构

```
gogw/
├── cmd/
│   └── main.go                 # 主程序入口
├── internal/
│   ├── auth/                   # 认证模块
│   │   └── auth.go             # 登录/会话管理
│   ├── collector/              # 数据采集模块
│   │   └── collector.go        # 采集调度器
│   ├── database/               # 数据库操作
│   │   └── database.go         # 双数据库管理
│   ├── driver/                 # 驱动管理
│   │   └── manager.go          # 驱动加载/执行
│   ├── handlers/               # Web处理器（模块化）
│   │   ├── auth.go             # 认证处理
│   │   ├── data.go             # 数据/存储处理
│   │   ├── device.go           # 设备处理
│   │   ├── driver.go           # 驱动处理
│   │   ├── handlers.go         # Handler结构体
│   │   ├── northbound.go       # 北向配置处理
│   │   ├── pages.go            # 页面路由 + 模板渲染
│   │   ├── resource.go         # 资源处理
│   │   ├── response.go         # 统一响应格式
│   │   └── user.go             # 用户处理
│   ├── models/                 # 数据模型
│   │   └── models.go           # 所有数据模型
│   ├── northbound/             # 北向接口
│   │   └── manager.go          # XunJi/HTTP/MQTT适配器
│   ├── pwdutil/                # 密码工具
│   │   └── password.go         # bcrypt加密
│   └── resource/               # 资源管理
│       └── manager.go          # 串口/网口管理
├── migrations/
│   ├── 001_init.sql            # 初始化SQL
│   ├── 002_param_schema.sql    # 配置库schema
│   └── 003_data_schema.sql     # 数据库schema
├── templates/                  # HTMX 模板
│   ├── base.html               # 基础布局
│   ├── dashboard.html          # 仪表盘
│   ├── login.html              # 登录页
│   ├── realtime.html           # 实时数据
│   ├── history.html            # 历史数据
│   ├── nav.html                # 导航组件
│   ├── scripts.html            # 脚本组件
│   └── fragments/              # HTMX 片段
│       ├── status-cards.html   # 状态卡片
│       ├── resources.html      # 资源列表
│       ├── devices.html        # 设备列表
│       ├── drivers.html        # 驱动列表
│       ├── northbound.html     # 北向配置
│       ├── thresholds.html     # 阈值配置
│       ├── alarms.html         # 报警日志
│       ├── realtime.html       # 实时数据片段
│       └── history.html        # 历史数据片段
├── drvs/                      # WASM 驱动源文件 (TinyGo)
├── deploy/                     # 部署包
│   ├── arm32/
│   ├── arm64/
│   ├── darwin/
│   └── windows/
├── Makefile                    # 编译脚本
├── config/
│   └── config.yaml             # 配置文件
├── go.mod
└── README.md
```

## HTMX 集成说明

### 工作原理

1. 页面首次加载时，Go handler 渲染完整 HTML 页面
2. HTMX 通过 `hx-get`、`hx-post` 等属性发起异步请求
3. 服务器返回 HTML 片段（而非 JSON），HTMX 替换指定区域
4. 无需编写任何客户端 JavaScript 代码

### 示例

```html
<!-- 自动刷新状态卡片 -->
<div hx-get="/api/status" hx-trigger="load, every 5s" hx-target="this">
    <!-- 内容由服务器渲染 -->
</div>

<!-- 删除操作 -->
<button hx-delete="/api/resources/1" hx-target="closest tr" hx-confirm="确定删除?">
    删除
</button>
```

## 驱动开发

### 架构原理

网关采用 **Extism + TinyGo** 实现插件式驱动架构：

```
┌─────────────────────────────────────────────────────────────────────┐
│                          Go 网关主程序 (Host)                         │
│  ┌─────────────────────────────────────────────────────────────────┐ │
│  │  DriverManager (驱动管理器)                                      │ │
│  │  ├── LoadDriver()    // 加载 WASM 文件 + 注册 Host Functions    │ │
│  │  ├── ExecuteDriver() // 调用插件 "collect" 函数                 │ │
│  │  └── UnloadDriver()  // 卸载插件                                │ │
│  └─────────────────────────────────────────────────────────────────┘ │
│                              ▲                                       │
│                              │ Extism SDK (WASM Runtime)             │
│                              ▼                                       │
│  ┌─────────────────────────────────────────────────────────────────┐ │
│  │                    WASM 插件 (Plugin)                           │ │
│  │  temperature_humidity.wasm                                      │ │
│  │  ├── serial_read()   // Host 提供的串口读取函数                  │ │
│  │  ├── serial_write()  // Host 提供的串口写入函数                  │ │
│  │  ├── sleep_ms()      // Host 提供的延时函数                     │ │
│  │  └── output()        // Host 提供的日志输出函数                 │ │
│  └─────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────┘
```

### Host Functions 接口

TinyGo 驱动通过以下 Host Functions 与网关交互：

| 函数名 | 参数 | 返回 | 说明 |
|--------|------|------|------|
| `serial_read` | buf: pointer, size: i32 | i32 | 从串口读取数据，返回实际读取字节数 |
| `serial_write` | buf: pointer, size: i32 | i32 | 向串口写入数据，返回实际写入字节数 |
| `sleep_ms` | ms: i32 | - | 毫秒级延时 |
| `output` | ptr: pointer, size: i32 | - | 输出日志到网关控制台 |

### TinyGo 驱动示例

```go
package main

import (
	"encoding/binary"
	"fmt"
	"unsafe"
)

//go:export serial_read
func serial_read(buf unsafe.Pointer, size int32) int32

//go:export serial_write
func serial_write(buf unsafe.Pointer, size int32) int32

//go:export sleep_ms
func sleep_ms(ms int32)

//go:export output
func output(ptr unsafe.Pointer, size int32)

//export collect
func collect() {
	// 从配置获取资源ID
	config := getConfig()
	
	// 构建 Modbus 请求
	req := buildRequest(1, 0x03, 0, 1)
	
	// 发送请求
	serial_write(unsafe.Pointer(&req[0]), int32(len(req)))
	
	// 等待响应
	sleep_ms(150)
	
	// 读取响应
	resp := make([]byte, 64)
	n := serial_read(unsafe.Pointer(&resp[0]), int32(len(resp)))
	
	// 解析数据
	value := binary.BigEndian.Uint16(resp[3:5])
	
	// 输出结果
	outputJSON(true, float64(int16(value))/10.0)
}

type Config struct {
	ResourceID int64 `json:"resource_id"`
}

func getConfig() Config {
	return Config{ResourceID: 1}
}
```

### 编译命令

```bash
# macOS 安装 TinyGo
brew install tinygo

# 编译为 WASM
cd drvs
tinygo build -o temperature_humidity.wasm -target=wasi -stack-size=64k temperature_humidity.go
```

### 配置传递

驱动通过 `ConfigSchema` 字段获取配置，配置以 JSON 格式存储：

```json
{
    "resource_id": 1
}
```

### 驱动文件命名规范

- 驱动文件放在 `drvs/` 目录
- 文件名格式: `{驱动名}.wasm`
- 示例: `temperature_humidity.wasm`

## 数据库架构

### param.db (配置数据库)

存储配置信息，直接持久化到磁盘：

- users - 用户表
- resources - 资源表
- devices - 设备表
- drivers - 驱动表
- northbound_configs - 北向配置表
- thresholds - 阈值配置表
- alarm_logs - 报警日志表
- storage_configs - 存储配置表

### data.db (历史数据数据库)

存储采集数据，采用内存模式+定期持久化策略：

- data_points - 数据点表

**同步策略**: 每5分钟将内存数据批量写入磁盘

## 跨平台编译

使用Makefile进行跨平台编译：

```bash
# 查看帮助
make help

# 编译所有平台
make build

# 编译指定平台
make deploy-darwin   # macOS
make deploy-arm64    # Linux ARM64
make deploy-arm32    # Linux ARM32
make deploy-windows  # Windows

# 清理
make clean
```

## 资源访问控制

系统实现了资源访问串行化机制：

- 同一串口/网口资源在同一时间只允许一个设备访问
- 使用互斥锁防止并发读取导致数据乱码
- 支持30秒超时等待

## 系统监控

### 状态监控

- 采集器运行状态
- 设备总数/已启用数量
- 北向配置总数/已启用数量
- 报警总数/待处理数量

### 日志

- HTTP请求日志
- 采集数据日志
- 报警触发日志
- 北向发送日志

## 许可证

MIT License

## 作者

[gonglijing](https://github.com/gonglijing)

## 致谢

- [Gorilla Mux](https://github.com/gorilla/mux) - Web路由
- [HTMX](https://htmx.org/) - 动态HTML交换
- [modernc.org/sqlite](https://gitlab.com/cznic/sqlite) - 纯Go SQLite
- [Extism](https://extism.org/) - WASM插件框架 (go-sdk)
