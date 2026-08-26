# 半导体洁净室环境监控服务（cleanroom-environment-monitor-service）

## 一、项目概述

基于 Go 实现的半导体洁净室环境监控 Web 项目，一款后端服务，完成洁净室分区管理、粒子/温湿度采集、洁净度等级判定、设备联锁与报警处置。

项目类型：**全栈 Web 应用**（Go 后端服务 + `go:embed` 内嵌前端页面）。

## 二、业务背景与领域规则

半导体洁净室按 ISO 洁净度等级分区（如 ISO5/ISO6/ISO7），要求空气中粒径粒子浓度、温湿度、压差保持在规定范围。系统实时采集各分区的粒子计数器、温湿度、压差数据，判定当前洁净度等级；粒子浓度超限时自动联锁新风/FFU 设备加大过滤，并生成报警要求工艺人员处置。不同工艺区（光刻、刻蚀、扩散）阈值不同，设备 PM 维护会影响数据有效性。

关键领域规则（这些规则是后续埋 bug 验证跨文件改动的核心约束，必须真实实现）：

1. 洁净区状态机：正常运行(normal) → 粒子偏高(elevated) → 超限(over_limit) → 联锁通风(interlocked) → 恢复确认(restored)；超限 10 分钟未恢复自动进入联锁通风。
2. 洁净度判定：按分区 ISO 等级对照表，用当前粒子浓度（≥0.3μm 与 ≥0.5μm 双粒径）判定等级；任一粒径超标即取较差等级。
3. 数据有效性：设备处于 PM 维护中或标定过期时上报的数据标记为 invalid，不参与等级判定；无效数据占比超 30% 时产生「数据可信度不足」提示。
4. 联锁规则：粒子浓度 ≥ 上限 1.5 倍时自动下发 FFU 提速/新风加大指令，记录联锁日志；浓度回落后需人工确认恢复。
5. 报警去重与闭环：同分区同类型 20 分钟内重复报警合并；报警需工艺员确认并填处置说明，超 1 小时未确认升级。
6. 分区互斥：同一洁净区（物理区域）内的多个监测分区，联锁动作必须整区一致（一个分区触发，同区其他分区也进入联锁通风）。

## 三、核心实体（≥3 个，必须贯穿全栈）

每个实体必须贯穿「数据库/存储表 → domain model → repository → service → handler → 前端 API 层 → 前端页面/组件」全链路。

| 实体 | 关键字段 | 业务动作 |
|---|---|---|
| 洁净区 CleanZone | id、物理区域、ISO等级、工艺用途、状态 | 维护、状态查询 |
| 监测分区 MonitorZone | id、洁净区id、粒子计数器id、阈值配置 | 维护、判定 |
| 环境数据 EnvSample | id、监测分区id、粒径浓度、温湿度、压差、有效性、时间戳 | 采集、判定 |
| 联锁日志 InterlockLog | id、洁净区id、触发分区、动作、下发时间、恢复时间 | 联锁留痕 |
| 洁净报警 CleanAlert | id、监测分区id、类型、级别、状态、处置人、说明 | 生成、去重、处置 |

## 四、核心页面与 API

### 前端页面（≥4 个路由，至少 2 个页面共用同一个业务组件）

| 项目 | 说明 |
|---|---|
| / 洁净区总览 | 分区等级状态卡片 + 实时粒子值 + 未确认报警 | CleanZone、MonitorZone |
| /zones/{id} 分区详情 | 粒子趋势 + 等级判定明细 + 联锁记录 | MonitorZone、EnvSample |
| /alerts 报警台 | 报警列表 + 确认处置 + 升级 | CleanAlert |
| /interlocks 联锁记录 | 联锁动作时间线 | InterlockLog |
| /equipment 设备状态 | 粒子计数器/FFU 状态 + PM 维护标记 | MonitorZone |

### 后端 REST API（与页面一一对应，命中真实业务链路）

| 项目 | 说明 |
|---|---|
| POST /api/monitors/{id}/samples | 环境数据上报（有效性判定 + 等级判定） |
| POST /api/monitors/{id}/maintenance | 标记设备 PM 维护/结束 |
| POST /api/zones/{id}/interlock | 下发联锁动作（整区一致） |
| POST /api/zones/{id}/restore | 恢复确认 |
| GET /api/alerts | 报警列表 |
| POST /api/alerts/{id}/ack | 确认报警 |
| GET /api/zones/{id}/samples | 粒子趋势 |
| GET /api/interlocks | 联锁记录查询 |
| GET /api/overview | 总览聚合 |
| GET /api/healthz | 健康检查 |

## 五、横切关注点（≥2 个）

1. 操作审计日志：联锁下发、恢复确认、报警处置、PM 标记全部留痕；触达 handler → service → audit store。
2. 报警升级扫描定时任务：每 5 分钟扫描超 1 小时未确认报警并升级；触达 service → store → 报警台。
3. 全局错误处理与统一响应格式。

## 六、共享枚举/常量（≥2 组）

枚举/常量要求前后端各自定义且保持一致，README 中列出所有出现位置。

1. 洁净区状态 ZoneStatus：normal / elevated / over_limit / interlocked / restored。
2. ISO 等级 IsoClass：iso5 / iso6 / iso7 / iso8。
3. 报警类型 AlertType：particle / temp_humidity / pressure / data_quality。

## 七、共享前端组件与 hooks（组件 ≥3 个、hooks ≥2 个）

### 共享组件（放 `web/components/`）

1. ZoneCard：分区状态卡片，被总览与详情共用。
2. ParticleTrend：粒子趋势组件，被分区详情与报警台共用。
3. InterlockTimeline：联锁时间线，被联锁页与分区详情共用。

### 自定义 hooks（放 `web/hooks/`）

1. useZones(poll)：分区状态轮询，被总览与详情共用。
2. useAlerts(filter)：报警列表，被报警台与总览共用。

## 八、后端中间件（≥2 个）

1. auditLogger：审计日志中间件。
2. errorHandler：统一错误/panic 处理中间件。
3. requestID：trace id 注入中间件。

## 九、技术要求

- 语言：**Go 1.23**（go.mod 声明 `go 1.23`，module 路径 `example.com/cleanroom-environment-monitor-service`）
- 运行：`go run .` 默认监听 `8080`，支持 `PORT` 环境变量覆盖
- 存储：SQLite（`modernc.org/sqlite` 纯 Go 驱动，CGO 关闭）或内置内存仓储 + JSON 文件持久化，二选一，必须可重复构建、无外部服务依赖
- 前端：纯原生 HTML/CSS/JS，`go:embed` 内嵌 `web/` 静态资源，禁止引入外部 CDN 依赖（离线可跑）
- 服务入口：`GET /healthz` 返回 200；页面 `GET /` 可访问
- 根目录必须包含 `runtime_smoke.json`：`mode: service` + `start: go run .` + `ready_url: /healthz`；`project_intro` 一句话简介必须包含项目类型（如「基于 Go 实现的XXX Web 项目，一款后端服务，完成……」）
- 根目录必须包含 `README.md`：项目说明、目录结构、运行与测试命令、环境变量说明
- 构建：`go build ./...` 与 `go test ./...` 必须全部通过（基线干净、无 bug）

## 十、文件结构强制清单（规模目标：≥2000 行 Go 功能代码、≥20 个 `.go` 文件）

```
backend/
├── go.mod
├── main.go
├── config/
│   └── config.go            # ISO 阈值表、无效数据占比、升级时限
├── domain/
│   ├── cleanzone.go         # 洁净区 + 状态机
│   ├── monitor.go           # 监测分区
│   ├── sample.go            # 环境数据 + 有效性 + 等级判定
│   ├── interlock.go         # 联锁日志
│   └── alert.go             # 洁净报警 + 去重
├── store/
│   ├── cleanzone_store.go
│   ├── monitor_store.go
│   ├── sample_store.go
│   ├── interlock_store.go
│   ├── alert_store.go
│   └── audit_store.go
├── service/
│   ├── ingest_service.go    # 采集 + 有效性 + 等级判定
│   ├── interlock_service.go # 联锁下发（整区一致）
│   ├── alert_service.go     # 去重/确认/升级
│   ├── escalate_sweeper.go  # 升级扫描
│   └── audit_service.go
├── httpapi/
│   ├── router.go
│   ├── zone_handler.go
│   ├── monitor_handler.go
│   ├── interlock_handler.go
│   ├── alert_handler.go
│   └── health_handler.go
├── middleware/
│   ├── audit.go
│   ├── error_handler.go
│   └── request_id.go
└── web/
    ├── index.html
    ├── app.js
    ├── style.css
    ├── components/
    └── hooks/
```

**严禁合并职责到单一文件**：handler、service、repository、domain 必须分层；禁止把所有逻辑塞进 `main.go` 或一个 `handlers.go`。目标规模下限 2000 行 / 20 个 `.go` 文件，实际建议做到 3000 行以上 / 30 个文件以上，保证每个业务模块（实体、状态机、联动、报表）都有独立文件。

## 十一、运行、测试与交付要求

1. `go build ./...` 通过；`go test ./...` 全绿（含各业务模块的单元测试，测试文件不计入规模）。
2. `go run .` 后 `GET /healthz` 返回 200，前端页面 `GET /` 可打开且核心接口可用。
3. 每个核心业务动作都要有可复现的输入（API 请求/页面操作），方便后续构造缺陷与验证命令。
4. 代码中不得出现任何「故意埋错」「TODO bug」类注释；交付为干净基线。

## 十二、质量红线

1. **天然多文件、多层耦合**：任何一个小改动（如给某状态新增一个合法迁移）都应触达 3-5 个文件（domain + repository + service + handler + 前端组件 + 枚举定义）。
2. 业务规则必须具体、可验证：状态机迁移表、联动逻辑、校验边界、生命周期管理必须真实存在，禁止空壳 CRUD。
3. 本项目用于评测跨文件协同改动能力，禁止做成本目录、对账/财务、库存盘点、电商订单、预约挂号、工单客服、数据可视化报表类业务。
4. 前端页面必须真实消费后端接口，禁止纯静态假页面。

---
*生成说明：本提示词面向 Go 标注数据流水线 2000 行档位，主题已对照禁选题材清单核验。*
