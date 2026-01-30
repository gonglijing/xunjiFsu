# GoGW - 工业网关管理系统

一个基于Go语言开发的工业网关管理系统，支持串口/网口资源配置、设备驱动管理、数据采集、阈值报警和北向接口对接。

## 功能特性

### 核心功能
- **资源配置管理**: 支持串口(Serial)和网口(Network)资源配置
- **设备管理**: 一个串口/网口可对接多个设备
- **驱动管理**: 基于Extism + TinyGo的WASM驱动支持
- **数据采集**: 支持独立设置采集周期和上传周期
- **阈值报警**: 采集数据与阈值对比，触发报警时调用北向接口

### 北向接口
- 支持多种北向接口类型（XunJi、HTTP、MQTT）
- 每个北向接口可单独设置上报周期
- XunJi接口支持批量属性上报和事件上报

### Web管理界面
- 基于htmx + TailwindCSS的现代化界面
- 实时状态监控
- 支持登录认证和密码管理

## 技术栈

- **后端**: Go 1.21+
- **数据库**: SQLite (纯Go实现，无cgo依赖)
- **Web框架**: Gorilla Mux
- **前端**: htmx + TailwindCSS
- **驱动**: Extism + TinyGo
- **认证**: gorilla/sessions + bcrypt

## 安装与运行

### 环境要求
- Go 1.21 或更高版本
- SQLite3

### 编译运行

```bash
# 克隆项目
git clone <repository-url>
cd gogw

# 下载依赖
go mod download

# 编译
go build -o gogw ./cmd/main.go

# 运行
./gogw --db=gogw.db --addr=:8080
```

### 默认配置
- 数据库路径: `gogw.db`
- 监听地址: `:8080`
- 默认用户: `admin`
- 默认密码: `123456`

## 配置说明

### 资源配置

支持两种类型的资源：

1. **串口资源**
   - 名称: 唯一标识
   - 类型: serial
   - 串口: 如 /dev/ttyUSB0
   - 波特率: 9600, 115200等
   - 数据位: 8
   - 停止位: 1
   - 校验位: N(无), O(奇), E(偶)

2. **网口资源**
   - 名称: 唯一标识
   - 类型: network
   - IP地址: 目标设备IP
   - 端口: 目标端口号
   - 协议: TCP/UDP

### 设备配置

设备关联资源和驱动：
- 名称: 设备唯一标识
- 资源: 关联的资源
- 驱动: 关联的驱动文件
- 采集周期: 毫秒
- 上传周期: 毫秒

### 北向配置

XunJi配置示例:
```json
{
    "productKey": "your-product-key",
    "deviceKey": "your-device-key",
    "serverUrl": "mqtt://broker.example.com",
    "username": "mqtt-username",
    "password": "mqtt-password"
}
```

### 阈值配置

阈值规则：
- 字段: 数据字段名
- 条件: >, <, >=, <=, ==, !=
- 阈值: 数值
- 严重程度: info, warning, error, critical

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
	
	// 读取数据
	data := readData()
	
	// 返回数据
	returnData(data)
}

func main() {
	// 注册函数
	// extism-pdk 特定代码
}
```

## API接口

### 认证接口
- `GET /login` - 登录页面
- `POST /login` - 登录提交
- `GET /logout` - 登出

### 状态接口
- `GET /api/status` - 系统状态

### 资源接口
- `GET /api/resources` - 获取资源列表
- `POST /api/resources` - 创建资源
- `PUT /api/resources/{id}` - 更新资源
- `DELETE /api/resources/{id}` - 删除资源
- `POST /api/resources/{id}/open` - 打开资源
- `POST /api/resources/{id}/close` - 关闭资源

### 设备接口
- `GET /api/devices` - 获取设备列表
- `POST /api/devices` - 创建设备
- `PUT /api/devices/{id}` - 更新设备
- `DELETE /api/devices/{id}` - 删除设备

### 驱动接口
- `GET /api/drivers` - 获取驱动列表
- `POST /api/drivers` - 创建驱动
- `PUT /api/drivers/{id}` - 更新驱动
- `DELETE /api/drivers/{id}` - 删除驱动

### 北向配置接口
- `GET /api/northbound` - 获取北向配置
- `POST /api/northbound` - 创建北向配置
- `PUT /api/northbound/{id}` - 更新北向配置
- `DELETE /api/northbound/{id}` - 删除北向配置

### 阈值接口
- `GET /api/thresholds` - 获取阈值列表
- `POST /api/thresholds` - 创建阈值
- `PUT /api/thresholds/{id}` - 更新阈值
- `DELETE /api/thresholds/{id}` - 删除阈值

### 报警接口
- `GET /api/alarms` - 获取报警日志
- `POST /api/alarms/{id}/acknowledge` - 确认报警

### 用户接口
- `GET /api/users` - 获取用户列表
- `POST /api/users` - 创建用户
- `PUT /api/users/{id}` - 更新用户
- `DELETE /api/users/{id}` - 删除用户
- `PUT /api/users/password` - 修改密码

## 项目结构

```
gogw/
├── cmd/
│   └── main.go           # 主程序入口
├── internal/
│   ├── auth/             # 认证模块
│   ├── collector/        # 数据采集模块
│   ├── database/         # 数据库操作
│   ├── driver/           # 驱动管理
│   ├── handlers/         # Web处理器
│   ├── models/           # 数据模型
│   ├── northbound/       # 北向接口
│   └── resource/         # 资源管理
├── migrations/
│   └── 001_init.sql      # 数据库迁移
├── web/
│   ├── pages/            # HTML页面
│   └── static/           # 静态资源
├── go.mod
└── README.md
```

## 许可证

MIT License
