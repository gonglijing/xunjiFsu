# HuShu智能网关

一个基于Go语言开发的工业网关管理系统，支持串口/网口资源配置、设备驱动管理、数据采集、阈值报警和北向接口对接。

## 功能特性

### 核心功能

- **资源配置管理**: 支持串口(Serial)、数字输入(DI)、数字输出(DO)资源配置
- **设备管理**: 一个串口可对接多个设备，设备级通信参数配置
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

- 基于htmx + TailwindCSS的现代化界面
- 实时状态监控和历史数据查询
- 支持登录认证和密码管理
- 支持使能/禁用设备采集

## 技术栈

- **后端**: Go 1.21+
- **数据库**: SQLite (纯Go实现，modernc.org/sqlite，无cgo依赖)
- **Web框架**: Gorilla Mux
- **前端**: htmx + TailwindCSS + Font Awesome
- **驱动**: Extism + TinyGo
- **认证**: gorilla/sessions + bcrypt
- **跨平台**: 支持 arm32、arm64、darwin、windows

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
docker build -t hushu-gateway .
docker run -d -p 8080:8080 -v $(pwd)/data:/app/data hushu-gateway
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

2. **数字输入 (DI)**
   - 名称: 唯一标识
   - 类型: di
   - 地址: 设备地址

3. **数字输出 (DO)**
   - 名称: 唯一标识
   - 类型: do
   - 地址: 设备地址

> **注意**: 波特率、数据位、停止位、校验位、IP地址、端口号等通信参数已移至设备级别配置。

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
| GET | `/api/status` | 系统状态 |

### 资源接口

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/resources` | 获取资源列表 |
| POST | `/api/resources` | 创建资源 |
| PUT | `/api/resources/{id}` | 更新资源 |
| DELETE | `/api/resources/{id}` | 删除资源 |
| POST | `/api/resources/{id}/open` | 打开资源 |
| POST | `/api/resources/{id}/close` | 关闭资源 |

### 设备接口

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/devices` | 获取设备列表 |
| POST | `/api/devices` | 创建设备 |
| PUT | `/api/devices/{id}` | 更新设备 |
| DELETE | `/api/devices/{id}` | 删除设备 |
| POST | `/api/devices/{id}/toggle` | 切换使能状态 |

### 驱动接口

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/drivers` | 获取驱动列表 |
| POST | `/api/drivers` | 创建驱动 |
| PUT | `/api/drivers/{id}` | 更新驱动 |
| DELETE | `/api/drivers/{id}` | 删除驱动 |
| POST | `/api/drivers/upload` | 上传驱动文件 (.wasm) |
| GET | `/api/drivers/{id}/download` | 下载驱动文件 |
| GET | `/api/drivers/files` | 列出驱动文件 |

### 北向配置接口

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/northbound` | 获取北向配置 |
| POST | `/api/northbound` | 创建北向配置 |
| PUT | `/api/northbound/{id}` | 更新北向配置 |
| DELETE | `/api/northbound/{id}` | 删除北向配置 |
| POST | `/api/northbound/{id}/toggle` | 切换使能状态 |

### 阈值接口

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/thresholds` | 获取阈值列表 |
| POST | `/api/thresholds` | 创建阈值 |
| PUT | `/api/thresholds/{id}` | 更新阈值 |
| DELETE | `/api/thresholds/{id}` | 删除阈值 |

### 报警接口

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/alarms` | 获取报警日志 |
| POST | `/api/alarms/{id}/acknowledge` | 确认报警 |

### 数据接口

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/data` | 获取数据缓存 |
| GET | `/api/data/cache/{id}` | 获取设备数据缓存 |
| GET | `/api/data/points/{id}` | 获取历史数据点 |
| GET | `/api/data/points` | 获取最新数据点 |

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
xunjiFsu/
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
│   │   ├── pages.go            # 页面路由
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
├── web/
│   ├── pages/                  # HTML页面
│   │   ├── dashboard.html      # 仪表盘
│   │   ├── login.html          # 登录页
│   │   ├── realtime.html       # 实时数据
│   │   └── history.html        # 历史数据
│   └── static/                 # 静态资源
├── drivers/                    # 驱动文件目录
├── deploy/                     # 部署包
│   ├── arm32/
│   ├── arm64/
│   ├── darwin/
│   └── windows/
├── Makefile                    # 编译脚本
├── config.yaml                 # 配置文件
├── go.mod
└── README.md
```

## 驱动开发

### TinyGo驱动模板

```go
package main

import (
	"github.com/extism/go-pdk"
)

func collect() {
	// 获取配置
	config := getConfig()
	
	// 读取数据（串口/网口通信）
	data := readData()
	
	// 返回数据
	returnData(data)
}

func main() {
	// 注册函数
	// extism-pdk 特定代码
}
```

### 驱动文件命名规范

- 驱动文件放在 `drivers/` 目录
- 文件名格式: `{驱动名}.wasm`
- 示例: `temperature.wasm`, `humidity.wasm`

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
- 已加载驱动数量
- 资源数量
- 北向适配器数量

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
- [modernc.org/sqlite](https://gitlab.com/cznic/sqlite) - 纯Go SQLite
- [htmx](https://htmx.org/) - 前端交互
- [TailwindCSS](https://tailwindcss.com/) - CSS框架
- [Extism](https://extism.org/) - WASM插件框架
