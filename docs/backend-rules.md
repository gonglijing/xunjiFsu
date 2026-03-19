# 后端规则（Go 1.26 / RAM First）

本文档不是空谈式“理想架构”。它面向 FSU 当前后端，给出一套能直接落地的规则：目录怎么摆，文件怎么命名，函数怎么命名，什么能力优先用 Go 1.26 和标准库，什么写法必须因为内存成本而禁止。

## 1. 总原则

后端规则按以下优先级执行：

1. 少用 RAM。
2. 标准库优先。
3. Go 1.26 优先。
4. 边界清晰，热路径保守。
5. 命名优美、稳定、可长期扩展。

这里的“少用 RAM”不是一句口号，而是默认决策原则：

- 能用结构体，不用 `map[string]any`
- 能流式处理，不先全量读入
- 能复用切片，不重复分配
- 能在调用方提供缓冲区时完成，不在函数内部偷偷 new 大对象
- 能返回单值或迭代，不默认返回大切片
- 能在标准库里解决，不再引入新框架

## 2. Go 版本与能力基线

后端规则目标版本统一为 `Go 1.26.x`。

当前仓库的落地要求也统一到 `go.mod = 1.26.0`。新代码允许直接采用 Go 1.26 API；旧代码在被修改时一并向 1.26 风格收敛，不再继续按 1.24 心智写新逻辑。

### 2.1 必须优先使用的标准库能力

- HTTP：`net/http`
- 路由：优先 `http.ServeMux`；存量 `gorilla/mux` 可渐进迁移，不强制一次性替换
- 日志：`log/slog`
- 上下文：`context`
- 错误：`errors`, `fmt`, `errors.Join`
- 持久化入口：`database/sql`
- JSON：`encoding/json`
- 配置：`os`, `flag`, `encoding/json`, `time`
- 排序与集合：`slices`, `maps`, `cmp`
- 并发：`sync`, `sync/atomic`, `context`
- 观测：`expvar`, `runtime/metrics`, `net/http/pprof`

### 2.2 Go 1.26 可以优先引入的能力

以下能力适合新代码优先采用，但要遵守“可读性和内存优先”：

- `new(expr)`：只在表达“可选指针值”时更清爽时使用，不为炫技引入
- `bytes.Buffer.Peek`：适合协议解析、探测头部，避免额外复制
- `runtime/metrics` 新调度指标：优先用于采集器、northbound、driver 的运行时观测
- `testing.T.ArtifactDir()`：用于输出集成测试产物，不再把测试垃圾散落到仓库根目录
- `reflect` 的 iterator API：只允许用于工具层、元数据层，不允许进入热路径

### 2.3 实验能力规则

Go 1.26 的实验能力默认关闭，只允许按需、显式、可回退地使用：

- `runtime/pprof` 的 `goroutineleak` profile：仅用于排障或 CI 专项检查
- `runtime/secret`：仅用于秘密材料清理，不允许在普通业务逻辑里铺开
- `simd/archsimd`：默认禁用；只有基准证明收益明确且架构绑定可接受时才允许

## 3. 可落地的后端目录结构树

目标不是把当前仓库一次性推翻，而是让新代码和重构代码按下面的树落位。新功能默认按此结构实现；存量代码逐步迁移，不做“大爆炸式”重写。

```text
internal/
├── app/
│   ├── bootstrap.go            # 启动顺序、依赖装配
│   ├── server_http.go          # HTTP server 构造与启动
│   ├── routes_api.go           # API 路由挂载
│   ├── routes_page.go          # 页面/静态资源路由
│   └── runtime_apply.go        # 运行时配置应用入口
├── httpapi/
│   ├── middleware_auth.go      # 认证中间件
│   ├── middleware_gzip.go      # 压缩中间件
│   ├── response.go             # 响应封装
│   ├── decode_json.go          # 请求解码
│   ├── gateway_read.go         # 网关读接口
│   ├── gateway_write.go        # 网关写接口
│   ├── device_read.go          # 设备读接口
│   ├── device_write.go         # 设备写接口
│   ├── device_exec.go          # 设备执行接口
│   ├── driver_read.go          # 驱动读接口
│   ├── driver_write.go         # 驱动写接口
│   ├── north_read.go           # 北向读接口
│   └── north_write.go          # 北向写接口
├── service/
│   ├── gateway_service.go      # 业务编排
│   ├── device_service.go
│   ├── driver_service.go
│   ├── north_service.go
│   ├── runtime_service.go
│   └── alarm_service.go
├── store/
│   ├── db.go                   # DB 装配
│   ├── migration.go            # schema / migration
│   ├── gateway_store.go        # 网关持久化
│   ├── device_store.go         # 设备持久化
│   ├── driver_store.go         # 驱动持久化
│   ├── north_store.go          # 北向持久化
│   ├── point_store.go          # 测点数据
│   ├── alarm_store.go          # 告警数据
│   ├── runtime_audit_store.go  # 运行时配置审计
│   └── retention_job.go        # 保留清理
├── collect/
│   ├── scheduler.go            # 调度器
│   ├── job_heap.go             # 任务堆
│   ├── cycle_run.go            # 单轮采集
│   ├── threshold_eval.go       # 阈值评估
│   ├── command_poll.go         # 北向写命令轮询
│   ├── runtime_snapshot.go     # 运行态快照
│   └── memory_budget.go        # 采集链路内存预算
├── driver/
│   ├── loader.go               # 驱动加载
│   ├── catalog.go              # 驱动索引与元数据
│   ├── executor.go             # 执行器
│   ├── prepared_exec.go        # PreparedExecution
│   ├── transport_serial.go     # 串口访问
│   ├── transport_tcp.go        # TCP 访问
│   ├── wasm_runtime.go         # WASM runtime
│   └── result_decode.go        # 执行结果解析
├── north/
│   ├── registry.go             # 适配器注册
│   ├── manager.go              # 北向实例管理
│   ├── schema.go               # 类型 schema 入口
│   ├── mqtt_adapter.go
│   ├── pandax_adapter.go
│   ├── ithings_adapter.go
│   └── sagoo_adapter.go
├── model/
│   ├── gateway.go
│   ├── resource.go
│   ├── device.go
│   ├── driver.go
│   ├── northbound.go
│   ├── threshold.go
│   └── alarm.go
├── platform/
│   ├── auth/
│   ├── config/
│   ├── logger/
│   ├── clock/
│   └── graceful/
└── testkit/
    ├── httpcase.go
    ├── sqlitecase.go
    └── fixture.go
```

## 4. 分层边界

### 4.1 `httpapi`

只做五件事：

- 解析请求
- 做轻量校验
- 调用 `service`
- 写统一响应
- 把错误映射为 HTTP 语义

禁止：

- 直接拼复杂 SQL
- 直接调多个 `store` 做事务编排
- 直接维护运行时缓存
- 把采集、驱动、北向细节写在 handler 里

### 4.2 `service`

这是默认业务编排层。

只做四件事：

- 串联多个 `store`
- 驱动 `collect` / `driver` / `north`
- 维护业务级事务边界
- 明确更新内存态与持久态

禁止：

- 持有 HTTP 请求对象
- 输出 HTTP 响应结构
- 吞掉底层错误上下文

### 4.3 `store`

只负责数据读写与数据层约束。

必须：

- 面向明确模型返回
- 用 `database/sql` 风格组织查询
- 为热点查询提供固定 SQL 分支
- 明确参数库与数据库职责

禁止：

- 混入 HTTP、采集调度、MQTT 重连之类上层语义
- 返回“万能 map”
- 在查询函数里偷偷做跨领域拼装

### 4.4 `collect` / `driver` / `north`

这三个包属于运行时热区。

热区规则：

- 改动前先判断是否会进入每轮采集
- 默认先写基准，再谈抽象
- 不允许为了“统一风格”引入额外分配
- 不允许把反射、接口装箱、动态 map 扔进主路径

## 5. 文件名命名规范

### 5.1 文件名总规则

- 全部小写
- 只用 ASCII
- 单词之间用下划线 `_`
- 文件名必须体现“领域 + 职责”
- 禁止无意义名字

禁止的文件名：

- `common.go`
- `util.go`
- `utils.go`
- `helper.go`
- `misc.go`
- `temp.go`
- `base.go`
- `manager2.go`

推荐的文件名：

- `device_read.go`
- `device_write.go`
- `device_exec.go`
- `runtime_apply.go`
- `transport_serial.go`
- `result_decode.go`
- `threshold_eval.go`

### 5.2 文件名格式

统一采用以下格式之一：

- `<domain>_<action>.go`
- `<domain>_<role>.go`
- `<domain>_<role>_test.go`
- `<domain>_<role>_bench_test.go`

其中：

- `domain` 是业务名词，如 `device`、`gateway`、`north`
- `action` 是清晰动作，如 `read`、`write`、`exec`、`apply`
- `role` 是稳定职责，如 `store`、`adapter`、`scheduler`、`runtime`

### 5.3 角色后缀白名单

后缀尽量收敛到下面这些词，避免团队里每人发明一套：

- `api`
- `middleware`
- `service`
- `store`
- `scheduler`
- `runtime`
- `adapter`
- `schema`
- `policy`
- `cache`
- `decode`
- `encode`
- `transport`
- `audit`
- `bench_test`

### 5.4 现有仓库的落地策略

不要求一次性重命名全仓。

从现在开始：

- 新增文件必须遵守新规范
- 旧文件只要被大改，顺手改成新规范
- 一次 PR 内只做同一领域的小范围迁移，不做全仓改名

## 6. 函数名命名规范

### 6.1 总规则

- 导出函数：`VerbNoun`
- 包内函数：`verbNoun`
- 布尔判断：`isX` / `hasX` / `canX` / `shouldX`
- 构造函数：`NewType`
- 返回快照：`Snapshot`
- 返回视图：`View`
- 返回迭代：`Walk` / `Visit`
- 追加到缓冲区：`Append`

函数名要短，但不能空。

禁止：

- `Do`
- `Handle`
- `Process`
- `RunThing`
- `Helper`
- `Util`
- `Manage`
- `Exec1`

推荐：

- `LoadDevice`
- `ListDevices`
- `CreateDriver`
- `UpdateGateway`
- `ApplyRuntimeConfig`
- `ScheduleCollect`
- `RunCollectCycle`
- `DecodeDriverResult`
- `AppendMetricLine`
- `WalkLatestPoints`

### 6.2 动词白名单

读路径优先使用：

- `Get`：只用于“按唯一键直取单值”
- `Load`：需要加载依赖或额外开销
- `List`：返回列表
- `Scan`：逐行扫描或轻量读取
- `Walk`：回调式遍历，避免构造大切片
- `Visit`：访问者模式或遍历输出

写路径优先使用：

- `Create`
- `Update`
- `Replace`
- `Delete`
- `Patch`
- `Upsert`
- `Ack`

运行时优先使用：

- `Start`
- `Stop`
- `Reload`
- `Apply`
- `Schedule`
- `Dispatch`
- `Acquire`
- `Release`
- `Reset`

编解码优先使用：

- `Decode`
- `Encode`
- `Marshal`
- `Unmarshal`
- `Append`
- `WriteTo`

### 6.3 RAM First 命名约束

命名必须显式暴露“这段逻辑会不会分配”。

优先使用：

- `AppendX(dst []byte, ...) []byte`
- `ScanX(rows *sql.Rows) (...)`
- `WalkX(fn func(...) error) error`
- `WriteXTo(w io.Writer, ...) error`
- `FillX(dst *X, ...) error`

谨慎使用：

- `BuildX`
- `MakeX`
- `CollectX`

除非函数真的在“构造一个新对象”，否则不要用这些词，因为它们通常暗示分配。

### 6.4 参数命名规则

- `ctx context.Context` 必须是第一个参数
- `dst` 代表可复用目标缓冲区
- `src` 代表输入对象
- `id` 只用于单一主键
- `ids` 只用于主键集合
- `cfg` 只用于配置
- `opt` 只用于小型可选参数结构
- `buf` 只用于 `[]byte` 或 `bytes.Buffer`

禁止：

- `data interface{}`
- `obj any`
- `temp`
- `info`
- `ret`
- `res1`

### 6.5 当前仓库的函数命名映射示例

下面这些例子是给当前 FSU 仓库直接套用的。

- 读配置：`GetGatewayConfig` 优先收敛为 `LoadGateway`
- 列表查询：`GetAllDevices` 优先收敛为 `ListDevices`
- 单设备运行态：`GetDeviceRuntimeStatus` 优先收敛为 `LoadDeviceRuntime`
- 采集调度入口：`StartCollector` / `StopCollector` 可以保留，但内部轮次函数优先叫 `RunCollectCycle`
- 北向重载：`ReloadNorthbound` 比 `RefreshNorthboundManager` 更清晰
- 数据点最新值遍历：优先 `WalkLatestPoints`，而不是 `GetAllDevicesLatestData`
- 结果格式化：优先 `DecodeDriverResult` / `AppendDriverResultJSON`，避免 `FormatResult` 这类空泛名字

## 7. 面向少用 RAM 的编码规则

### 7.1 数据结构

- 热路径优先结构体，不用 `map[string]any`
- 固定字段优先显式字段，不做动态 key 组装
- 大对象优先按需字段，不默认整对象复制
- 查询结果优先预估容量：`make([]T, 0, n)`

### 7.2 数据流

- 能边读边写，就不要先读满
- 能按页返回，就不要一次性查全量
- 能走 callback，就不要先拼切片
- 能写入调用方缓冲区，就不要返回新缓冲区

### 7.3 JSON

- 外部协议仍用 `encoding/json`
- 内部热路径不要把结构体转 `map` 再转 JSON
- 不允许为了省几行代码引入反射型 JSON 包

### 7.4 SQL

- 坚持 `database/sql`
- 查询列显式列出，禁止热路径 `SELECT *`
- 大结果集优先 `rows.Next()` 流式处理
- 不允许仓促引入 ORM

### 7.5 缓冲与池

- 只对“大而短命”的缓冲区使用 `sync.Pool`
- 小对象池化前必须有基准
- 池里的对象必须可完全复位
- 不能把业务状态对象长期塞进池里

## 8. 标准库优先清单

新增依赖前，先按下面的顺序否决自己一次：

### 8.1 HTTP

- 先看 `net/http`
- 再看是否真的需要第三方 router

### 8.2 日志

- 先用 `log/slog`
- 不再新增其他日志框架

### 8.3 重试、超时、取消

- 先用 `context`
- 重试策略用小函数本地实现
- 不要引入“万能 resilience 框架”

### 8.4 数据处理

- 排序/裁剪/比较先用 `slices` / `maps` / `cmp`
- 缓冲区先用 `bytes`
- 文本拼接先用 `strings.Builder` 或 `bytes.Buffer`

## 9. 评审硬规则

出现以下任一情况，默认不合并：

- 新代码继续往 `handlers` 里塞业务编排
- 新函数名含糊到读不出职责
- 新文件名是 `utils` / `common` / `helper`
- 热路径引入 `map[string]any`、反射或额外 JSON 往返
- 查询路径无上限地构造大切片
- 为了统一抽象牺牲内存占用
- 引入第三方库但标准库足够

## 10. 迁移顺序建议

按下面顺序落地，成本最低：

1. 新增代码先遵守文件名和函数名规则。
2. `handlers` 新逻辑先下沉到 `service`。
3. `database` 新查询按 `store` 风格命名。
4. 热路径函数逐步改成 `Append` / `Walk` / `Scan` 风格。
5. 路由层在触达时逐步向 `net/http` / `ServeMux` 收敛。

## 11. 从当前仓库到目标结构的映射

这一步是给当前 FSU 仓库直接执行的，不是抽象建议。

### 11.1 包级迁移映射

- `internal/handlers` 逐步迁到 `internal/httpapi`
- `internal/database` 逐步迁到 `internal/store`
- `internal/models` 逐步迁到 `internal/model`
- `internal/collector` 可逐步收敛命名到 `internal/collect`
- `internal/northbound` 可逐步收敛命名到 `internal/north`
- `internal/auth`、`internal/config`、`internal/logger`、`internal/graceful` 逐步收拢到 `internal/platform/*`

### 11.2 不应立即迁动的热点文件

以下文件在没有配套测试和基准前，不做“纯命名式迁移”：

- `internal/collector/collector.go`
- `internal/collector/modbus_collect.go`
- `internal/driver/executor.go`
- `internal/driver/manager.go`
- `internal/database/data_points.go`
- `internal/database/data_cache.go`

这些文件先做“职责收敛”，再做“包路径迁移”。顺序反了，风险会很高。

### 11.3 新增文件的首选命名

如果你今天继续往现有目录加文件，先按新命名法写，不必等全量迁移完成。

- 在 `internal/handlers` 中新增文件时，优先使用 `device_read.go`、`device_write.go`、`gateway_read.go` 这种名字
- 在 `internal/database` 中新增文件时，优先使用 `device_store.go`、`alarm_store.go`、`point_store.go`
- 在 `internal/collector` 中新增文件时，优先使用 `scheduler.go`、`cycle_run.go`、`runtime_snapshot.go`
- 在 `internal/northbound/adapters` 中新增文件时，优先使用 `<type>_adapter.go`、`<type>_runtime.go`、`<type>_schema.go`

## 12. 评审检查清单

- 这个函数名是否一眼能看出动作和对象？
- 这个文件名是否体现了领域和职责？
- 这段逻辑是否真的需要新分配？
- 这条数据流能否改成流式或 callback？
- 这处能力是否能直接用标准库完成？
- 这段代码是否适合放在热路径？
- 这次改动是否让运行态与持久态保持一致？

如果以上任何一项答案不清楚，这段代码就还没写完。
